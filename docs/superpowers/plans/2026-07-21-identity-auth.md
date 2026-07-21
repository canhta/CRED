# Identity & Auth Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the shared `X-CRED-Principal` header trust model with real accounts — email/password registration (first account becomes admin), login, logout, sessions, and login-attempt rate limiting — by activating the dormant `principals` table instead of adding a parallel one.

**Architecture:** A new migration adds `user_credentials`, `sessions`, and `login_attempts`, all keyed to the existing `principals.id` (= `claim.PrincipalID`). `internal/limit` gains a fourth pure decision function, `LoginAttempts`, following the exact shape of `Contribution`/`Cost`/`RecallRate`. `internal/store/pg/identity.go` adds the counters and inserts those decisions run over. `internal/api/auth.go` adds three handlers; `authenticate()` gains a session-cookie path that is authoritative over the header when present. The frontend adds a login/register pair outside the authenticated `Shell`, guarded by a `beforeLoad` check on a new pathless layout route wrapping every existing page.

**Tech Stack:** Go 1.25, Gin, pgx, `golang.org/x/crypto/bcrypt` (new direct dependency) · React 19, TanStack Query/Router, Astryx, Vitest.

## Global Constraints

- `CGO_ENABLED=0` for every Go build.
- No hand-written TypeScript type may mirror a Go struct — regenerate with `task gen:types`.
- No `<div>` for layout in React — compose Astryx components.
- Every API route lives once in `web/src/api/routes.ts`.
- Handlers read the principal via `principalFrom(c)`, never a header or cookie directly.
- Comments are sparse, explain the *why*, never cite a decision number or law number.
- Password hashing: `golang.org/x/crypto/bcrypt`, `bcrypt.DefaultCost`. No hand-rolled hashing.
- Sessions: hand-rolled (`crypto/rand` + a `sessions` table + `net/http.Cookie`), no session library.
- `principal_id` for a new user is a generated UUID (`gen_random_uuid()::text`), never the email — an email change must not be an identity change.
- Role enforcement on routes, the invite flow, MCP auth, password reset, account lockout, and configurable session lifetime are **out of scope for this plan** (see `docs/superpowers/specs/2026-07-21-identity-auth-design.md`, "What's deliberately out of scope").
- This codebase's `internal/api` precedent is **no per-handler Go test file** — frontend tests against a mocked client plus manual end-to-end verification. This plan follows that precedent; do not add one unilaterally.

**Corrections made while planning** (the spec described the shape; these are the exact resolutions found while turning it into code):
- The spec suggested registering `/api/auth/*` "outside the existing `authenticate()` middleware." Re-reading `internal/api/api.go`, `authenticate()` never rejects a request for having no principal — it only rejects a `WebToken` mismatch. Register and login can therefore be plain routes in the same `/api` group as everything else; no second route group is needed.
- The spec's cookie `Secure` flag description ("skipped when `cfg.WebAddr` binds to loopback") would have needed new config and TLS-detection logic. Simpler and correct: always `Secure: true`. Browsers treat `localhost`/`127.0.0.1` as a secure context even over plain HTTP, and `cred web` always serves the API and the SPA from the same origin (embedded binary or Vite's same-origin proxy), so this never breaks local dev.
- Login timing: comparing against a real bcrypt hash only when the email exists would let a caller distinguish "no such email" from "wrong password" by response time, even though both return the same 401 body. Fixed with a fixed dummy hash compared against on every login attempt where the email doesn't exist, keeping the timing constant.
- The frontend `request()` helper unconditionally calls `res.json()`; logout returns `204 No Content` with no body, which would throw. `request()` gains a one-line 204 short-circuit.
- `router.test.tsx`'s existing test navigates straight to `/claims` with no session; the new `beforeLoad` guard needs `getHealth` mocked in that file too, or the test breaks.
- The spec's "role attached to context, no route enforces it yet" is implemented as a context key with no accessor function — an unused accessor (nothing reads it until sub-project 2) would fail `golangci-lint`'s unused-code check. The `roleKey{}` type itself has real usage (constructing values with it), so it's not flagged.
- The Frontend section of the spec didn't include a way to trigger logout from the UI. Added: a "Sign out" button in `Shell.tsx`'s existing `SideNav` `footer` slot, using the already-planned `useLogout()` hook — without it, `POST /api/auth/logout` would ship with nothing in the UI ever calling it.

---

### Task 1: Migration and the bcrypt dependency

**Files:**
- Create: `internal/store/migrations/00006_identity.sql`
- Modify: `go.mod`, `go.sum` (via `go mod tidy` after Task 4 adds the import — this task only stages readiness; see Step 3)

**Interfaces:**
- Produces: tables `user_credentials(principal_id, email, password_hash, role, created_at)`, `sessions(id, principal_id, token_hash, created_at, expires_at)`, `login_attempts(id, email, succeeded, recorded_at)`. Task 3 consumes these exact table/column names — do not rename after this task.

- [ ] **Step 1: Write the migration**

Create `internal/store/migrations/00006_identity.sql`:

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
-- token_hash, never the raw token -- the same reason claim content is never
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

- [ ] **Step 2: Apply it against the dev database**

Run: `docker compose up -d db` (if not already running)
Run: `go build -o cred . && ./cred migrate`
Expected: `migrate    applied 1 migration(s): 0 -> 6` (or similar — the important part is no error, and the reported version is `6`).

Verify structurally:
Run: `docker compose exec db psql -U cred -d cred -c '\d user_credentials' -c '\d sessions' -c '\d login_attempts'`
Expected: all three tables described with the columns above, no errors.

- [ ] **Step 3: Stage the bcrypt dependency**

Run: `go get golang.org/x/crypto/bcrypt`
Expected: `go.mod` now lists `golang.org/x/crypto` as a direct dependency (no `// indirect` suffix). `go.sum` is unchanged in content (the checksum was already present via the module graph) or gains no new entries beyond what's already there.

This step only stages the dependency; nothing imports it yet until Task 4. That's expected — `go build ./...` still succeeds because an unused-but-required module is not a compile error in Go, only an unused *import* is.

- [ ] **Step 4: Commit**

```bash
git add internal/store/migrations/00006_identity.sql go.mod go.sum
git commit -m "feat: identity schema -- activate principals with credentials, sessions, login attempts

New migration adds user_credentials, sessions, and login_attempts,
all keyed to the existing (and previously dormant) principals
table rather than a parallel users table. Stages golang.org/x/crypto
for the next task's bcrypt use."
```

---

### Task 2: `internal/limit` — `LoginAttempts`

**Files:**
- Modify: `internal/limit/limit.go`
- Modify: `internal/limit/limit_test.go`

**Interfaces:**
- Produces: `Config.MaxLoginAttempts int`, `Config.LoginWindow time.Duration`, `ReasonLoginAttempts Reason`, `func LoginAttempts(failedInWindow int, cfg Config) Decision`. Task 4 calls this exact function signature — do not rename.

- [ ] **Step 1: Write the failing test**

In `internal/limit/limit_test.go`, add after `TestRecallRateBoundary`:

```go
func TestLoginAttemptsBoundary(t *testing.T) {
	cfg := Config{MaxLoginAttempts: 3}
	if d := LoginAttempts(2, cfg); !d.Allowed || d.Remaining != 1 {
		t.Fatalf("one below the ceiling must be allowed with one remaining: %+v", d)
	}
	if d := LoginAttempts(3, cfg); d.Allowed || d.Reason != ReasonLoginAttempts {
		t.Fatalf("at the ceiling must deny: %+v", d)
	}
}

func TestLoginAttemptsDisabledIsUnlimited(t *testing.T) {
	got := LoginAttempts(1_000_000, Config{MaxLoginAttempts: 0})
	if !got.Allowed || got.Remaining != -1 {
		t.Fatalf("a non-positive ceiling must disable the control: got %+v", got)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/limit/... -run TestLoginAttempts -v`
Expected: FAIL — `undefined: LoginAttempts` (and `undefined: ReasonLoginAttempts`, `unknown field MaxLoginAttempts`).

- [ ] **Step 3: Add the Config fields, the Reason, and the function**

In `internal/limit/limit.go`, change the `Reason` block:

```go
const (
	ReasonNone              Reason = ""
	ReasonContributionQuota Reason = "contribution_quota"
	ReasonCostCeiling       Reason = "cost_ceiling"
	ReasonRecallRate        Reason = "recall_rate"
)
```

to:

```go
const (
	ReasonNone              Reason = ""
	ReasonContributionQuota Reason = "contribution_quota"
	ReasonCostCeiling       Reason = "cost_ceiling"
	ReasonRecallRate        Reason = "recall_rate"
	ReasonLoginAttempts     Reason = "login_attempts"
)
```

Add to the `Config` struct, after the `RecallRate`/`RecallWindow`/`MaxPackageClaims` block and before the `ScopeClaimCeiling` block:

```go
	// Login attempts: failed logins per email per window. The brute-force /
	// credential-stuffing defence on the one unauthenticated write path.
	MaxLoginAttempts int
	LoginWindow      time.Duration
```

Add to `Defaults()`, after the `RecallRate`/`RecallWindow`/`MaxPackageClaims` block:

```go
		// Ten failed attempts per email per fifteen minutes: well above a
		// human mistyping a password twice, well below a scripted attempt
		// loop.
		MaxLoginAttempts: 10,
		LoginWindow:      15 * time.Minute,
```

Add the function, after `RecallRate`:

```go
// LoginAttempts decides whether one more login attempt for this email may
// proceed, given how many have failed in the current window. Only failures
// count toward the ceiling -- a user who logs in correctly every day must
// never be at risk of tripping a limit meant for attackers.
func LoginAttempts(failedInWindow int, cfg Config) Decision {
	return atMost(failedInWindow, cfg.MaxLoginAttempts, ReasonLoginAttempts)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/limit/... -v`
Expected: PASS, every test including the two new ones and `TestDefaultsAreUsable` (unaffected — it doesn't check login attempts, and the new default doesn't break any existing assertion).

- [ ] **Step 5: Commit**

```bash
git add internal/limit/limit.go internal/limit/limit_test.go
git commit -m "feat: add LoginAttempts to internal/limit

A fourth pure decision function alongside Contribution/Cost/
RecallRate, same atMost shape. Backs the login rate limit added
in the next task."
```

---

### Task 3: `internal/store/pg` — identity storage

**Files:**
- Modify: `internal/store/pg/pg.go`
- Create: `internal/store/pg/identity.go`

**Interfaces:**
- Consumes: `internal/claim.PrincipalID` (unchanged).
- Produces: `pg.ErrEmailTaken`; `(*Store).CreateUser`, `.UserCount`, `.CredentialsByEmail`, `.CreateSession`, `.SessionPrincipal`, `.DeleteSession`, `.RecordLoginAttempt`, `.FailedLoginsInWindow` — exact signatures below. Task 4 calls these — do not rename.

- [ ] **Step 1: Add the `ErrEmailTaken` sentinel and translate() case**

In `internal/store/pg/pg.go`, change:

```go
// ErrExtensionMissing reports that pgvector is not installed. It is separate
// because it is the failure a new user hits most often and it has a specific,
// printable fix.
var ErrExtensionMissing = errors.New("cred: the pgvector extension is not installed")
```

to:

```go
// ErrExtensionMissing reports that pgvector is not installed. It is separate
// because it is the failure a new user hits most often and it has a specific,
// printable fix.
var ErrExtensionMissing = errors.New("cred: the pgvector extension is not installed")

// ErrEmailTaken reports a registration attempt against an email that already
// has an account.
var ErrEmailTaken = errors.New("cred: email already registered")
```

Then change the `translate` switch:

```go
		switch {
		case pgErr.Code == "42704" && strings.Contains(pgErr.Message, "vector"),
			pgErr.Code == "42883" && strings.Contains(pgErr.Message, "halfvec"):
			return ErrExtensionMissing
		}
```

to:

```go
		switch {
		case pgErr.Code == "42704" && strings.Contains(pgErr.Message, "vector"),
			pgErr.Code == "42883" && strings.Contains(pgErr.Message, "halfvec"):
			return ErrExtensionMissing
		case pgErr.Code == "23505" && pgErr.ConstraintName == "user_credentials_email_key":
			return ErrEmailTaken
		}
```

- [ ] **Step 2: Write the store methods**

Create `internal/store/pg/identity.go`:

```go
package pg

import (
	"context"
	"time"

	"github.com/canhta/cred/internal/claim"
)

// This file activates the principals table: a human account is a principals
// row (kind='user') plus the credentials that don't belong on team/org/agent
// principals. The store returns rows and counters here, exactly as it does
// everywhere else in this package -- internal/limit decides what a counter
// means, this file only produces and stores them.

// CreateUser inserts a new principal (kind='user') and its credentials in one
// transaction. The principal id is a generated UUID, never the email: an
// email change must not be an identity change for every claim already
// attributed to this principal.
func (s *Store) CreateUser(ctx context.Context, email, passwordHash, role string) (claim.PrincipalID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", translate(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var principalID string
	err = tx.QueryRow(ctx, `
		INSERT INTO principals (id, kind, display_name)
		VALUES (gen_random_uuid()::text, 'user', $1)
		RETURNING id`, email).Scan(&principalID)
	if err != nil {
		return "", translate(err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_credentials (principal_id, email, password_hash, role)
		VALUES ($1, $2, $3, $4)`, principalID, email, passwordHash, role)
	if err != nil {
		return "", translate(err)
	}

	return claim.PrincipalID(principalID), translate(tx.Commit(ctx))
}

// UserCount reports how many accounts exist. The register handler decides
// what to do with the count (open until the first account, closed after);
// this method only counts.
func (s *Store) UserCount(ctx context.Context) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM user_credentials`).Scan(&n)
	return n, translate(err)
}

// CredentialsByEmail looks up a principal's credentials for login.
// ErrNotFound (translated) means no account has this email.
func (s *Store) CredentialsByEmail(ctx context.Context, email string) (
	principalID claim.PrincipalID, passwordHash string, role string, err error,
) {
	var pid string
	err = s.pool.QueryRow(ctx, `
		SELECT principal_id, password_hash, role FROM user_credentials WHERE email = $1`,
		email).Scan(&pid, &passwordHash, &role)
	if err != nil {
		return "", "", "", translate(err)
	}
	return claim.PrincipalID(pid), passwordHash, role, nil
}

// CreateSession inserts a new session row. tokenHash is the SHA-256 of the
// raw session token -- the raw token is a bearer credential and is never
// written to the database, the same reason claim content is never logged.
func (s *Store) CreateSession(ctx context.Context, principal claim.PrincipalID, tokenHash string, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (principal_id, token_hash, expires_at)
		VALUES ($1, $2, $3)`, string(principal), tokenHash, expiresAt)
	return translate(err)
}

// SessionPrincipal resolves a session token hash to the principal and role it
// belongs to. The query filters expires_at itself rather than returning a row
// the caller must re-check, matching the half-open interval convention
// internal/temporal uses elsewhere. ErrNotFound (translated) covers both an
// absent and an expired session -- indistinguishable to the caller, exactly
// as an authorization failure and a nonexistent row are elsewhere in this
// codebase.
func (s *Store) SessionPrincipal(ctx context.Context, tokenHash string, now time.Time) (
	principal claim.PrincipalID, role string, err error,
) {
	var pid string
	err = s.pool.QueryRow(ctx, `
		SELECT s.principal_id, coalesce(uc.role, '')
		  FROM sessions s
		  LEFT JOIN user_credentials uc ON uc.principal_id = s.principal_id
		 WHERE s.token_hash = $1 AND s.expires_at > $2`,
		tokenHash, now).Scan(&pid, &role)
	if err != nil {
		return "", "", translate(err)
	}
	return claim.PrincipalID(pid), role, nil
}

// DeleteSession removes a session row -- logout.
func (s *Store) DeleteSession(ctx context.Context, tokenHash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	return translate(err)
}

// RecordLoginAttempt appends one row to the login-attempt ledger. Plain
// INSERT, no read-modify-write, mirroring RecordUsage's append-only shape.
func (s *Store) RecordLoginAttempt(ctx context.Context, email string, succeeded bool, now time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO login_attempts (email, succeeded, recorded_at)
		VALUES ($1, $2, $3)`, email, succeeded, now)
	return translate(err)
}

// FailedLoginsInWindow counts an email's failed attempts since the cutoff --
// the counter internal/limit.LoginAttempts decides over. This method never
// compares the count to a ceiling itself, the same discipline every other
// counter in this package follows.
func (s *Store) FailedLoginsInWindow(ctx context.Context, email string, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM login_attempts
		 WHERE email = $1 AND succeeded = false AND recorded_at > $2`,
		email, since).Scan(&n)
	return n, translate(err)
}
```

- [ ] **Step 3: Build, vet, and run the existing test suite**

Run: `task api:build`
Expected: exits 0.

Run: `go vet ./...`
Expected: exits 0.

Run: `task api:test`
Expected: `ok` for every package with tests (including the two new `internal/limit` cases from Task 2); `internal/store/pg` still reports `[no test files]`, consistent with this codebase's precedent (see Global Constraints).

- [ ] **Step 4: Commit**

```bash
git add internal/store/pg/pg.go internal/store/pg/identity.go
git commit -m "feat: identity store methods -- create/lookup users, sessions, login attempts

Activates the principals table via a new identity.go, following
the existing package's shape: plain counters and inserts, no
principal-taking decision methods."
```

---

### Task 4: `internal/api` — register, login, logout

**Files:**
- Modify: `internal/api/types.go`
- Modify: `internal/api/api.go`
- Create: `internal/api/auth.go`

**Interfaces:**
- Consumes: Task 2's `limit.LoginAttempts`, `limit.WindowStart`; Task 3's eight store methods; `pg.ErrEmailTaken`, `pg.ErrNotFound`.
- Produces: `POST /api/auth/register`, `POST /api/auth/login`, `POST /api/auth/logout`; `HealthResponse.RegistrationOpen`. Task 5 consumes these exact routes and the exact JSON field names below — do not rename.

- [ ] **Step 1: Add the wire types**

In `internal/api/types.go`, change:

```go
// HealthResponse reports liveness and the resolved caller identity, so the
// console can show who it is acting as before any data loads.
type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Principal string `json:"principal"`
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
	RegistrationOpen bool   `json:"registration_open"`
}
```

Append to the end of the file:

```go

// RegisterRequest is the body of POST /api/auth/register.
type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// LoginRequest is the body of POST /api/auth/login.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse is the body of a successful register or login.
type AuthResponse struct {
	Principal string `json:"principal"`
	Role      string `json:"role"`
}
```

- [ ] **Step 2: Write the auth handlers**

Create `internal/api/auth.go`:

```go
package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/limit"
	"github.com/canhta/cred/internal/store/pg"
)

const (
	sessionCookieName = "cred_session"
	sessionLifetime   = 30 * 24 * time.Hour
)

// dummyHash is compared against when an email does not exist, so a login
// attempt takes the same time whether or not the email is registered --
// closing the timing side-channel the identical 401 response already closes
// at the response-body level.
var dummyHash []byte

func init() {
	h, err := bcrypt.GenerateFromPassword([]byte("cred-dummy-password-for-timing"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("api: dummy bcrypt hash: %v", err))
	}
	dummyHash = h
}

// hashToken returns the SHA-256 hex digest of a raw session token. Sessions
// store only this -- the raw token is a bearer credential and is never
// written to the database.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// newSessionToken generates a fresh 32-byte random token and its hash.
func newSessionToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashToken(raw), nil
}

// setSessionCookie writes the session cookie. Secure is always true:
// browsers treat localhost/127.0.0.1 as a secure context even over plain
// HTTP, and cred web always serves the API and the SPA from the same origin,
// so this never breaks local dev.
func setSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	c.SetCookieData(&http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(c *gin.Context) {
	c.SetCookieData(&http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// startSession creates a session for principal, sets the cookie, and writes
// the response. Shared by register and login so the two paths cannot drift.
func (s *server) startSession(c *gin.Context, principal claim.PrincipalID, role string) {
	token, hash, err := newSessionToken()
	if err != nil {
		s.fail(c, err)
		return
	}
	expiresAt := time.Now().UTC().Add(sessionLifetime)
	if err := s.store.CreateSession(c.Request.Context(), principal, hash, expiresAt); err != nil {
		s.fail(c, err)
		return
	}
	setSessionCookie(c, token, expiresAt)
	c.JSON(http.StatusOK, AuthResponse{Principal: string(principal), Role: role})
}

// register creates the first account as admin and closes registration for
// every account after it -- until the invite flow ships, closed registration
// is the only way a second account can exist.
func (s *server) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	ctx := c.Request.Context()
	count, err := s.store.UserCount(ctx)
	if err != nil {
		s.fail(c, err)
		return
	}
	if count > 0 {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "registration is closed"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.fail(c, err)
		return
	}

	// The first account is always admin, regardless of what the request
	// carried -- a client-supplied role on the only unauthenticated write
	// path this feature adds is exactly the privilege-escalation seam this
	// feature exists to close.
	principal, err := s.store.CreateUser(ctx, req.Email, string(hash), "admin")
	if err != nil {
		if errors.Is(err, pg.ErrEmailTaken) {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "email already registered"})
			return
		}
		s.fail(c, err)
		return
	}

	s.startSession(c, principal, "admin")
}

// login rate-limits by email before touching bcrypt: a bcrypt compare is
// deliberately slow, and running it before the rate check would let an
// attacker burn server CPU right up to the limit on every window.
func (s *server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	ctx := c.Request.Context()
	now := time.Now().UTC()
	lc := s.cfg.Limits

	failed, err := s.store.FailedLoginsInWindow(ctx, req.Email, limit.WindowStart(now, lc.LoginWindow))
	if err != nil {
		s.fail(c, err)
		return
	}
	if !limit.LoginAttempts(failed, lc).Allowed {
		c.JSON(http.StatusTooManyRequests,
			ErrorResponse{Error: "too many login attempts, try again later"})
		return
	}

	principal, hash, role, err := s.store.CredentialsByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, pg.ErrNotFound) {
		s.fail(c, err)
		return
	}

	hashToCompare := dummyHash
	if err == nil {
		hashToCompare = []byte(hash)
	}
	match := bcrypt.CompareHashAndPassword(hashToCompare, []byte(req.Password)) == nil
	valid := err == nil && match

	if recErr := s.store.RecordLoginAttempt(ctx, req.Email, valid, now); recErr != nil {
		s.fail(c, recErr)
		return
	}

	// A missing email and a wrong password fail identically -- an email-
	// existence oracle is the same failure class getClaim's 404 already
	// avoids for authorization.
	if !valid {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
		return
	}

	s.startSession(c, principal, role)
}

func (s *server) logout(c *gin.Context) {
	if cookie, err := c.Cookie(sessionCookieName); err == nil {
		_ = s.store.DeleteSession(c.Request.Context(), hashToken(cookie))
	}
	clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}
```

- [ ] **Step 3: Wire the session-cookie path into `authenticate()` and register the routes**

In `internal/api/api.go`, change the imports:

```go
import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/mcpsrv"
	"github.com/canhta/cred/internal/recall"
	"github.com/canhta/cred/internal/store/pg"
)
```

No import changes needed here — `pg` is already imported. Now change `New`:

```go
func New(st *pg.Store, emb recall.Embedder, count recall.TokenCounter,
	cfg config.Config, log *slog.Logger,
) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(requestLogger(log), gin.Recovery(), authenticate(cfg))

	s := &server{store: st, embedder: emb, count: count, cfg: cfg, log: log}
	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		api.GET("/claims", s.listClaims)
		api.GET("/claims/:id", s.getClaim)
		api.GET("/recall", s.recall)
		api.GET("/usage", s.usage)
	}
	return r
}
```

to:

```go
func New(st *pg.Store, emb recall.Embedder, count recall.TokenCounter,
	cfg config.Config, log *slog.Logger,
) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(requestLogger(log), gin.Recovery(), authenticate(cfg, st))

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

Add the role context key, next to `principalKey`:

```go
// principalKey is the context key the auth middleware stores the resolved
// principal under. A private type keeps it from colliding with any other
// package's context values.
type principalKey struct{}
```

to:

```go
// principalKey is the context key the auth middleware stores the resolved
// principal under. A private type keeps it from colliding with any other
// package's context values.
type principalKey struct{}

// roleKey is the context key for the resolved session's role, when a session
// (not a header) supplied the principal. Attached now so a later route guard
// can read it without a second query; nothing reads it yet.
type roleKey struct{}
```

Change `authenticate`:

```go
// authenticate resolves the principal for every request and, when a token is
// configured, gates access on it. This is the whole authentication seam:
// replacing it with OIDC/SSO later touches no handler, because handlers read
// the principal the middleware put on the context, not the header.
func authenticate(cfg config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.WebToken != "" {
			const prefix = "Bearer "
			auth := c.GetHeader("Authorization")
			if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix ||
				auth[len(prefix):] != cfg.WebToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized,
					ErrorResponse{Error: "unauthorized"})
				return
			}
		}

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
// authenticate resolves the principal for every request and, when a token is
// configured, gates access on it. A session cookie is checked first and, when
// valid, is authoritative -- its principal came from a verified login, not a
// client-supplied header, so it is never overridden by one. Replacing this
// with OIDC/SSO later touches no handler, because handlers read the
// principal the middleware put on the context, not the header or the cookie.
func authenticate(cfg config.Config, store *pg.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.WebToken != "" {
			const prefix = "Bearer "
			auth := c.GetHeader("Authorization")
			if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix ||
				auth[len(prefix):] != cfg.WebToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized,
					ErrorResponse{Error: "unauthorized"})
				return
			}
		}

		// An invalid or expired cookie falls through to the header/config-
		// default path below rather than failing the request outright -- a
		// stale cookie from a rotated session must not lock out a caller
		// that also has a valid bearer token configured.
		if cookie, err := c.Cookie(sessionCookieName); err == nil {
			principal, role, serr := store.SessionPrincipal(c.Request.Context(), hashToken(cookie), time.Now().UTC())
			if serr == nil {
				ctx := context.WithValue(c.Request.Context(), principalKey{}, principal)
				ctx = context.WithValue(ctx, roleKey{}, role)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}
		}

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

Change `health` to report registration status:

```go
func (s *server) health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "ok",
		Version:   mcpsrv.Version,
		Principal: string(principalFrom(c)),
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
		RegistrationOpen: count == 0,
	})
}
```

- [ ] **Step 4: Tidy modules, build, vet, lint, test**

Run: `go mod tidy`
Expected: exits 0; `golang.org/x/crypto` now appears without `// indirect` in `go.mod` (bcrypt is imported by `auth.go` now).

Run: `task api:build`
Expected: exits 0.

Run: `go vet ./...`
Expected: exits 0.

Run: `task api:lint`
Expected: exits 0 — this is the step that would catch an unused `roleKey` if it weren't constructed anywhere; it is (in `authenticate`), so this must pass clean.

Run: `task api:test`
Expected: `ok` for every package with tests, unchanged from Task 3's run.

- [ ] **Step 5: Commit**

```bash
git add internal/api/types.go internal/api/api.go internal/api/auth.go go.mod go.sum
git commit -m "feat: add register/login/logout endpoints

authenticate() gains a session-cookie path that is authoritative
over the header when present. Login rate-limits by email before
touching bcrypt, and closes the email-existence timing side-channel
with a dummy-hash comparison. The first registration becomes admin;
every registration after it is closed until the invite flow ships."
```

---

### Task 5: Frontend API layer

**Files:**
- Modify (generated): `web/src/api/types.gen.ts`
- Modify: `web/src/api/types.ts`
- Modify: `web/src/api/routes.ts`
- Modify: `web/src/api/client.ts`
- Modify: `web/src/api/hooks.ts`
- Modify: `web/src/api/index.ts`

**Interfaces:**
- Consumes: Task 4's Go types (`RegisterRequest`, `LoginRequest`, `AuthResponse`, updated `HealthResponse`).
- Produces: `register(params: RegisterRequest): Promise<AuthResponse>`, `login(params: LoginRequest): Promise<AuthResponse>`, `logout(): Promise<void>` (`client.ts`); `useRegister()`, `useLogin()`, `useLogout()` (`hooks.ts`), all re-exported from `web/src/api/index.ts`. Task 6 imports these — do not rename.

- [ ] **Step 1: Regenerate the TypeScript types**

Run: `task gen:types`
Expected: exits 0. `web/src/api/types.gen.ts` now has `RegisterRequest`, `LoginRequest`, and `AuthResponse` interfaces, and `HealthResponse` gains `registration_open: boolean`.

Verify: `grep -c "interface AuthResponse" web/src/api/types.gen.ts`
Expected: `1`

- [ ] **Step 2: Re-export the new types**

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
  LimitStatus,
  ScopeCost,
  ScopeGrowth,
  RegisterRequest,
  LoginRequest,
  AuthResponse,
} from './types.gen';
```

- [ ] **Step 3: Add the routes**

In `web/src/api/routes.ts`, change:

```ts
  recall: () => `${API_BASE}/recall`,
  usage: () => `${API_BASE}/usage`,
} as const;
```

to:

```ts
  recall: () => `${API_BASE}/recall`,
  usage: () => `${API_BASE}/usage`,
  register: () => `${API_BASE}/auth/register`,
  login: () => `${API_BASE}/auth/login`,
  logout: () => `${API_BASE}/auth/logout`,
} as const;
```

- [ ] **Step 4: Fix `request()` for a 204 response, and add the client functions**

In `web/src/api/client.ts`, change the type import:

```ts
import type {
  ClaimDetail,
  ClaimList,
  Health,
  RecallResponse,
  UsageResponse,
} from './types';
```

to:

```ts
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

Change `request`:

```ts
async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: {
      Accept: 'application/json',
      'X-CRED-Principal': currentPrincipal,
      ...init?.headers,
    },
  });

  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new ApiError(res.status, res.statusText, body);
  }

  return (await res.json()) as T;
}
```

to:

```ts
async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const res = await fetch(url, {
    ...init,
    headers: {
      Accept: 'application/json',
      'X-CRED-Principal': currentPrincipal,
      ...init?.headers,
    },
  });

  if (!res.ok) {
    const body = await res.text().catch(() => '');
    throw new ApiError(res.status, res.statusText, body);
  }

  // logout returns 204 with no body; res.json() would throw on an empty
  // response.
  if (res.status === 204) {
    return undefined as T;
  }

  return (await res.json()) as T;
}
```

Append to the end of the file:

```ts

function postJSON<T>(url: string, body: unknown): Promise<T> {
  return request<T>(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  });
}

export function register(params: RegisterRequest): Promise<AuthResponse> {
  return postJSON<AuthResponse>(routes.register(), params);
}

export function login(params: LoginRequest): Promise<AuthResponse> {
  return postJSON<AuthResponse>(routes.login(), params);
}

export function logout(): Promise<void> {
  return request<void>(routes.logout(), { method: 'POST' });
}
```

Also in `client.ts`, update the now-outdated comment above `currentPrincipal`. Change:

```ts
// The principal is a seam: today it rides on a header, later an OIDC/SSO
// middleware replaces the source without any handler changing. It is settable
// so the console can act as a different principal without a rebuild.
let currentPrincipal = 'local';
```

to:

```ts
// X-CRED-Principal remains the identity source for the header/bearer-token
// path (automation, testing, MCP) -- it is not used for the browser session
// path added in this feature, where authenticate() resolves the principal
// from a verified session cookie and a client-supplied header cannot
// override it. Settable so a script can act as a different principal
// without a rebuild.
let currentPrincipal = 'local';
```

- [ ] **Step 5: Add the mutation hooks**

In `web/src/api/hooks.ts`, change:

```ts
import { useQuery } from '@tanstack/react-query';
import { getClaim, getClaims, getHealth, getRecall, getUsage } from './client';
import type { ClaimsParams, RecallParams, UsageParams } from './client';
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
  login,
  logout,
  register,
} from './client';
import type { ClaimsParams, RecallParams, UsageParams } from './client';
```

Append to the end of the file:

```ts

export function useRegister() {
  return useMutation({ mutationFn: register });
}

export function useLogin() {
  return useMutation({ mutationFn: login });
}

export function useLogout() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: logout,
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.health });
    },
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

- [ ] **Step 7: Typecheck**

Run: `cd web && npm run typecheck`
Expected: exits 0.

- [ ] **Step 8: Commit**

```bash
git add web/src/api/
git commit -m "feat: wire register/login/logout into the frontend API layer

useRegister/useLogin/useLogout follow the mutation-hook pattern;
request() gains a 204 short-circuit for logout's empty body.
Types regenerated with task gen:types."
```

---

### Task 6: `LoginPage` and `RegisterPage`

**Files:**
- Create: `web/src/pages/Login.test.tsx`
- Create: `web/src/pages/Register.test.tsx`
- Create: `web/src/pages/LoginPage.tsx`
- Create: `web/src/pages/RegisterPage.tsx`

**Interfaces:**
- Consumes: `useLogin`, `useRegister`, `useHealth`, `ApiError` (Task 5, `web/src/api`).
- Produces: `export function LoginPage({ onSuccess, onNavigateToRegister }: { onSuccess: () => void; onNavigateToRegister: () => void })`; `export function RegisterPage({ onSuccess, onNavigateToLogin }: { onSuccess: () => void; onNavigateToLogin: () => void })`. Task 7 wires these into the router with real navigation — do not rename the props.

- [ ] **Step 1: Write the failing tests**

Create `web/src/pages/Login.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { LoginPage } from './LoginPage';
import { login, ApiError } from '../api/client';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, login: vi.fn() };
});

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('LoginPage', () => {
  const onSuccess = vi.fn();
  const onNavigateToRegister = vi.fn();

  beforeEach(() => {
    onSuccess.mockClear();
    onNavigateToRegister.mockClear();
  });

  it('calls onSuccess after a successful login', async () => {
    vi.mocked(login).mockResolvedValue({ principal: 'p1', role: 'admin' });
    const user = userEvent.setup();
    renderWithClient(
      <LoginPage
        onSuccess={onSuccess}
        onNavigateToRegister={onNavigateToRegister}
      />,
    );

    await user.type(screen.getByLabelText('Email'), 'a@b.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByText('Sign in'));

    expect(login).toHaveBeenCalledWith({
      email: 'a@b.com',
      password: 'password123',
    });
    expect(onSuccess).toHaveBeenCalled();
  });

  it('shows an inline error on invalid credentials, not a full-page failure', async () => {
    vi.mocked(login).mockRejectedValue(new ApiError(401, 'Unauthorized', ''));
    const user = userEvent.setup();
    renderWithClient(
      <LoginPage
        onSuccess={onSuccess}
        onNavigateToRegister={onNavigateToRegister}
      />,
    );

    await user.type(screen.getByLabelText('Email'), 'a@b.com');
    await user.type(screen.getByLabelText('Password'), 'wrong');
    await user.click(screen.getByText('Sign in'));

    expect(
      await screen.findByText('Invalid email or password.'),
    ).toBeInTheDocument();
    expect(onSuccess).not.toHaveBeenCalled();
  });

  it('navigates to register when the link is clicked', async () => {
    const user = userEvent.setup();
    renderWithClient(
      <LoginPage
        onSuccess={onSuccess}
        onNavigateToRegister={onNavigateToRegister}
      />,
    );

    await user.click(screen.getByText('Need an account? Register'));
    expect(onNavigateToRegister).toHaveBeenCalled();
  });
});
```

Create `web/src/pages/Register.test.tsx`:

```tsx
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import type { ReactNode } from 'react';
import { RegisterPage } from './RegisterPage';
import { register, getHealth } from '../api/client';
import type { Health } from '../api';

vi.mock('../api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api/client')>();
  return { ...actual, register: vi.fn(), getHealth: vi.fn() };
});

const OPEN: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: '',
  registration_open: true,
};

const CLOSED: Health = { ...OPEN, registration_open: false };

function renderWithClient(ui: ReactNode) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>{ui}</QueryClientProvider>,
  );
}

describe('RegisterPage', () => {
  const onSuccess = vi.fn();
  const onNavigateToLogin = vi.fn();

  beforeEach(() => {
    onSuccess.mockClear();
    onNavigateToLogin.mockClear();
  });

  it('registers and calls onSuccess when registration is open', async () => {
    vi.mocked(getHealth).mockResolvedValue(OPEN);
    vi.mocked(register).mockResolvedValue({ principal: 'p1', role: 'admin' });
    const user = userEvent.setup();
    renderWithClient(
      <RegisterPage onSuccess={onSuccess} onNavigateToLogin={onNavigateToLogin} />,
    );

    await user.type(await screen.findByLabelText('Email'), 'a@b.com');
    await user.type(screen.getByLabelText('Password'), 'password123');
    await user.click(screen.getByText('Create account'));

    expect(register).toHaveBeenCalledWith({
      email: 'a@b.com',
      password: 'password123',
    });
    expect(onSuccess).toHaveBeenCalled();
  });

  it('redirects to login when registration is closed', async () => {
    vi.mocked(getHealth).mockResolvedValue(CLOSED);
    renderWithClient(
      <RegisterPage onSuccess={onSuccess} onNavigateToLogin={onNavigateToLogin} />,
    );

    expect(await screen.findByText(/registration is closed/i)).toBeInTheDocument();
  });
});
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd web && npx vitest run src/pages/Login.test.tsx src/pages/Register.test.tsx`
Expected: FAIL — `Cannot find module './LoginPage'` and `Cannot find module './RegisterPage'`.

- [ ] **Step 3: Write the pages**

Create `web/src/pages/LoginPage.tsx`:

```tsx
import { useState } from 'react';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { VStack } from '@astryxdesign/core/Stack';
import { Heading, Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Button } from '@astryxdesign/core/Button';
import { Banner } from '@astryxdesign/core/Banner';
import { Link } from '@astryxdesign/core/Link';
import { useLogin, ApiError } from '../api';

export function LoginPage({
  onSuccess,
  onNavigateToRegister,
}: {
  onSuccess: () => void;
  onNavigateToRegister: () => void;
}) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const login = useLogin();

  const submit = () => {
    login.mutate({ email, password }, { onSuccess });
  };

  const errorMessage =
    login.error instanceof ApiError
      ? login.error.status === 429
        ? 'Too many login attempts. Wait a moment and try again.'
        : 'Invalid email or password.'
      : null;

  return (
    <Center height="100%">
      <Card padding={4} width={360}>
        <VStack gap={4}>
          <VStack gap={1}>
            <Heading level={4}>Sign in</Heading>
            <Text type="body" color="secondary">
              CRED console
            </Text>
          </VStack>
          {errorMessage ? <Banner status="error" title={errorMessage} /> : null}
          <TextInput
            label="Email"
            type="email"
            value={email}
            onChange={setEmail}
            onEnter={submit}
            hasAutoFocus
          />
          <TextInput
            label="Password"
            type="password"
            value={password}
            onChange={setPassword}
            onEnter={submit}
          />
          <Button
            label="Sign in"
            variant="primary"
            onClick={submit}
            isLoading={login.isPending}
          />
          <Link
            href="/register"
            onClick={(e) => {
              e.preventDefault();
              onNavigateToRegister();
            }}
          >
            Need an account? Register
          </Link>
        </VStack>
      </Card>
    </Center>
  );
}
```

Create `web/src/pages/RegisterPage.tsx`:

```tsx
import { useState } from 'react';
import { Center } from '@astryxdesign/core/Center';
import { Card } from '@astryxdesign/core/Card';
import { VStack } from '@astryxdesign/core/Stack';
import { Heading, Text } from '@astryxdesign/core/Text';
import { TextInput } from '@astryxdesign/core/TextInput';
import { Button } from '@astryxdesign/core/Button';
import { Banner } from '@astryxdesign/core/Banner';
import { Link } from '@astryxdesign/core/Link';
import { EmptyState } from '@astryxdesign/core/EmptyState';
import { useHealth, useRegister, ApiError } from '../api';

export function RegisterPage({
  onSuccess,
  onNavigateToLogin,
}: {
  onSuccess: () => void;
  onNavigateToLogin: () => void;
}) {
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const health = useHealth();
  const register = useRegister();

  if (health.isLoading) {
    return null;
  }

  if (!health.data?.registration_open) {
    return (
      <Center height="100%">
        <EmptyState
          title="Registration is closed"
          description="An account already exists on this instance. Sign in instead."
        />
      </Center>
    );
  }

  const submit = () => {
    register.mutate({ email, password }, { onSuccess });
  };

  const errorMessage =
    register.error instanceof ApiError
      ? register.error.status === 409
        ? 'That email is already registered.'
        : 'Registration failed. Check your details and try again.'
      : null;

  return (
    <Center height="100%">
      <Card padding={4} width={360}>
        <VStack gap={4}>
          <VStack gap={1}>
            <Heading level={4}>Create the first account</Heading>
            <Text type="body" color="secondary">
              This account becomes the console's admin.
            </Text>
          </VStack>
          {errorMessage ? <Banner status="error" title={errorMessage} /> : null}
          <TextInput
            label="Email"
            type="email"
            value={email}
            onChange={setEmail}
            onEnter={submit}
            hasAutoFocus
          />
          <TextInput
            label="Password"
            type="password"
            description="At least 8 characters."
            value={password}
            onChange={setPassword}
            onEnter={submit}
          />
          <Button
            label="Create account"
            variant="primary"
            onClick={submit}
            isLoading={register.isPending}
          />
          <Link
            href="/login"
            onClick={(e) => {
              e.preventDefault();
              onNavigateToLogin();
            }}
          >
            Already have an account? Sign in
          </Link>
        </VStack>
      </Card>
    </Center>
  );
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd web && npx vitest run src/pages/Login.test.tsx src/pages/Register.test.tsx`
Expected: PASS, 5 tests (3 in `Login.test.tsx`, 2 in `Register.test.tsx`).

- [ ] **Step 5: Typecheck and lint**

Run: `cd web && npm run typecheck`
Expected: exits 0.

Run: `cd web && npm run lint`
Expected: exits 0.

- [ ] **Step 6: Commit**

```bash
git add web/src/pages/LoginPage.tsx web/src/pages/RegisterPage.tsx web/src/pages/Login.test.tsx web/src/pages/Register.test.tsx
git commit -m "feat: login and register pages

Pure components with injected navigation, matching the
ClaimsPage/ClaimDetailPage pattern -- router.tsx supplies the real
useNavigate() calls in the next task. RegisterPage checks
useHealth().registration_open and shows a closed state instead of
a form once an account exists."
```

---

### Task 7: Router restructuring, auth guard, and sign-out

**Files:**
- Modify: `web/src/router.tsx` (full rewrite of the route tree)
- Modify: `web/src/router.test.tsx` (mock `getHealth`)
- Modify: `web/src/app/Shell.tsx` (add sign-out)

**Interfaces:**
- Consumes: `LoginPage`, `RegisterPage` (Task 6); `getHealth` (existing); `useLogout` (Task 5).
- Produces: `/login` and `/register` routes outside the authenticated shell; every existing route now requires a session via a `beforeLoad` guard on a new pathless `appRoute`.

- [ ] **Step 1: Rewrite the router**

Read the current `web/src/router.tsx` first (needed because this step replaces the whole file, and the exact current content must be diffed against, not assumed).

Replace the full contents of `web/src/router.tsx` with:

```tsx
import {
  createRootRoute,
  createRoute,
  createRouter,
  redirect,
  useNavigate,
  Outlet,
} from '@tanstack/react-router';
import { Shell } from './app/Shell';
import { ClaimsPage } from './pages/ClaimsPage';
import { ClaimDetailPage } from './pages/ClaimDetailPage';
import { RecallPage } from './pages/RecallPage';
import { UsagePage } from './pages/UsagePage';
import { LoginPage } from './pages/LoginPage';
import { RegisterPage } from './pages/RegisterPage';
import { Placeholder } from './pages/Placeholder';
import { getHealth } from './api/client';

const rootRoute = createRootRoute({
  component: () => <Outlet />,
});

const loginRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/login',
  component: LoginRoute,
});

const registerRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/register',
  component: RegisterRoute,
});

function LoginRoute() {
  const navigate = useNavigate();
  return (
    <LoginPage
      onSuccess={() => navigate({ to: '/claims' })}
      onNavigateToRegister={() => navigate({ to: '/register' })}
    />
  );
}

function RegisterRoute() {
  const navigate = useNavigate();
  return (
    <RegisterPage
      onSuccess={() => navigate({ to: '/claims' })}
      onNavigateToLogin={() => navigate({ to: '/login' })}
    />
  );
}

// The authenticated app: every route inside here requires a session, checked
// once in beforeLoad rather than per-page, so an unauthenticated visitor
// never sees a protected page flash before the redirect. A pathless route
// (id, not path) so it adds a layout without adding a URL segment.
const appRoute = createRoute({
  getParentRoute: () => rootRoute,
  id: 'app',
  beforeLoad: async () => {
    const health = await getHealth();
    if (!health.principal) {
      throw redirect({ to: '/login' });
    }
  },
  component: () => (
    <Shell>
      <Outlet />
    </Shell>
  ),
});

const indexRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/',
  beforeLoad: () => {
    throw redirect({ to: '/claims' });
  },
});

const claimsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/claims',
  component: ClaimsRoute,
});

const claimDetailRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/claims/$id',
  component: ClaimDetailRoute,
});

// Navigation is injected into the pages rather than reached for with a router
// hook inside them, so ClaimsPage and ClaimDetailPage stay pure components the
// tests can render without a router context. These wrappers are the only place
// that seam is closed.
function ClaimsRoute() {
  const navigate = useNavigate();
  return (
    <ClaimsPage
      onOpen={(id) => navigate({ to: '/claims/$id', params: { id } })}
    />
  );
}

function ClaimDetailRoute() {
  const { id } = claimDetailRoute.useParams();
  const navigate = useNavigate();
  return (
    <ClaimDetailPage id={id} onBack={() => navigate({ to: '/claims' })} />
  );
}

const recallRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/recall',
  component: RecallPage,
});

const usageRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/usage',
  component: UsagePage,
});

const teamRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/team',
  component: () => <Placeholder title="Team" />,
});

const projectsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/projects',
  component: () => <Placeholder title="Projects" />,
});

const settingsRoute = createRoute({
  getParentRoute: () => appRoute,
  path: '/settings',
  component: () => <Placeholder title="Settings" />,
});

const routeTree = rootRoute.addChildren([
  loginRoute,
  registerRoute,
  appRoute.addChildren([
    indexRoute,
    claimsRoute,
    claimDetailRoute,
    recallRoute,
    usageRoute,
    teamRoute,
    projectsRoute,
    settingsRoute,
  ]),
]);

export const router = createRouter({ routeTree });

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router;
  }
}
```

- [ ] **Step 2: Fix the existing router test for the new auth guard**

In `web/src/router.test.tsx`, change:

```tsx
import { router } from './router';
import { getClaims, getClaim } from './api/client';
import type { ClaimList, ClaimDetail } from './api';

vi.mock('./api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api/client')>();
  return { ...actual, getClaims: vi.fn(), getClaim: vi.fn() };
});
```

to:

```tsx
import { router } from './router';
import { getClaims, getClaim, getHealth } from './api/client';
import type { ClaimList, ClaimDetail, Health } from './api';

vi.mock('./api/client', async (importOriginal) => {
  const actual = await importOriginal<typeof import('./api/client')>();
  return {
    ...actual,
    getClaims: vi.fn(),
    getClaim: vi.fn(),
    getHealth: vi.fn(),
  };
});

const AUTHENTICATED: Health = {
  status: 'ok',
  version: '0.1.0',
  principal: 'local',
  registration_open: false,
};
```

Then change:

```tsx
describe('claim navigation', () => {
  beforeEach(() => {
    vi.mocked(getClaims).mockResolvedValue(CLAIMS);
    vi.mocked(getClaim).mockResolvedValue(DETAIL);
  });
```

to:

```tsx
describe('claim navigation', () => {
  beforeEach(() => {
    vi.mocked(getClaims).mockResolvedValue(CLAIMS);
    vi.mocked(getClaim).mockResolvedValue(DETAIL);
    vi.mocked(getHealth).mockResolvedValue(AUTHENTICATED);
  });
```

- [ ] **Step 3: Add sign-out to the Shell**

In `web/src/app/Shell.tsx`, change the imports:

```tsx
import type { ReactNode } from 'react';
import { AppShell } from '@astryxdesign/core/AppShell';
import {
  SideNav,
  SideNavHeading,
  SideNavItem,
  SideNavSection,
} from '@astryxdesign/core/SideNav';
import { useNavigate, useRouterState } from '@tanstack/react-router';
```

to:

```tsx
import type { ReactNode } from 'react';
import { AppShell } from '@astryxdesign/core/AppShell';
import {
  SideNav,
  SideNavHeading,
  SideNavItem,
  SideNavSection,
} from '@astryxdesign/core/SideNav';
import { Button } from '@astryxdesign/core/Button';
import { useNavigate, useRouterState } from '@tanstack/react-router';
import { useLogout } from '../api';
```

Change the component body:

```tsx
export function Shell({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });

  return (
    <AppShell
      contentPadding={0}
      sideNav={
        <SideNav
          collapsible
          header={
            <SideNavHeading
              heading="CRED"
              subheading="Console"
              headingHref="/claims"
            />
          }
        >
```

to:

```tsx
export function Shell({ children }: { children: ReactNode }) {
  const navigate = useNavigate();
  const pathname = useRouterState({ select: (s) => s.location.pathname });
  const logout = useLogout();

  return (
    <AppShell
      contentPadding={0}
      sideNav={
        <SideNav
          collapsible
          header={
            <SideNavHeading
              heading="CRED"
              subheading="Console"
              headingHref="/claims"
            />
          }
          footer={
            <Button
              label="Sign out"
              variant="ghost"
              isLoading={logout.isPending}
              onClick={() => {
                logout.mutate(undefined, {
                  onSuccess: () => navigate({ to: '/login' }),
                });
              }}
            />
          }
        >
```

(The closing `</SideNav>` and everything below it is unchanged — only the opening tag gains the `footer` prop and the two new lines above it.)

- [ ] **Step 4: Run the full frontend suite, typecheck, and lint**

Run: `task web:test`
Expected: all suites pass, including the existing `router.test.tsx` (now passing `getHealth`) and the 5 new Login/Register tests from Task 6 — total should be 7 test files.

Run: `cd web && npm run typecheck`
Expected: exits 0.

Run: `cd web && npm run lint`
Expected: exits 0.

- [ ] **Step 5: Commit**

```bash
git add web/src/router.tsx web/src/router.test.tsx web/src/app/Shell.tsx
git commit -m "feat: gate the console behind a session, add sign-out

login/register live outside the authenticated layout; every
existing route moves under a new pathless appRoute whose beforeLoad
redirects to /login when getHealth() reports no principal. Shell
gets a Sign out control in SideNav's footer slot, since nothing
in the UI could reach POST /api/auth/logout otherwise."
```

---

### Task 8: End-to-end verification

This codebase's established practice (see the Usage-page plan's Task 4) is
manual end-to-end verification against a real Postgres-backed server. This
task follows it, plus a couple of API-level checks that are faster to run
directly than through the browser.

**Files:** none (verification only).

- [ ] **Step 1: Rebuild and confirm migration state**

Run: `go build -o cred . && ./cred migrate`
Expected: reports the schema already at version 6 (Task 1 applied it; this just confirms nothing drifted).

- [ ] **Step 2: Register the first account via curl and confirm it's admin**

Start the server on a port that won't collide with anything else running (check first: `lsof -i :8091 -sTCP:LISTEN || echo free`).

Run:
```sh
./cred web --addr :8091 > /tmp/cred-web-auth-verify.log 2>&1 &
sleep 2
curl -s -c /tmp/cred-cookies.txt -X POST localhost:8091/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"admin@example.com","password":"correcthorsebattery"}'
echo
```
Expected: `{"principal":"<uuid>","role":"admin"}`, and `/tmp/cred-cookies.txt` now contains a `cred_session` cookie.

Verify the database agrees:
```sh
docker compose exec db psql -U cred -d cred -c \
  "SELECT email, role FROM user_credentials;"
```
Expected: one row, `admin@example.com | admin`.

- [ ] **Step 3: Confirm registration is now closed**

Run:
```sh
curl -s -o /dev/null -w '%{http_code}\n' -X POST localhost:8091/api/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"second@example.com","password":"anotherpassword1"}'
```
Expected: `403`.

- [ ] **Step 4: Confirm the session cookie authenticates `/api/health`, then log out and confirm it stops**

Run:
```sh
curl -s -b /tmp/cred-cookies.txt localhost:8091/api/health
echo
curl -s -b /tmp/cred-cookies.txt -X POST localhost:8091/api/auth/logout -w '\n%{http_code}\n'
curl -s -b /tmp/cred-cookies.txt localhost:8091/api/health
echo
```
Expected: first call shows `"principal":"<the same uuid>"`; logout returns `204`; the call after logout shows `"principal":""` (the cookie is now invalid — deleted server-side and cleared client-side, though curl's saved cookie jar will still send the stale value until it expires, so `""` here confirms the *server* rejected it, not that curl stopped sending it).

Run: `kill %1` (or `lsof -ti:8091 -sTCP:LISTEN | xargs -r kill`) to stop the verification server.

- [ ] **Step 5: Verify login rate limiting**

Run:
```sh
for i in $(seq 1 11); do
  curl -s -o /dev/null -w '%{http_code} ' -X POST localhost:8091/api/auth/login \
    -H 'Content-Type: application/json' \
    -d '{"email":"admin@example.com","password":"wrong-password"}'
done
echo
```

(Restart the server first if Step 4 killed it: `./cred web --addr :8091 > /tmp/cred-web-auth-verify.log 2>&1 &`, `sleep 2`.)

Expected: the first 10 requests return `401`, the 11th returns `429` — matching `MaxLoginAttempts: 10` from Task 2's default.

Stop the server: `lsof -ti:8091 -sTCP:LISTEN | xargs -r kill`.

- [ ] **Step 6: Live browser check**

Run `task dev` in the background (check `:5173` and `:8080` are free first, same caution as the Usage-page plan — do not kill a process you don't recognize as yours; ask if either port is occupied by something unexplained).

Open `http://localhost:5173/claims` in a browser. Verify:
- Redirected to `/login` (no session yet in this browser).
- `/register` shows the create-account form (registration is still open in
  the dev database *unless* Step 2's curl registration already used it —
  if so, `/register` should show "Registration is closed" instead; either
  outcome is correct, confirm whichever one matches the current database
  state).
- Logging in with `admin@example.com` / `correcthorsebattery` (or
  registering fresh, if Step 2 wasn't run against this same database)
  lands on `/claims`.
- The SideNav shows a "Sign out" control; clicking it returns to `/login`.
- No errors in the browser console at any step.

Stop `task dev` (Ctrl-C) once verified.

- [ ] **Step 7: No commit for this task**

Verification-only. If any step surfaces a bug, fix it under the task where
it originated, re-run that task's checks, commit the fix there, and re-run
this task from Step 1.
