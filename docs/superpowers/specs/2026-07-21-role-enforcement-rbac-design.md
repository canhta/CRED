# Role enforcement + RBAC/ACL integration — design

- **Date:** 2026-07-21
- **Status:** Approved by operator, pending implementation plan.
- **Decision:** `docs/research/decision-log.md` D-025 — multi-user auth and
  RBAC pulled into v1. This spec is the second of four sub-projects D-025
  names: identity & auth
  (`docs/superpowers/specs/2026-07-21-identity-auth-design.md`) → role
  model + RBAC/ACL integration (this doc) → invite flow → MCP auth. Only
  sub-project 2 is designed here.

## Problem

Sub-project 1 activated the `principals` table with real accounts and
sessions, and resolves a role (`admin`/`member`) into the request context
whenever a session cookie is present (`internal/api/api.go:84-123`,
`roleKey{}` at line 67). No accessor function reads it, and no route checks
it — every authenticated request today reaches every route regardless of
role. `GET /api/usage` (`internal/api/usage.go:18-62`) is the one concrete
consequence: it mixes the caller's own limit headroom (`PrincipalWindowState`,
`DeniedInWindow` — both scoped to the calling principal) with an org-wide
cost/growth-by-scope report (`UsageByScope`, `ScopeSizes` — both take no
principal argument at all, `internal/store/pg/usage.go:192,220`), and any
authenticated caller sees both today.

Two more findings shape this design:

- `internal/store/migrations/00007_one_admin.sql` (already shipped, part of
  the admin-registration race fix) adds a partial unique index enforcing at
  most one admin row. Admin is a singleton in the current schema — a real,
  current constraint this design has to respect, not something it introduces.
- `roleKey{}` is populated only on the session-cookie auth path
  (`internal/api/api.go:101-110`). The header (`X-CRED-Principal`) and
  config-default (`cfg.Principal`) paths — used by the CLI, scripts, and
  eventually MCP — never resolve a role. Any route guard built only against
  the session path would leave every non-browser caller either unrestricted
  or, if the guard defaults to deny, locked out of admin routes it should be
  able to reach.

## Prior art consulted

`docs/research/evidence/mem0.md` documents OpenMemory's `AccessControl`
table as dormant by construction: an evaluator exists, but
`grep -rn "AccessControl(" openmemory/` finds no writer anywhere. That
pattern — an enforcement mechanism with zero call sites — is the risk this
design avoids by giving the guard a real, current consumer (`/api/usage`'s
org-wide section) rather than shipping the mechanism alone.

A second, more directly useful precedent came from mem0's newer `server/`
tree (a FastAPI server distinct from `openmemory/`, not covered by the
existing evidence file): `server/auth.py:198-219` implements `require_admin`
as a dependency wrapping `verify_auth`, applied to whole routes via FastAPI's
`Depends()` (`server/main.py:332`, `547`; `server/routers/entities.py:71`) —
raising 403 on `role != "admin"`. Its migration
`server/alembic/versions/004_unique_admin_role.py` independently enforces the
same "at most one admin" constraint CRED's own `00007_one_admin.sql` does.
(Read directly from the local clone at
`/Users/canh/Solo/OSS/mem0`, 2026-07-21; not the version pinned in the
project's own evidence file.)

This matches the industry standard: OWASP API5:2023, Broken Function Level
Authorization, uses the same shape as its worked example — a separate
admin-only endpoint (`GET /api/admin/v1/users/all`) rather than a role
branch inside a shared handler — and its stated mitigation is "make sure all
administrative controllers inherit from an administrative abstract
controller that implements authorization checks based on role"
(<https://owasp.org/API-Security/editions/2023/en/0xa5-broken-function-level-authorization/>,
fetched 2026-07-21). A centralized guard a route is registered under, not a
conditional a handler body could later drop, is also consistent with this
codebase's own `internal/acl` precedent: `Permitted` fails closed by
construction, not by a check a caller could forget
(`internal/acl/acl.go:66-68`).

## Architecture: role gates routes, ACL gates content

Two axes, deliberately not merged:

- **`internal/acl`** answers "may this principal read this claim?" —
  evaluated per-claim, per-request, as the intersection over evidence ACLs
  (L5). Unchanged by this sub-project.
- **Role** answers "may this principal call this console/API route?" — a
  coarse `admin`/`member` binary attached to the request context, checked
  once per route by a guard, before any handler reaches `internal/acl`.

Admin role never bypasses claim ACL. A caller with `role=admin` hitting
`GET /api/claims` still only sees what `acl.Filter` permits for their
principal, exactly as today. Role and ACL run in different packages, at
different points in the request, and neither can see or override the
other's decision — that separation is what keeps this sub-project from
duplicating or conflicting with the existing model, structurally rather than
by convention.

## Store layer

One new method, `internal/store/pg/identity.go`:

```go
// RoleForPrincipal resolves a principal's console role. A scalar subquery,
// not a plain SELECT, so a principal with no user_credentials row (a
// team/org/agent principal, or an automation identity never registered as a
// user) resolves to "" rather than ErrNotFound -- having no console account
// is the normal case for most principals, not a failure.
func (s *Store) RoleForPrincipal(ctx context.Context, principal claim.PrincipalID) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT coalesce((SELECT role FROM user_credentials WHERE principal_id = $1), '')`,
		string(principal)).Scan(&role)
	return role, translate(err)
}
```

No new migration. `role` already lives on `user_credentials`
(`00006_identity.sql`); singleton-admin is already enforced
(`00007_one_admin.sql`).

## Backend mechanism

`authenticate()` (`internal/api/api.go:84-123`) resolves role uniformly on
both paths, not only the session path:

- Session path: unchanged — already gets role free from `SessionPrincipal`.
- Header/default-principal path: gains one call to `RoleForPrincipal`,
  attaching `roleKey{}` exactly like the session branch does today. A
  header-authenticated admin is still an admin; a principal with no console
  account resolves to `""`, which fails every admin check by construction.

New accessor, alongside `principalFrom`:

```go
func roleFrom(c *gin.Context) string {
	if r, ok := c.Request.Context().Value(roleKey{}).(string); ok {
		return r
	}
	return ""
}
```

New guard middleware:

```go
// requireAdmin aborts with 403 unless the resolved principal's role is
// "admin". Applied to a route group, never inlined in a handler -- a
// centralized check every admin route inherits, not a conditional a future
// edit could silently drop.
func requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if roleFrom(c) != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "admin role required"})
			return
		}
		c.Next()
	}
}
```

## API surface

`internal/api/types.go`:

```go
// UsageResponse is the body of GET /api/usage: the calling principal's own
// limit headroom and denied-contribution count. Org-wide data moved to
// OrgUsageResponse -- a member's own view carries no other principal's data.
type UsageResponse struct {
	Principal          string      `json:"principal"`
	Contribution       LimitStatus `json:"contribution"`
	Cost               LimitStatus `json:"cost"`
	InputTokensUsed    int         `json:"input_tokens_used"`
	InputTokensCeiling int         `json:"input_tokens_ceiling"`
	Recall             LimitStatus `json:"recall"`
	DeniedWindow       string      `json:"denied_window"`
	Denied             int         `json:"denied"`
}

// OrgUsageResponse is the body of GET /api/usage/org: cost and growth by
// scope across every principal -- "which teams actually use this", visible
// to admins only.
type OrgUsageResponse struct {
	CostByScope []ScopeCost   `json:"cost_by_scope"`
	ScopeGrowth []ScopeGrowth `json:"scope_growth"`
}
```

`HealthResponse` gains `Role string`, populated from `roleFrom(c)` in the
health handler — the frontend's one source of truth for what to show.

`internal/api/usage.go`: `usage()` drops the `UsageByScope`/`ScopeSizes`
calls into a new `usageOrg()` handler. The queries themselves are unchanged;
only which handler calls them, and which route reaches that handler,
changes.

`internal/api/api.go` route registration:

```go
api.GET("/usage", s.usage)
// ...
admin := api.Group("")
admin.Use(requireAdmin())
admin.GET("/usage/org", s.usageOrg)
```

## Frontend

- `tygo` regenerates `Health`, `UsageResponse`, and the new
  `OrgUsageResponse` into `web/src/api/types.ts` automatically — no
  hand-written type mirrors, per `web.md` §3.
- New `getUsageOrg()` in `client.ts`, a new route entry in `routes.ts`, and a
  new `useUsageOrg()` hook in `hooks.ts` with
  `enabled: health.data?.role === 'admin'` — a member's browser never issues
  a request that would 403.
- `UsagePage.tsx`: reads `role` from the `useHealth()` call already in
  scope. The cost-by-scope and scope-growth tables render only when
  `role === 'admin'`, sourced from `useUsageOrg()` instead of the single
  `useUsage()` response.

## What's deliberately out of scope for this pass

- **Team and Settings pages.** Still placeholders with no real data —
  nothing to gate yet. Fleshing them out (member list, invites) is
  sub-project 3.
- **A second admin.** `user_credentials_one_admin` (`00007_one_admin.sql`)
  enforces exactly one admin row at the database level; nothing in this pass
  changes that. Named here as a real, current limitation feeding
  sub-project 3, not hidden.
- **MCP picking up role.** `internal/mcpsrv` has no auth surface yet at all
  (sub-project 4); this pass only touches the console API.
- **Role bypassing claim ACL.** Explicitly rejected above — admin is a
  console/administrative capability, never a content-access override.
- **Any admin-gated route beyond `/api/usage/org`.** This pass builds the
  general mechanism (`requireAdmin`, `roleFrom`, uniform role resolution in
  `authenticate()`) and applies it to the one real, currently-unrestricted
  surface. Future admin routes reuse the mechanism without new design.

## Testing

Follows sub-project 1's established pattern — no per-handler Go test file
for the HTTP layer, real unit tests where pure logic exists:

- `internal/store/pg/identity_test.go` (or extending the existing
  integration-test pattern): `RoleForPrincipal` against a real Postgres — a
  `user_credentials` row resolves its role, a principal with no row resolves
  `""`, a non-`'user'`-kind principal (team/org/agent) resolves `""`.
- Manual end-to-end verification (real Postgres, live browser): log in as
  the bootstrap admin, confirm the org-wide usage panel renders; set a
  second test principal's role to `'member'` directly in the database (the
  only way to get a second, non-admin account until sub-project 3's invite
  flow ships) and confirm `GET /api/usage/org` returns 403 and the panel
  does not render.
- Frontend: extend `UsagePage.test.tsx`'s existing pattern — mock
  `useHealth` to return each role, assert the org-wide tables render for
  admin and are absent for member.
