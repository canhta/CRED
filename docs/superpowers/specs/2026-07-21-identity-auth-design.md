# Identity & auth mechanism — design

- **Date:** 2026-07-21
- **Status:** Approved by operator, pending implementation plan.
- **Decision:** `docs/research/decision-log.md` D-025 — multi-user auth and
  RBAC pulled into v1, reversing D-001's "no SSO, RBAC, audit, or org
  hierarchy" line. This spec is the first of four sub-projects D-025 names:
  identity & auth (this doc) → role model + RBAC/ACL integration → invite
  flow → MCP auth. Only sub-project 1 is designed here.

## Problem

The web console's `authenticate()` middleware (`internal/api/api.go:73-97`)
resolves identity from an `X-CRED-Principal` header, optionally gated by one
shared bearer token. Any caller holding the token can claim to be any
principal — there is no verification that the caller *is* who the header
says. This is fine for the single-operator self-hosted case the console
shipped for, and it is explicitly commented as a placeholder ("replacing it
with OIDC/SSO later touches no handler"). D-025 spends that placeholder: real
accounts, so a session's principal is something the server verified, not
something the client asserted.

## A finding that changes the design

`internal/store/migrations/00001_initial_schema.sql` already has a
`principals` table:

```sql
CREATE TABLE principals (
    id             text PRIMARY KEY,
    kind           text        NOT NULL CHECK (kind IN ('user', 'team', 'org', 'agent')),
    display_name   text        NOT NULL,
    recorded_at    timestamptz NOT NULL DEFAULT now(),
    superseded_at  timestamptz,
    CONSTRAINT principals_recorded_half_open
        CHECK (superseded_at IS NULL OR recorded_at < superseded_at)
);
```

Its own migration comment states the intent directly: *"org, member and role
primitives live in the binary a solo developer runs offline... This slice
ships exactly one principal and the table is here anyway, because every
competitor that deferred the principal type could not add it back"* — a
reference to D-014.

**It is dormant.** `grep -rn "principals\b" internal/ --include='*.go'`
matches only the `claim.PrincipalID` *type* and function names — no `INSERT
INTO principals`, no `SELECT ... FROM principals`, anywhere. This is the same
defect this project's own research documented in mem0's OpenMemory
`AccessControl` table: a schema built for governance with zero writers,
dormant by construction (`docs/research/evidence/mem0.md` §5, "a schema
without an implementation"). Recorded here rather than smoothed over, per
this repo's own evidence standard.

**This design activates `principals` instead of adding a parallel `users`
table.** `principals.id` is already `claim.PrincipalID` — the exact type used
throughout ACL grants (`internal/claim.Grant.Principal`), `contributed_by`,
and `usage_events.principal_id`. A session that resolves to a real
`principals` row participates in every existing mechanism immediately, with
no translation layer, and it is what D-014 bet on: the table survived
specifically so this would not need to be rebuilt from scratch.

## Approach

**Password auth by default, hand-rolled sessions, bcrypt for hashing, an
explicit seam for OAuth later.** Three library decisions, each argued on its
own terms rather than as a bundle:

- **`golang.org/x/crypto/bcrypt` for password hashing.** Not currently
  compiled in (`go mod why golang.org/x/crypto` → "main module does not need
  package"), so this is a new direct dependency — but it is the same tier as
  the already-accepted `golang.org/x/text` (an `x/` extended-stdlib package,
  not a framework), its checksum is already in `go.sum` via the module
  graph, and hand-rolling password hashing is a real security risk, not a
  place to economize. The reason goes in the commit message per `go.md` §9.
- **Hand-rolled sessions, no library.** A `sessions` table, `crypto/rand`
  (stdlib) for the opaque token, `net/http.Cookie` (stdlib) for the cookie.
  This is the "copy thirty lines over adding a module" case `go.md` §9
  describes — and it matches how mem0's own `server/auth.py` did it (220
  lines, hand-rolled, no framework), the one auth surface in that codebase's
  review that was called competent.
- **No change needed for the future OAuth seam or MCP auth.**
  `golang.org/x/oauth2` and `github.com/golang-jwt/jwt/v5` are **already
  live dependencies** — pulled in by the MCP Go SDK's own `auth`/`oauthex`
  packages (`go mod why golang.org/x/oauth2` confirms the import chain
  through `internal/mcpsrv`), which implement OAuth 2.1 resource-server
  support matching the PRD's stated MCP-auth target
  (`docs/research/spikes/tech-decisions.md:17`). Nothing to add now; noted
  so sub-project 4 doesn't re-discover this.

Rejected: an auth framework (`authboss`, `goth`) or an external identity
service (`ory/kratos`, Keycloak) — either fights the one-binary self-hosted
model (D-022) or adds unjustified weight for a feature this small.

**Login rate limiting is in scope for this pass**, not deferred to
sub-project 2 as an earlier draft of this spec had it — `POST
/api/auth/login` is the one unauthenticated write path this feature adds,
and it is exactly the surface credential stuffing and brute force target.
`internal/limit` already has the shape for this: a pure decision function
over a windowed count (`Contribution`, `Cost`, `RecallRate`, all following
the same `atMost` helper), with the store supplying the counter and never
the decision. This adds a fourth: `LoginAttempts`.

## Schema

New migration `internal/store/migrations/00006_identity.sql`:

```sql
-- +goose Up
-- +goose StatementBegin

-- Activates the principals table D-014 shipped and D-025 spends: a human
-- account is a principals row with kind='user', plus the auth-specific
-- fields that don't belong on team/org/agent principals. Separate table
-- rather than columns on principals, so principals stays the general
-- identity model every kind shares.
CREATE TABLE user_credentials (
    principal_id   text        PRIMARY KEY REFERENCES principals(id),
    email          text        NOT NULL UNIQUE,
    password_hash  text        NOT NULL,
    role           text        NOT NULL DEFAULT 'member'
                       CHECK (role IN ('admin', 'member')),
    created_at     timestamptz NOT NULL DEFAULT now()
);

-- Sessions are revocable by construction: a row deleted (or expired) here is
-- a session that stops working on the next request, unlike a stateless JWT.
-- token_hash, never the raw token — the same reason claim content is never
-- logged: a leaked table dump must not itself be a bearer credential.
CREATE TABLE sessions (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    principal_id   text        NOT NULL REFERENCES principals(id),
    token_hash     text        NOT NULL UNIQUE,
    created_at     timestamptz NOT NULL DEFAULT now(),
    expires_at     timestamptz NOT NULL
);

CREATE INDEX sessions_token_hash_idx ON sessions (token_hash);
CREATE INDEX sessions_principal_idx ON sessions (principal_id);

-- login_attempts backs the login rate limit. Keyed by email, not
-- principal_id: a failed login against an email with no account must still
-- count, or an attacker learns which emails are registered by which ones
-- never trip the limiter -- the same existence-oracle failure the login
-- handler's identical 401 already avoids at the response level.
CREATE TABLE login_attempts (
    id           bigint      GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email        text        NOT NULL,
    succeeded    boolean     NOT NULL,
    recorded_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX login_attempts_email_time ON login_attempts (email, recorded_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS login_attempts;
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_credentials;
-- +goose StatementEnd
```

`translate()` (`internal/store/pg/pg.go:82-100`) gains one more case,
matching the existing `ErrExtensionMissing` pattern: a unique violation on
`user_credentials_email_key` translates to a new `ErrEmailTaken` sentinel,
so the register handler can return 409 without importing `pgconn`.

## Rate limiting (`internal/limit`)

New `Config` fields, alongside the existing contribution/cost/recall ones
(`internal/limit/limit.go:39-62`), with the same "generous enough a single
developer never hits it" defaults philosophy:

```go
// Login attempts: failed logins per email per window. The brute-force /
// credential-stuffing defence on the one unauthenticated write path.
MaxLoginAttempts int
LoginWindow      time.Duration
```

Default: `MaxLoginAttempts: 10, LoginWindow: 15 * time.Minute` — well above
a human mistyping a password twice, well below a scripted attempt loop.

```go
// LoginAttempts decides whether one more login attempt for this email may
// proceed, given how many have failed in the current window. Only failures
// count toward the ceiling -- a user who logs in correctly every day must
// never be at risk of tripping a limit meant for attackers.
func LoginAttempts(failedInWindow int, cfg Config) Decision {
	return atMost(failedInWindow, cfg.MaxLoginAttempts, ReasonLoginAttempts)
}
```

`ReasonLoginAttempts` joins `ReasonContributionQuota`, `ReasonCostCeiling`,
`ReasonRecallRate` (`internal/limit/limit.go:27-32`) as a fourth `Reason`
constant. `internal/limit/limit_test.go` gets one more table-driven case
alongside the existing `Contribution`/`Cost`/`RecallRate` tests — same
shape, new function, no new test infrastructure.

## Store layer

New file `internal/store/pg/identity.go`, following the existing package's
shape (plain methods on `*Store`, no principal-taking "decision" methods —
identity resolution is a lookup, access control still happens in
`internal/acl`):

- `CreateUser(ctx, email, passwordHash string) (claim.PrincipalID, error)` —
  one transaction: `INSERT INTO principals (id, kind, display_name) VALUES
  (gen_random_uuid()::text, 'user', email)`, then `INSERT INTO
  user_credentials`. Returns the new principal ID. `id` is a generated UUID
  string, not the email — an email change must not be an identity change
  for every claim already attributed to this principal.
- `UserCount(ctx) (int, error)` — `SELECT count(*) FROM user_credentials`.
  Backs the "first registration becomes admin, then registration closes"
  rule; the handler decides, this only counts.
- `CredentialsByEmail(ctx, email string) (principalID claim.PrincipalID,
  passwordHash string, role string, err error)` — for login.
- `CreateSession(ctx, principal claim.PrincipalID, tokenHash string, expiresAt
  time.Time) error`.
- `SessionPrincipal(ctx, tokenHash string, now time.Time) (claim.PrincipalID,
  string /* role */, error)` — `ErrNotFound` (translated) for an absent or
  expired session; the query filters `expires_at > now` itself rather than
  returning a row the caller must re-check, matching the half-open interval
  convention `internal/temporal` already uses elsewhere.
- `DeleteSession(ctx, tokenHash string) error` — logout.
- `RecordLoginAttempt(ctx, email string, succeeded bool, now time.Time)
  error` — one append-only row, mirroring `RecordUsage`'s "plain INSERT, no
  read-modify-write" note (`internal/store/pg/usage.go:36-38`).
- `FailedLoginsInWindow(ctx, email string, since time.Time) (int, error)` —
  `SELECT count(*) FROM login_attempts WHERE email = $1 AND succeeded =
  false AND recorded_at > $2`. The counter `limit.LoginAttempts` decides
  over; this package never compares it to a ceiling itself, same discipline
  as every other counter in this file.

## API surface

New file `internal/api/auth.go`, three routes, registered outside the
existing `authenticate()` middleware (login and register cannot require a
session to reach) but still inside a route group so `requestLogger` still
covers them:

```go
authGroup := r.Group("/api/auth")
{
    authGroup.POST("/register", s.register)
    authGroup.POST("/login", s.login)
}
api.POST("/auth/logout", s.logout) // inside the authenticated group
```

- `POST /api/auth/register` — `{email, password}`. If `UserCount() > 0`,
  `403` (`ErrorResponse{Error: "registration is closed"}`) — registration is
  open exactly until the first account exists, per the operator's chosen
  bootstrap. Otherwise: `bcrypt.GenerateFromPassword`, `CreateUser`, force
  `role='admin'` on this one row regardless of what the request carried (a
  client-supplied role on the only unauthenticated write path is exactly
  the kind of privilege-escalation seam this whole feature exists to close)
  — starts a session, sets the cookie, `200`.
- `POST /api/auth/login` — `{email, password}`. Checks
  `FailedLoginsInWindow` against `limit.LoginAttempts` **before** touching
  bcrypt (a bcrypt compare is deliberately slow; running it before the rate
  check would let an attacker burn server CPU right up to the limit on
  every window). Over the limit: `429`
  (`ErrorResponse{Error: "too many login attempts, try again later"}`),
  matching the existing 429 shape `internal/api/recall.go:120-125` uses for
  the recall-rate limit. Otherwise: `CredentialsByEmail`,
  `bcrypt.CompareHashAndPassword`, `RecordLoginAttempt` either way. Same
  `401` for "no such email" and "wrong password" — an email-existence
  oracle is the same failure class `getClaim`'s 404 already avoids for
  authorization (`internal/api/claims.go:109-115`). On success: starts a
  session, sets the cookie, `200`.
- `POST /api/auth/logout` — reads the session cookie, `DeleteSession`,
  clears the cookie (`Max-Age=-1`), `204`.
- Session cookie: name `cred_session`, `HttpOnly`, `Secure` (skipped only
  when `cfg.WebAddr` binds to loopback in dev — matches how `CRED_WEB_TOKEN`
  is already optional for the same reason), `SameSite=Lax`, `Path=/`,
  `Max-Age` matching the session's `expires_at` (30 days, not configurable
  in this pass).
- Token generation: 32 bytes from `crypto/rand`, base64url-encoded for the
  cookie value; the *hash* of that value (SHA-256, stdlib `crypto/sha256`)
  is what `sessions.token_hash` stores — the raw token is a bearer
  credential and is never written to the database, mirroring why claim
  content is never logged.

## `authenticate()` change

`internal/api/api.go:73-97` gains a session-cookie path **ahead of** the
existing header/bearer path, and the session path is authoritative when
present — a session cookie is never overridden by a client-supplied header:

```go
func authenticate(cfg config.Config, store *pg.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cookie, err := c.Cookie("cred_session"); err == nil {
			if principal, _, err := sessionPrincipal(c.Request.Context(), store, cookie); err == nil {
				ctx := context.WithValue(c.Request.Context(), principalKey{}, principal)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}
			// Invalid/expired cookie falls through to the existing path rather
			// than failing the request outright — a stale cookie from a rotated
			// session must not lock out a caller who also has a valid bearer
			// token configured.
		}
		// ...existing header/bearer-token body, unchanged...
	}
}
```

`role` is resolved and attached to context alongside the principal in the
same call (a second, distinct context key) so sub-project 2's route guards
have it without a second query — but no route enforces it yet in this pass.

## Frontend

- `web/src/pages/LoginPage.tsx` — email/password form. On success,
  `navigate({ to: '/claims' })`; the session cookie is set by the response,
  nothing stored in JS-reachable state (no token in `localStorage`, closing
  off the XSS-exfiltration path a JWT-in-localStorage design would carry).
- `web/src/pages/RegisterPage.tsx` — same form shape, posts to
  `/api/auth/register`. `HealthResponse` (`internal/api/types.go:12-18`)
  gains one field, `RegistrationOpen bool`, backed by `UserCount() == 0` —
  reusing the health check every page already has access to rather than
  adding a second existence-check route. The page renders the form only
  when `RegistrationOpen`; otherwise it redirects to `/login` with a
  message explaining why.
- Route guard: a `beforeLoad` on the root route (`web/src/router.tsx`)
  checks `useHealth()`-style session state and redirects to `/login` when
  unauthenticated — implemented as a check on `GET /api/health`'s existing
  `principal` field, which is `""` for an unauthenticated request once
  `authenticate()` stops defaulting to `cfg.Principal` for browser-session
  callers. (The default-principal fallback stays for CLI/MCP/automation
  callers using the header/bearer path — only the *cookie* path treats an
  absent identity as unauthenticated, not the whole middleware.)
- `setPrincipal`/`getPrincipal` in `web/src/api/client.ts` (currently a
  client-settable `X-CRED-Principal` override) get a doc-comment update:
  they remain for the header/bearer path (automation, testing), and no
  longer describe themselves as "the seam" now that a real session exists
  for the browser path.

## What's deliberately out of scope for this pass

- **Enforcing `role` on any route.** Sub-project 2. `role` is stored and
  attached to context now so that work doesn't need a second migration.
- **Invite flow.** Sub-project 3. Registration closing after the first
  account is the only way a second account can exist until that ships —
  named as a real, temporary limitation, not hidden.
- **MCP picking up this identity.** Sub-project 4. `internal/mcpsrv` is
  unaffected by this pass.
- **Password reset.** Needs an SMTP/email-sending story CRED doesn't have.
  An admin recreating a lost account via direct database access is the
  accepted gap for this pass.
- **Configurable session lifetime.** Hardcoded 30 days in this pass.
- **Account lockout** (as opposed to the login-attempt *rate* limit, which
  is in scope — see above). A ceiling that denies further attempts for the
  window is not the same as disabling the account until an admin
  intervenes; the latter needs an admin surface that doesn't exist until
  sub-project 2, so it stays out for now.

## Testing

Follows this codebase's established `internal/api` precedent (no per-handler
Go test file — see the Usage-page plan's Task 1 note) for the HTTP layer,
plus real unit tests where pure logic exists:

- `internal/limit/limit_test.go` — one new table-driven case for
  `LoginAttempts`, same shape as the existing `Contribution`/`Cost`/
  `RecallRate` cases (under ceiling allows, at ceiling denies, disabled
  ceiling is unlimited).
- `internal/store/pg/identity_test.go` — none beyond what
  `internal/store/pg/integration_test.go` already exercises as a pattern;
  the store methods here are plain counters and inserts, not decisions.
- Frontend: `web/src/pages/Login.test.tsx`, `Register.test.tsx` — mocked
  client, following `Claims.test.tsx`'s pattern: submit the form, assert
  the redirect call, assert a 401 renders an inline error rather than a
  full-page `EmptyState` (a login failure is not "the server is down").
- Manual end-to-end verification (real Postgres, live browser: register as
  the first user, confirm `role=admin` in the database, log out, log back
  in, confirm a second registration attempt is rejected) — same practice
  used for Usage, Recall, and Claim Detail.
