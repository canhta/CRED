# Role Enforcement + RBAC/ACL Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Give the dormant `roleKey{}` context value a real accessor and a route-group guard, and apply it to the one currently-unrestricted admin-shaped surface: the org-wide cost/growth section of `/api/usage`, split into a new admin-only route.

**Architecture:** `authenticate()` resolves role uniformly on both auth paths (session gets it free from `SessionPrincipal`; the header/default-principal path gains one new store lookup, `RoleForPrincipal`). A new `roleFrom(c)` accessor and `requireAdmin()` middleware follow the same shape as `principalFrom`/`authenticate`. `GET /api/usage` splits into two routes: the existing one keeps the caller's own headroom, and a new `GET /api/usage/org` (behind `requireAdmin()`) carries the org-wide cost/growth report. The frontend adds `role` to `HealthResponse`, adds a `useUsageOrg` hook gated by an external `enabled` flag (the same shape `useRecall` already uses), and `UsagePage` renders the org-wide tables only for an admin.

**Tech Stack:** Go 1.25, Gin, pgx · React 19, TanStack Query, Astryx, Vitest.

## Global Constraints

- `CGO_ENABLED=0` for every Go build.
- No hand-written TypeScript type may mirror a Go struct — regenerate with `task gen:types`.
- No `<div>` for layout in React — compose Astryx components.
- Every API route lives once in `web/src/api/routes.ts`.
- Handlers read the principal via `principalFrom(c)`, the role via `roleFrom(c)` — never a header, cookie, or context key directly.
- Comments are sparse, explain the *why*, never cite a decision number or law number.
- Role never bypasses claim ACL. `internal/acl` is untouched by this plan; no handler skips `acl.Filter`/`acl.CanRead` because the caller is an admin.
- This codebase's `internal/api` precedent is **no per-handler Go test file** — frontend tests against a mocked client plus manual end-to-end verification. This plan follows that precedent.
- Out of scope (see `docs/superpowers/specs/2026-07-21-role-enforcement-rbac-design.md`, "What's deliberately out of scope"): Team/Settings pages, a second admin, MCP picking up role, any admin-gated route beyond `/api/usage/org`.

**Corrections made while planning** (the spec described the shape; these are the exact resolutions found while turning it into code):

- The spec didn't address what happens to `UsageQuery`'s `Scopes` field (the `?scopes=` param that sizes `ScopeSizes`'s top-N) once the org-wide data moves to its own route. It belongs entirely to the new route now — the plain `/api/usage` handler has no query parameters left at all after the split. `UsageQuery` is renamed `OrgUsageQuery` and its binding moves into `usageOrg()`.
- Following that, the frontend's `UsageParams`/`queryKeys.usage(params)` (a function taking an unused `{}` everywhere it's called) is replaced with a plain `queryKeys.usage` constant, and a new `OrgUsageParams`/`queryKeys.usageOrg(params)` takes over the `scopes` param. `queryKeys.usageOrg` nests under the same `['usage', ...]` prefix as `queryKeys.usage`, so the existing "Refresh" button's single `invalidateQueries({ queryKey: queryKeys.usage })` call still refreshes both queries — TanStack Query's invalidation matches by key prefix.
- `useUsageOrg` takes an external `enabled: boolean` rather than calling `useHealth()` itself, matching `useRecall(params, enabled)`'s existing shape — the hook stays decoupled from where the gating condition comes from; `UsagePage` computes `isAdmin` from its own `useHealth()` call and passes it in.
- `authenticate()`'s new `RoleForPrincipal` call, on the header/default-principal path, ignores a lookup error and falls back to `role = ""` rather than aborting the request with 500. This mirrors the existing session-cookie branch immediately above it, which already falls through silently on any `SessionPrincipal` error. The alternative — hard-failing every header/token-authenticated request on a role-lookup error — would turn an availability property that doesn't exist today (every request in this path already reaches a handler that hits the same pool) into a new single point of failure inside the auth middleware itself, for a check that only gates admin routes.
- `HealthResponse.Role` becomes a required (non-optional) generated field once added to the Go struct, which breaks TypeScript compilation of two existing test files that construct full `Health` object literals without it: `web/src/router.test.tsx` (`AUTHENTICATED`, and one inline literal in the "redirects to login when principal is empty" test) and `web/src/pages/Register.test.tsx` (`OPEN`/`CLOSED`). Both get a `role` field added in Task 3 — not optional, since the type change itself is what breaks them.
- `roleKey{}`'s doc comment (`internal/api/api.go`) currently says "Attached now so a later route guard can read it without a second query; nothing reads it yet." That's stale the moment `roleFrom` exists; Task 2 updates it.

---

### Task 1: `internal/store/pg` — `RoleForPrincipal`

**Files:**
- Modify: `internal/store/pg/identity.go`
- Modify: `internal/store/pg/integration_test.go`

**Interfaces:**
- Produces: `func (s *Store) RoleForPrincipal(ctx context.Context, principal claim.PrincipalID) (string, error)`. Task 2 calls this exact signature.

- [ ] **Step 1: Write the failing test**

In `internal/store/pg/integration_test.go`, add after `TestCreateUserSecondAdminReturnsErrAdminExists` (the file's last test function):

```go

// TestRoleForPrincipalResolvesRealRoleOrEmpty covers both callers this
// backs: authenticate()'s session path already gets role from
// SessionPrincipal, but the header/default-principal path calls this
// directly. A principal with a user_credentials row resolves its real role;
// a principal with none (a team/org/agent principal, or a header/default
// value with no console account) resolves "" rather than an error -- having
// no console account is the normal case for most principals, not a failure.
func TestRoleForPrincipalResolvesRealRoleOrEmpty(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()

	principal, err := st.CreateUser(ctx, uniqueEmail(t, "roleforprincipal"), "hash", "member")
	require.NoError(t, err)

	role, err := st.RoleForPrincipal(ctx, principal)
	require.NoError(t, err)
	require.Equal(t, "member", role)

	role, err = st.RoleForPrincipal(ctx, claim.PrincipalID("no-such-principal"))
	require.NoError(t, err)
	require.Empty(t, role)
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `docker compose up -d db` (if not already running)
Run: `go test -tags=integration ./internal/store/pg/... -run TestRoleForPrincipalResolvesRealRoleOrEmpty -v`
Expected: FAIL — `st.RoleForPrincipal undefined`.

- [ ] **Step 3: Write the method**

In `internal/store/pg/identity.go`, append to the end of the file:

```go

// RoleForPrincipal resolves a principal's console role. A scalar subquery,
// not a plain SELECT, so a principal with no user_credentials row resolves
// to "" rather than ErrNotFound -- having no console account is the normal
// case for most principals, not a failure.
func (s *Store) RoleForPrincipal(ctx context.Context, principal claim.PrincipalID) (string, error) {
	var role string
	err := s.pool.QueryRow(ctx, `
		SELECT coalesce((SELECT role FROM user_credentials WHERE principal_id = $1), '')`,
		string(principal)).Scan(&role)
	return role, translate(err)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test -tags=integration ./internal/store/pg/... -run TestRoleForPrincipalResolvesRealRoleOrEmpty -v`
Expected: PASS.

Run: `go test -tags=integration -shuffle=on -p 1 ./internal/store/pg/...`
Expected: `ok` — the full integration suite still passes, unaffected by this addition.

- [ ] **Step 5: Build, vet, lint**

Run: `task api:build`
Expected: exits 0.

Run: `go vet ./...`
Expected: exits 0.

Run: `task api:lint`
Expected: exits 0. `RoleForPrincipal` is exported, so it is not flagged as unused even though nothing calls it until Task 2.

- [ ] **Step 6: Commit**

```bash
git add internal/store/pg/identity.go internal/store/pg/integration_test.go
git commit -m "feat: add RoleForPrincipal to internal/store/pg

Resolves a principal's console role for the header/default-principal
auth path, which (unlike the session path) has no role attached yet.
A principal with no user_credentials row resolves to \"\" rather than
an error."
```

---

### Task 2: `internal/api` — role enforcement mechanism and the usage split

**Files:**
- Modify: `internal/api/types.go`
- Modify: `internal/api/api.go`
- Modify: `internal/api/usage.go`

**Interfaces:**
- Consumes: Task 1's `(*pg.Store).RoleForPrincipal`.
- Produces: `roleFrom(c *gin.Context) string`, `requireAdmin() gin.HandlerFunc`, `GET /api/usage/org` (admin-only), `HealthResponse.Role`, trimmed `UsageResponse` (no `CostByScope`/`ScopeGrowth`), new `OrgUsageResponse{CostByScope, ScopeGrowth}`, `OrgUsageQuery{Scopes}`. Task 3 consumes these exact JSON field names and the route — do not rename.

This task has no dedicated Go test file, following this codebase's `internal/api` precedent (see Global Constraints). It's verified by build/vet/lint and, at the end of the plan, manual end-to-end verification (Task 5).

- [ ] **Step 1: Update the wire types**

In `internal/api/types.go`, change:

```go
// HealthResponse reports liveness and the resolved caller identity, so the
// console can show who it is acting as before any data loads.
type HealthResponse struct {
	Status           string `json:"status"`
	Version          string `json:"version"`
	Principal        string `json:"principal"`
	RegistrationOpen bool   `json:"registration_open"`
}
```

to:

```go
// HealthResponse reports liveness and the resolved caller identity, so the
// console can show who it is acting as before any data loads.
type HealthResponse struct {
	Status           string `json:"status"`
	Version          string `json:"version"`
	Principal        string `json:"principal"`
	Role             string `json:"role"`
	RegistrationOpen bool   `json:"registration_open"`
}
```

Change:

```go
// UsageQuery is the query string of GET /api/usage.
type UsageQuery struct {
	Scopes int `json:"scopes" form:"scopes"`
}
```

to:

```go
// OrgUsageQuery is the query string of GET /api/usage/org.
type OrgUsageQuery struct {
	Scopes int `json:"scopes" form:"scopes"`
}
```

Change:

```go
// UsageResponse is the body of GET /api/usage: the calling principal's limit
// headroom, its denied-contribution count, and the org-wide cost/growth
// report — the same counters and the same internal/limit decisions
// `cred usage` prints, so the console never shows a number the enforcement
// path didn't also compute.
type UsageResponse struct {
	Principal          string        `json:"principal"`
	Contribution       LimitStatus   `json:"contribution"`
	Cost               LimitStatus   `json:"cost"`
	InputTokensUsed    int           `json:"input_tokens_used"`
	InputTokensCeiling int           `json:"input_tokens_ceiling"`
	Recall             LimitStatus   `json:"recall"`
	DeniedWindow       string        `json:"denied_window"`
	Denied             int           `json:"denied"`
	CostByScope        []ScopeCost   `json:"cost_by_scope"`
	ScopeGrowth        []ScopeGrowth `json:"scope_growth"`
}
```

to:

```go
// UsageResponse is the body of GET /api/usage: the calling principal's own
// limit headroom and denied-contribution count — the same counters and the
// same internal/limit decisions `cred usage` prints, so the console never
// shows a number the enforcement path didn't also compute. Org-wide data
// lives in OrgUsageResponse: a member's own view carries no other
// principal's data.
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

- [ ] **Step 2: Split `usage.go` into the member-scoped and org-wide handlers**

Replace the full contents of `internal/api/usage.go` with:

```go
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/canhta/cred/internal/limit"
	"github.com/canhta/cred/internal/store/pg"
)

// usage reports the calling principal's own limit headroom and denied-
// contribution count. Every number comes from the same store counts and the
// same internal/limit decisions the enforcement path uses, so the console
// can never show a number that disagrees with what a write or a recall was
// actually allowed to do.
func (s *server) usage(c *gin.Context) {
	ctx := c.Request.Context()
	principal := principalFrom(c)
	now := time.Now().UTC()
	lc := s.cfg.Limits

	state, err := s.store.PrincipalWindowState(ctx, principal,
		limit.WindowStart(now, lc.ContributionWindow),
		limit.WindowStart(now, lc.CostWindow),
		limit.WindowStart(now, lc.RecallWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	denied, err := s.store.DeniedInWindow(ctx, principal, limit.WindowStart(now, lc.ContributionWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, usageResponse(string(principal), lc, state, denied))
}

func usageResponse(principal string, lc limit.Config, state pg.ContributionState, denied int) UsageResponse {
	contribution := limit.Contribution(state.Contributions, lc)
	cost := limit.Cost(state.InferenceCall, state.InputTokens, lc)
	recall := limit.RecallRate(state.Recalls, lc)

	return UsageResponse{
		Principal: principal,
		Contribution: LimitStatus{
			Window: lc.ContributionWindow.String(), Used: state.Contributions,
			Ceiling: lc.ContributionQuota, Remaining: contribution.Remaining,
			Allowed: contribution.Allowed, Reason: string(contribution.Reason),
		},
		Cost: LimitStatus{
			Window: lc.CostWindow.String(), Used: state.InferenceCall,
			Ceiling: lc.MaxInferenceCalls, Remaining: cost.Remaining,
			Allowed: cost.Allowed, Reason: string(cost.Reason),
		},
		InputTokensUsed:    state.InputTokens,
		InputTokensCeiling: lc.MaxInputTokens,
		Recall: LimitStatus{
			Window: lc.RecallWindow.String(), Used: state.Recalls,
			Ceiling: lc.RecallRate, Remaining: recall.Remaining,
			Allowed: recall.Allowed, Reason: string(recall.Reason),
		},
		DeniedWindow: lc.ContributionWindow.String(),
		Denied:       denied,
	}
}

// usageOrg reports cost and growth by scope across every principal --
// "which teams actually use this". Gated by requireAdmin() at the route
// level (see api.go); this handler assumes that check already ran.
func (s *server) usageOrg(c *gin.Context) {
	var q OrgUsageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid query parameters"})
		return
	}
	topN := q.Scopes
	if topN <= 0 {
		topN = 10
	}

	ctx := c.Request.Context()
	now := time.Now().UTC()
	lc := s.cfg.Limits

	costs, err := s.store.UsageByScope(ctx, limit.WindowStart(now, lc.CostWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	sizes, err := s.store.ScopeSizes(ctx, topN)
	if err != nil {
		s.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, orgUsageResponse(lc, costs, sizes))
}

func orgUsageResponse(lc limit.Config, costs []pg.ScopeCost, sizes []pg.ScopeSize) OrgUsageResponse {
	costByScope := make([]ScopeCost, 0, len(costs))
	for _, c := range costs {
		costByScope = append(costByScope, ScopeCost{
			Scope:        Scope{Kind: string(c.Scope.Kind), Value: c.Scope.Value},
			Calls:        c.Calls,
			InputTokens:  c.InputTokens,
			OutputTokens: c.OutputTokens,
		})
	}

	scopeGrowth := make([]ScopeGrowth, 0, len(sizes))
	for _, sz := range sizes {
		scopeGrowth = append(scopeGrowth, ScopeGrowth{
			Scope:     Scope{Kind: string(sz.Scope.Kind), Value: sz.Scope.Value},
			Live:      sz.Live,
			Ceiling:   lc.ScopeClaimCeiling,
			NextPrune: limit.PruneTarget(sz.Live, lc),
		})
	}

	return OrgUsageResponse{
		CostByScope: costByScope,
		ScopeGrowth: scopeGrowth,
	}
}
```

- [ ] **Step 3: Add `roleFrom` and `requireAdmin`, fix the `roleKey` comment, resolve role on the header/default path, and wire the new route**

In `internal/api/api.go`, change:

```go
// principalFrom returns the principal the auth middleware resolved. Handlers
// read the caller's identity through here, never from the request header.
func principalFrom(c *gin.Context) claim.PrincipalID {
	if p, ok := c.Request.Context().Value(principalKey{}).(claim.PrincipalID); ok {
		return p
	}
	return ""
}
```

to:

```go
// principalFrom returns the principal the auth middleware resolved. Handlers
// read the caller's identity through here, never from the request header.
func principalFrom(c *gin.Context) claim.PrincipalID {
	if p, ok := c.Request.Context().Value(principalKey{}).(claim.PrincipalID); ok {
		return p
	}
	return ""
}

// roleFrom returns the resolved principal's role ("admin" or "member"), or
// "" when none is known -- a header/default-principal caller with no
// console account resolves the same way as an unauthenticated one. Empty
// never passes requireAdmin, so "role unknown" and "role member" fail
// identically.
func roleFrom(c *gin.Context) string {
	if r, ok := c.Request.Context().Value(roleKey{}).(string); ok {
		return r
	}
	return ""
}
```

Change:

```go
// roleKey is the context key for the resolved session's role, when a session
// (not a header) supplied the principal. Attached now so a later route guard
// can read it without a second query; nothing reads it yet.
type roleKey struct{}
```

to:

```go
// roleKey is the context key for the resolved principal's role, attached by
// authenticate() on every path -- session and header/default alike -- and
// read by roleFrom.
type roleKey struct{}
```

Change `authenticate`'s header/default branch:

```go
		// The header names who the request acts as; absent, the single-user
		// self-host identity from config stands in.
		principal := claim.PrincipalID(cfg.Principal)
		if h := c.GetHeader("X-CRED-Principal"); h != "" {
			principal = claim.PrincipalID(h)
		}

		ctx := context.WithValue(c.Request.Context(), principalKey{}, principal)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
```

to:

```go
		// The header names who the request acts as; absent, the single-user
		// self-host identity from config stands in.
		principal := claim.PrincipalID(cfg.Principal)
		if h := c.GetHeader("X-CRED-Principal"); h != "" {
			principal = claim.PrincipalID(h)
		}

		// A header/default-authenticated caller still gets its real console
		// role, not an assumed one -- an admin using the CLI or a bearer
		// token is still an admin. A lookup error is treated the same as no
		// role found, matching the tolerant fall-through the session branch
		// above already uses for its own error case.
		role, _ := store.RoleForPrincipal(c.Request.Context(), principal)

		ctx := context.WithValue(c.Request.Context(), principalKey{}, principal)
		ctx = context.WithValue(ctx, roleKey{}, role)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
```

Add `requireAdmin`, after `authenticate`'s closing brace and before `requestLogger`:

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

Change `New`'s route registration:

```go
	s := &server{store: st, embedder: emb, count: count, cfg: cfg, log: log}
	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		api.GET("/claims", s.listClaims)
		api.GET("/claims/:id", s.getClaim)
		api.GET("/recall", s.recall)
		api.GET("/usage", s.usage)
		api.POST("/auth/register", s.register)
		api.POST("/auth/login", s.login)
		api.POST("/auth/logout", s.logout)
	}
	return r
}
```

to:

```go
	s := &server{store: st, embedder: emb, count: count, cfg: cfg, log: log}
	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		api.GET("/claims", s.listClaims)
		api.GET("/claims/:id", s.getClaim)
		api.GET("/recall", s.recall)
		api.GET("/usage", s.usage)
		api.POST("/auth/register", s.register)
		api.POST("/auth/login", s.login)
		api.POST("/auth/logout", s.logout)

		admin := api.Group("")
		admin.Use(requireAdmin())
		admin.GET("/usage/org", s.usageOrg)
	}
	return r
}
```

Change `health` to report role:

```go
func (s *server) health(c *gin.Context) {
	count, err := s.store.UserCount(c.Request.Context())
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, HealthResponse{
		Status:           "ok",
		Version:          mcpsrv.Version,
		Principal:        string(principalFrom(c)),
		RegistrationOpen: count == 0,
	})
}
```

to:

```go
func (s *server) health(c *gin.Context) {
	count, err := s.store.UserCount(c.Request.Context())
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, HealthResponse{
		Status:           "ok",
		Version:          mcpsrv.Version,
		Principal:        string(principalFrom(c)),
		Role:             roleFrom(c),
		RegistrationOpen: count == 0,
	})
}
```

- [ ] **Step 4: Build, vet, lint, test**

Run: `task api:build`
Expected: exits 0.

Run: `go vet ./...`
Expected: exits 0.

Run: `task api:lint`
Expected: exits 0 — this is the step that would catch `roleFrom` or `requireAdmin` as unused if either weren't wired in; both are (`roleFrom` by `health` and `requireAdmin` itself; `requireAdmin` by the new `admin` route group), so this must pass clean.

Run: `task api:test`
Expected: `ok` for every package with tests, unchanged from Task 1's run.

- [ ] **Step 5: Commit**

```bash
git add internal/api/types.go internal/api/api.go internal/api/usage.go
git commit -m "feat: add role enforcement and split /api/usage's org-wide report

roleFrom/requireAdmin follow authenticate()/principalFrom's existing
shape. authenticate() now resolves a real role on the header/default
path too, not only the session path. GET /api/usage/org carries the
org-wide cost/growth report behind requireAdmin(); GET /api/usage
keeps only the caller's own headroom."
```

---

### Task 3: Frontend API layer

**Files:**
- Modify (generated): `web/src/api/types.gen.ts`
- Modify: `web/src/api/types.ts`
- Modify: `web/src/api/routes.ts`
- Modify: `web/src/api/client.ts`
- Modify: `web/src/api/hooks.ts`
- Modify: `web/src/api/index.ts`
- Modify: `web/src/router.test.tsx`
- Modify: `web/src/pages/Register.test.tsx`

**Interfaces:**
- Consumes: Task 2's Go types (`HealthResponse.Role`, trimmed `UsageResponse`, `OrgUsageResponse`) and route (`GET /api/usage/org`).
- Produces: `getUsageOrg(params?: OrgUsageParams): Promise<OrgUsageResponse>`, `useUsageOrg(params: OrgUsageParams, enabled: boolean)`, updated `useUsage()` (no params), `queryKeys.usage` (plain key), `queryKeys.usageOrg(params)`. Task 4 consumes these exact names — do not rename.

- [ ] **Step 1: Regenerate the TypeScript types**

Run: `task gen:types`
Expected: exits 0. `web/src/api/types.gen.ts` now has `role: string` on `HealthResponse`, `UsageResponse` without `cost_by_scope`/`scope_growth`, a new `OrgUsageResponse` interface, and `OrgUsageQuery` in place of `UsageQuery`.

Verify: `grep -c "interface OrgUsageResponse" web/src/api/types.gen.ts`
Expected: `1`

- [ ] **Step 2: Re-export the new type**

In `web/src/api/types.ts`, change:

```ts
export type {
  ErrorResponse,
  HealthResponse as Health,
  Scope,
  Source,
  ClaimListItem,
  ClaimListResponse as ClaimList,
  EvidenceItem as Evidence,
  ClaimDetail,
  RecallResponse,
  RecalledClaim,
  Contribution as ArmContribution,
  RecallTimings,
  UsageResponse,
  LimitStatus,
  ScopeCost,
  ScopeGrowth,
  RegisterRequest,
  LoginRequest,
  AuthResponse,
} from './types.gen';
```

to:

```ts
export type {
  ErrorResponse,
  HealthResponse as Health,
  Scope,
  Source,
  ClaimListItem,
  ClaimListResponse as ClaimList,
  EvidenceItem as Evidence,
  ClaimDetail,
  RecallResponse,
  RecalledClaim,
  Contribution as ArmContribution,
  RecallTimings,
  UsageResponse,
  OrgUsageResponse,
  LimitStatus,
  ScopeCost,
  ScopeGrowth,
  RegisterRequest,
  LoginRequest,
  AuthResponse,
} from './types.gen';
```

- [ ] **Step 3: Add the route**

In `web/src/api/routes.ts`, change:

```ts
  recall: () => `${API_BASE}/recall`,
  usage: () => `${API_BASE}/usage`,
  register: () => `${API_BASE}/auth/register`,
  login: () => `${API_BASE}/auth/login`,
  logout: () => `${API_BASE}/auth/logout`,
} as const;
```

to:

```ts
  recall: () => `${API_BASE}/recall`,
  usage: () => `${API_BASE}/usage`,
  usageOrg: () => `${API_BASE}/usage/org`,
  register: () => `${API_BASE}/auth/register`,
  login: () => `${API_BASE}/auth/login`,
  logout: () => `${API_BASE}/auth/logout`,
} as const;
```

- [ ] **Step 4: Update `client.ts`**

Change the type import:

```ts
import { routes } from './routes';
import type {
  AuthResponse,
  ClaimDetail,
  ClaimList,
  Health,
  LoginRequest,
  RecallResponse,
  RegisterRequest,
  UsageResponse,
} from './types';
```

to:

```ts
import { routes } from './routes';
import type {
  AuthResponse,
  ClaimDetail,
  ClaimList,
  Health,
  LoginRequest,
  OrgUsageResponse,
  RecallResponse,
  RegisterRequest,
  UsageResponse,
} from './types';
```

Change:

```ts
export interface UsageParams {
  scopes?: number;
}

export function getUsage(params: UsageParams = {}): Promise<UsageResponse> {
  const query = new URLSearchParams();
  if (params.scopes !== undefined) query.set('scopes', String(params.scopes));

  const qs = query.toString();
  return request<UsageResponse>(qs ? `${routes.usage()}?${qs}` : routes.usage());
}
```

to:

```ts
export function getUsage(): Promise<UsageResponse> {
  return request<UsageResponse>(routes.usage());
}

export interface OrgUsageParams {
  scopes?: number;
}

export function getUsageOrg(params: OrgUsageParams = {}): Promise<OrgUsageResponse> {
  const query = new URLSearchParams();
  if (params.scopes !== undefined) query.set('scopes', String(params.scopes));

  const qs = query.toString();
  return request<OrgUsageResponse>(
    qs ? `${routes.usageOrg()}?${qs}` : routes.usageOrg(),
  );
}
```

- [ ] **Step 5: Update `hooks.ts`**

Change:

```ts
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  getClaim,
  getClaims,
  getHealth,
  getRecall,
  getUsage,
  login,
  logout,
  register,
} from './client';
import type { ClaimsParams, RecallParams, UsageParams } from './client';
import type { LoginRequest, RegisterRequest } from './types';

export const queryKeys = {
  health: ['health'] as const,
  claims: (params: ClaimsParams) => ['claims', params] as const,
  claim: (id: string) => ['claim', id] as const,
  recall: (params: RecallParams) => ['recall', params] as const,
  usage: (params: UsageParams) => ['usage', params] as const,
};
```

to:

```ts
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  getClaim,
  getClaims,
  getHealth,
  getRecall,
  getUsage,
  getUsageOrg,
  login,
  logout,
  register,
} from './client';
import type { ClaimsParams, OrgUsageParams, RecallParams } from './client';
import type { LoginRequest, RegisterRequest } from './types';

export const queryKeys = {
  health: ['health'] as const,
  claims: (params: ClaimsParams) => ['claims', params] as const,
  claim: (id: string) => ['claim', id] as const,
  recall: (params: RecallParams) => ['recall', params] as const,
  usage: ['usage'] as const,
  usageOrg: (params: OrgUsageParams) => ['usage', 'org', params] as const,
};
```

Change:

```ts
export function useUsage(params: UsageParams = {}) {
  return useQuery({
    queryKey: queryKeys.usage(params),
    queryFn: () => getUsage(params),
  });
}
```

to:

```ts
export function useUsage() {
  return useQuery({
    queryKey: queryKeys.usage,
    queryFn: getUsage,
  });
}

// The org-wide report is admin-only; the caller passes enabled so a
// member's browser never issues a request that would 403 -- the same
// enabled-gating shape useRecall already uses for its own precondition.
export function useUsageOrg(params: OrgUsageParams, enabled: boolean) {
  return useQuery({
    queryKey: queryKeys.usageOrg(params),
    queryFn: () => getUsageOrg(params),
    enabled,
  });
}
```

- [ ] **Step 6: Re-export from the API barrel**

In `web/src/api/index.ts`, change:

```ts
export * from './types';
export { routes, API_BASE } from './routes';
export {
  ApiError,
  getHealth,
  getClaims,
  getClaim,
  getRecall,
  getUsage,
  login,
  logout,
  register,
  setPrincipal,
  getPrincipal,
} from './client';
export type {
  ClaimsParams,
  StatusFilter,
  RecallParams,
  UsageParams,
} from './client';
export {
  useHealth,
  useClaims,
  useClaim,
  useRecall,
  useUsage,
  useLogin,
  useLogout,
  useRegister,
  queryKeys,
} from './hooks';
```

to:

```ts
export * from './types';
export { routes, API_BASE } from './routes';
export {
  ApiError,
  getHealth,
  getClaims,
  getClaim,
  getRecall,
  getUsage,
  getUsageOrg,
  login,
  logout,
  register,
  setPrincipal,
  getPrincipal,
} from './client';
export type {
  ClaimsParams,
  StatusFilter,
  RecallParams,
  OrgUsageParams,
} from './client';
export {
  useHealth,
  useClaims,
  useClaim,
  useRecall,
  useUsage,
  useUsageOrg,
  useLogin,
  useLogout,
  useRegister,
  queryKeys,
} from './hooks';
```

- [ ] **Step 7: Fix the two test files that construct a full `Health` object**

In `web/src/router.test.tsx`, change:

```ts
const AUTHENTICATED: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: 'local',
  registration_open: false,
};
```

to:

```ts
const AUTHENTICATED: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: 'local',
  role: 'admin',
  registration_open: false,
};
```

And change:

```ts
    vi.mocked(getHealth).mockResolvedValue({
      status: 'ok',
      version: '0.1.0',
      principal: '',
      registration_open: false,
    });
```

to:

```ts
    vi.mocked(getHealth).mockResolvedValue({
      status: 'ok',
      version: '0.1.0',
      principal: '',
      role: '',
      registration_open: false,
    });
```

In `web/src/pages/Register.test.tsx`, change:

```ts
const OPEN: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: '',
  registration_open: true,
};
```

to:

```ts
const OPEN: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: '',
  role: '',
  registration_open: true,
};
```

(`CLOSED: Health = { ...OPEN, registration_open: false };` needs no change — it spreads `OPEN`, which now already carries `role`.)

- [ ] **Step 8: Typecheck and run the existing frontend suite**

Run: `cd web && npm run typecheck`
Expected: exits 0.

Run: `cd web && npx vitest run src/router.test.tsx src/pages/Register.test.tsx`
Expected: PASS — these two files are fixed by Step 7. (`src/pages/Usage.test.tsx` is expected to FAIL at this point; Task 4 rewrites it.)

- [ ] **Step 9: Commit**

```bash
git add web/src/api/ web/src/router.test.tsx web/src/pages/Register.test.tsx
git commit -m "feat: wire the usage split and role into the frontend API layer

useUsageOrg follows useRecall's enabled-gating shape so a member's
browser never requests the admin-only report. Health gaining a
required role field breaks two existing test fixtures that construct
a full Health literal; both get one added. Types regenerated with
task gen:types."
```

---

### Task 4: `UsagePage` — role-gated org-wide tables

**Files:**
- Modify: `web/src/pages/UsagePage.tsx`
- Modify: `web/src/pages/Usage.test.tsx`

**Interfaces:**
- Consumes: Task 3's `useHealth`, `useUsage`, `useUsageOrg`, `queryKeys.usage`, `OrgUsageResponse` type.

- [ ] **Step 1: Rewrite `UsagePage.tsx`**

Replace the full contents of `web/src/pages/UsagePage.tsx` with:

```tsx
import { Layout, LayoutContent, LayoutHeader } from '@astryxdesign/core/Layout';
import { HStack, VStack } from '@astryxdesign/core/Stack';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { Heading, Text } from '@astryxdesign/core/Text';
import { Spinner } from '@astryxdesign/core/Spinner';
import { Badge } from '@astryxdesign/core/Badge';
import { Banner } from '@astryxdesign/core/Banner';
import { Button } from '@astryxdesign/core/Button';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { ProgressBar } from '@astryxdesign/core/ProgressBar';
import { Table, proportional, pixel } from '@astryxdesign/core/Table';
import type { TableColumn } from '@astryxdesign/core/Table';
import { useQueryClient } from '@tanstack/react-query';
import { useHealth, useUsage, useUsageOrg, queryKeys } from '../api';
import type {
  LimitStatus,
  OrgUsageResponse,
  ScopeCost,
  ScopeGrowth,
  UsageResponse,
} from '../api';

type ScopeCostRow = ScopeCost & Record<string, unknown>;
type ScopeGrowthRow = ScopeGrowth & Record<string, unknown>;

const scopeCostColumns: TableColumn<ScopeCostRow>[] = [
  {
    key: 'scope',
    header: 'Scope',
    width: proportional(2),
    renderCell: (row) => (
      <Text type="body" maxLines={1}>
        {row.scope.kind}: {row.scope.value}
      </Text>
    ),
  },
  {
    key: 'calls',
    header: 'Calls',
    width: pixel(100),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.calls}
      </Text>
    ),
  },
  {
    key: 'input_tokens',
    header: 'Input tokens',
    width: pixel(140),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.input_tokens}
      </Text>
    ),
  },
  {
    key: 'output_tokens',
    header: 'Output tokens',
    width: pixel(140),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.output_tokens}
      </Text>
    ),
  },
];

const scopeGrowthColumns: TableColumn<ScopeGrowthRow>[] = [
  {
    key: 'scope',
    header: 'Scope',
    width: proportional(2),
    renderCell: (row) => (
      <Text type="body" maxLines={1}>
        {row.scope.kind}: {row.scope.value}
      </Text>
    ),
  },
  {
    key: 'live',
    header: 'Live claims',
    width: pixel(120),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.live}
      </Text>
    ),
  },
  {
    key: 'ceiling',
    header: 'Ceiling',
    width: pixel(100),
    renderCell: (row) => (
      <Text type="body" color="secondary" hasTabularNumbers>
        {row.ceiling <= 0 ? 'off' : row.ceiling}
      </Text>
    ),
  },
  {
    key: 'next_prune',
    header: 'Next prune',
    width: pixel(120),
    renderCell: (row) => (
      <Text type="body" hasTabularNumbers>
        {row.next_prune}
      </Text>
    ),
  },
];

export function UsagePage() {
  const health = useHealth();
  const isAdmin = health.data?.role === 'admin';
  const usage = useUsage();
  const usageOrg = useUsageOrg({}, isAdmin);
  const queryClient = useQueryClient();

  return (
    <Layout
      header={
        <LayoutHeader hasDivider>
          <HStack vAlign="center" hAlign="between">
            <Heading level={4}>Usage</Heading>
            <Button
              label="Refresh"
              variant="secondary"
              onClick={() => {
                void queryClient.invalidateQueries({
                  queryKey: queryKeys.usage,
                });
              }}
            />
          </HStack>
        </LayoutHeader>
      }
      content={
        <LayoutContent>
          <UsageBody
            isLoading={usage.isLoading}
            isError={usage.isError}
            data={usage.data}
            isAdmin={isAdmin}
            orgIsLoading={usageOrg.isLoading}
            orgData={usageOrg.data}
          />
        </LayoutContent>
      }
    />
  );
}

function UsageBody({
  isLoading,
  isError,
  data,
  isAdmin,
  orgIsLoading,
  orgData,
}: {
  isLoading: boolean;
  isError: boolean;
  data: UsageResponse | undefined;
  isAdmin: boolean;
  orgIsLoading: boolean;
  orgData: OrgUsageResponse | undefined;
}) {
  if (isLoading) {
    return (
      <Center height="100%">
        <Spinner label="Loading usage" />
      </Center>
    );
  }

  if (isError || !data) {
    return (
      <Center height="100%">
        <EmptyState
          title="Couldn't load usage"
          description="The API request failed. Check that the CRED server is running."
        />
      </Center>
    );
  }

  return (
    <VStack gap={4}>
      {data.denied > 0 ? (
        <Banner
          status="warning"
          title={`${data.denied} contribution(s) denied in the last ${data.denied_window}`}
          description="Recorded and on the record — nothing was silently dropped."
        />
      ) : null}

      <HStack gap={3} wrap="wrap">
        <LimitCard label="Contribution" status={data.contribution} />
        <LimitCard
          label="Inference cost"
          status={data.cost}
          extra={`${data.input_tokens_used} / ${
            data.input_tokens_ceiling <= 0 ? 'off' : data.input_tokens_ceiling
          } input tokens`}
        />
        <LimitCard label="Recall" status={data.recall} />
      </HStack>

      {isAdmin ? (
        <OrgUsageSection isLoading={orgIsLoading} data={orgData} />
      ) : null}
    </VStack>
  );
}

function OrgUsageSection({
  isLoading,
  data,
}: {
  isLoading: boolean;
  data: OrgUsageResponse | undefined;
}) {
  if (isLoading || !data) {
    return (
      <Center height={120}>
        <Spinner label="Loading org usage" />
      </Center>
    );
  }

  return (
    <VStack gap={4}>
      <VStack gap={2}>
        <Heading level={5}>Cost by scope</Heading>
        {data.cost_by_scope.length === 0 ? (
          <Text type="body" color="secondary">
            No inference cost recorded in this window.
          </Text>
        ) : (
          <Table<ScopeCostRow>
            data={data.cost_by_scope as ScopeCostRow[]}
            columns={scopeCostColumns}
            idKey={(row) => `${row.scope.kind}:${row.scope.value}`}
            textOverflow="truncate"
          />
        )}
      </VStack>

      <VStack gap={2}>
        <Heading level={5}>Scope growth</Heading>
        {data.scope_growth.length === 0 ? (
          <Text type="body" color="secondary">
            No live claims.
          </Text>
        ) : (
          <Table<ScopeGrowthRow>
            data={data.scope_growth as ScopeGrowthRow[]}
            columns={scopeGrowthColumns}
            idKey={(row) => `${row.scope.kind}:${row.scope.value}`}
            textOverflow="truncate"
          />
        )}
      </VStack>
    </VStack>
  );
}

function LimitCard({
  label,
  status,
  extra,
}: {
  label: string;
  status: LimitStatus;
  extra?: string;
}) {
  const hasCeiling = status.ceiling > 0;
  const value = hasCeiling
    ? Math.min((status.used / status.ceiling) * 100, 100)
    : 0;

  return (
    <Card padding={3}>
      <VStack gap={2} width={220}>
        <HStack hAlign="between" vAlign="center">
          <Text type="body" weight="semibold">
            {label}
          </Text>
          {!status.allowed ? (
            <Badge variant="warning" label="exhausted" />
          ) : null}
        </HStack>
        {hasCeiling ? (
          <ProgressBar
            label={`${label} usage`}
            isLabelHidden
            value={value}
            variant={status.allowed ? 'accent' : 'error'}
          />
        ) : (
          <Text type="supporting" color="secondary">
            unlimited
          </Text>
        )}
        <Text type="supporting" color="secondary" hasTabularNumbers>
          {status.used} / {hasCeiling ? status.ceiling : 'off'} per{' '}
          {status.window}
        </Text>
        {extra ? (
          <Text type="supporting" color="secondary" hasTabularNumbers>
            {extra}
          </Text>
        ) : null}
      </VStack>
    </Card>
  );
}
```

- [ ] **Step 2: Rewrite `Usage.test.tsx`**

Replace the full contents of `web/src/pages/Usage.test.tsx` with:

```tsx
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { UsagePage } from './UsagePage';
import { getHealth, getUsage, getUsageOrg } from '../api/client';
import type { Health, OrgUsageResponse, UsageResponse } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return {
    ...actual,
    getHealth: vi.fn(),
    getUsage: vi.fn(),
    getUsageOrg: vi.fn(),
  };
});

const ADMIN: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: 'local',
  role: 'admin',
  registration_open: false,
};

const MEMBER: Health = { ...ADMIN, role: 'member' };

const SAMPLE_USAGE: UsageResponse = {
  principal: 'local',
  contribution: {
    window: '1h0m0s',
    used: 12,
    ceiling: 120,
    remaining: 108,
    allowed: true,
    reason: '',
  },
  cost: {
    window: '1h0m0s',
    used: 3,
    ceiling: 500,
    remaining: 497,
    allowed: true,
    reason: '',
  },
  input_tokens_used: 40000,
  input_tokens_ceiling: 2000000,
  recall: {
    window: '1m0s',
    used: 120,
    ceiling: 120,
    remaining: 0,
    allowed: false,
    reason: 'recall_rate',
  },
  denied_window: '1h0m0s',
  denied: 2,
};

const SAMPLE_ORG_USAGE: OrgUsageResponse = {
  cost_by_scope: [
    {
      scope: { kind: 'repo', value: 'cred' },
      calls: 3,
      input_tokens: 40000,
      output_tokens: 900,
    },
  ],
  scope_growth: [
    {
      scope: { kind: 'repo', value: 'cred' },
      live: 4800,
      ceiling: 5000,
      next_prune: 0,
    },
  ],
};

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('UsagePage', () => {
  beforeEach(() => {
    vi.mocked(getUsage).mockResolvedValue(SAMPLE_USAGE);
    vi.mocked(getUsageOrg).mockResolvedValue(SAMPLE_ORG_USAGE);
  });

  it('renders the three limit stats for any authenticated role', async () => {
    vi.mocked(getHealth).mockResolvedValue(MEMBER);
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('Contribution')).toBeInTheDocument();
    expect(screen.getByText('Inference cost')).toBeInTheDocument();
    expect(screen.getByText('Recall')).toBeInTheDocument();
  });

  it('shows a warning banner when contributions have been denied', async () => {
    vi.mocked(getHealth).mockResolvedValue(MEMBER);
    renderWithClient(<UsagePage />);

    expect(
      await screen.findByText(/2 contribution\(s\) denied/),
    ).toBeInTheDocument();
  });

  it('marks an exhausted limit with a warning badge', async () => {
    vi.mocked(getHealth).mockResolvedValue(MEMBER);
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('exhausted')).toBeInTheDocument();
  });

  it('renders the org-wide scope tables for an admin', async () => {
    vi.mocked(getHealth).mockResolvedValue(ADMIN);
    renderWithClient(<UsagePage />);

    expect(await screen.findByText('Cost by scope')).toBeInTheDocument();
    expect(screen.getByText('Scope growth')).toBeInTheDocument();
    expect(screen.getAllByText('repo: cred').length).toBeGreaterThan(0);
  });

  it('hides the org-wide tables for a member and never requests them', async () => {
    vi.mocked(getHealth).mockResolvedValue(MEMBER);
    renderWithClient(<UsagePage />);

    await screen.findByText('Contribution');
    expect(screen.queryByText('Cost by scope')).not.toBeInTheDocument();
    expect(screen.queryByText('Scope growth')).not.toBeInTheDocument();
    expect(getUsageOrg).not.toHaveBeenCalled();
  });
});
```

- [ ] **Step 3: Run the frontend suite and typecheck**

Run: `cd web && npx vitest run src/pages/Usage.test.tsx`
Expected: PASS, all five tests.

Run: `cd web && npm run typecheck`
Expected: exits 0.

Run: `cd web && npm test`
Expected: PASS for every test file, including `router.test.tsx` and `Register.test.tsx` fixed in Task 3.

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/UsagePage.tsx web/src/pages/Usage.test.tsx
git commit -m "feat: gate UsagePage's org-wide tables on role

An admin's own useHealth() result decides whether useUsageOrg fires
at all, so a member's browser never requests /api/usage/org."
```

---

### Task 5: Manual end-to-end verification

No code changes. Confirms the whole path — store, middleware, route, frontend — works together against a real Postgres and a real browser, the way sub-project 1 verified register/login/logout.

- [ ] **Step 1: Start the stack**

Run: `docker compose up -d db`
Run: `task dev` (or `go run . web` plus `cd web && npm run dev` separately)

- [ ] **Step 2: Verify the admin path**

Log in as the bootstrap admin account (created during sub-project 1's own manual verification, or register fresh if none exists and registration is still open).

Open `/usage` in the browser.

Expected: the Contribution/Inference cost/Recall cards render, and below them "Cost by scope" and "Scope growth" tables render (or their empty-state text, if no usage has been recorded yet).

Open the browser's network tab, refresh.

Expected: a request to `GET /api/usage/org` returns `200`.

- [ ] **Step 3: Verify the member path**

In `psql` (`docker compose exec db psql -U cred -d cred`), flip the bootstrap admin's role to `member` on a *second* principal — since sub-project 1's `user_credentials_one_admin` constraint keeps this a single-admin schema for now, create a second user_credentials row directly for this check rather than touching the real admin:

```sql
INSERT INTO principals (id, kind, display_name)
VALUES ('test-member', 'user', 'test-member@example.test');

INSERT INTO user_credentials (principal_id, email, password_hash, role)
VALUES ('test-member', 'test-member@example.test', '$2a$10$invalidhashfortestingonly', 'member');
```

Run: `curl -i -H "X-CRED-Principal: test-member" http://localhost:8080/api/usage/org`
Expected: `HTTP/1.1 403 Forbidden`, body `{"error":"admin role required"}`.

Run: `curl -i -H "X-CRED-Principal: test-member" http://localhost:8080/api/usage`
Expected: `HTTP/1.1 200 OK`, a normal `UsageResponse` body with no `cost_by_scope`/`scope_growth` fields.

Clean up:

```sql
DELETE FROM user_credentials WHERE principal_id = 'test-member';
DELETE FROM principals WHERE id = 'test-member';
```

- [ ] **Step 4: Verify claim ACL is unaffected**

Run: `curl -s -H "X-CRED-Principal: test-member" http://localhost:8080/api/claims | jq '.count'` (before cleanup, or re-insert the test principal first)

Expected: the same claim-visibility result an ordinary, non-admin principal would get today — confirms role enforcement didn't change `GET /api/claims`'s behavior at all, since that route was never touched by this plan.

No commit for this task — it's verification, not a code change.
