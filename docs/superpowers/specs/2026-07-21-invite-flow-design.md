# Invite flow — design

- **Date:** 2026-07-21
- **Status:** Approved by operator, pending implementation plan.
- **Decision:** `docs/research/decision-log.md` D-025 — multi-user auth and
  RBAC pulled into v1. This spec is the third of four sub-projects D-025
  names: identity & auth
  (`docs/superpowers/specs/2026-07-21-identity-auth-design.md`) → role model
  + RBAC/ACL integration
  (`docs/superpowers/specs/2026-07-21-role-enforcement-rbac-design.md`) →
  invite flow (this doc) → MCP auth. Only sub-project 3 is designed here.

## Problem

Registration is open exactly until the first account exists
(`POST /api/auth/register`, `internal/api/auth.go:107-153`), then closed —
every subsequent attempt gets `403 "registration is closed"`. There is no
way for a second account, admin or member, to exist except an operator
inserting rows directly via `psql` — which is what sub-project 2's manual
verification had to do
(`docs/superpowers/plans/2026-07-21-role-enforcement-rbac.md`, Task 5,
Step 3). This sub-project builds the missing path: an admin invites a
specific person, at a specific role, and that invite is how the account
gets created.

CRED has no SMTP or email-sending capability — sub-project 1's design
named password reset out of scope for exactly this reason
(`docs/superpowers/specs/2026-07-21-identity-auth-design.md`, "What's
deliberately out of scope"). An invite cannot be emailed by the system
itself; delivery has to be something the admin does out-of-band.

## A constraint this decision reopens

`user_credentials_one_admin` (`internal/store/migrations/00007_one_admin.sql`)
is a partial unique index enforcing at most one admin row, ever. This
sub-project's role model — an admin can invite another admin — requires
dropping it: admins are no longer capped.

Dropping it is not enough on its own. That index's own migration comment
states its real purpose: closing a registration race, not capping admin
count forever. *"Two concurrent first-registrations can both observe
UserCount() == 0 before either has committed, and the email UNIQUE
constraint only catches a same-email race — it does nothing when the two
requests use different emails."* `POST /api/auth/register`'s bootstrap path
(no invite, `count == 0` → admin) is still the one open, unauthenticated
write endpoint this race targets. Removing the constraint without replacing
it reopens exactly that race: two simultaneous unauthenticated requests
could each observe `count == 0` and both become admins through the bootstrap
path, which is not what "the first account becomes admin" is supposed to
mean once every *other* admin is created deliberately, through an invite.

The replacement keys off *how* an account was created, not *what role* it
holds. `user_credentials` gains a nullable `invited_by`: `NULL` for the
bootstrap path, set for every invite-created account. A new partial unique
index, `user_credentials_one_bootstrap`, enforces at most one row with
`invited_by IS NULL` — the same partial-unique-index technique the index it
replaces used, aimed at the actual race instead of a permanent cap.

## Schema

New migration `internal/store/migrations/00008_invites.sql`:

```sql
-- +goose Up
-- +goose StatementBegin

DROP INDEX IF EXISTS user_credentials_one_admin;

-- invited_by distinguishes how an account was created: NULL is the open
-- bootstrap path (register with no invite), non-null is an invite
-- redemption. This is what the replacement race-closer below keys on --
-- admin count itself is no longer capped.
ALTER TABLE user_credentials
    ADD COLUMN invited_by text REFERENCES principals(id);

-- Two concurrent bootstrap registrations can still both observe
-- UserCount() == 0 before either commits, the same race
-- user_credentials_one_admin used to close -- but that index capped
-- total admins, which this sub-project's role model no longer allows.
-- This closes the same race by capping bootstrap-created rows instead:
-- at most one account may ever be created without an invite.
CREATE UNIQUE INDEX user_credentials_one_bootstrap
    ON user_credentials ((invited_by IS NULL))
    WHERE invited_by IS NULL;

CREATE TABLE invites (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email        text        NOT NULL,
    role         text        NOT NULL CHECK (role IN ('admin', 'member')),
    token_hash   text        NOT NULL UNIQUE,
    invited_by   text        NOT NULL REFERENCES principals(id),
    created_at   timestamptz NOT NULL DEFAULT now(),
    expires_at   timestamptz NOT NULL,
    used_at      timestamptz,
    revoked_at   timestamptz
);

CREATE INDEX invites_token_hash_idx ON invites (token_hash);

-- Nothing in this sub-project ever updates or deletes a user_credentials
-- row -- there is no demote or remove capability yet. This trigger is
-- forward-looking scaffolding for when one exists: it is cheap to add now
-- and expensive to retrofit once a demote/remove path is live and already
-- shipped without this guard.
CREATE FUNCTION user_credentials_admin_floor() RETURNS trigger AS $$
BEGIN
    IF (SELECT count(*) FROM user_credentials WHERE role = 'admin') = 0 THEN
        RAISE EXCEPTION 'at least one admin must exist';
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE CONSTRAINT TRIGGER user_credentials_admin_floor_trigger
    AFTER UPDATE OR DELETE ON user_credentials
    FOR EACH ROW
    WHEN (OLD.role = 'admin')
    EXECUTE FUNCTION user_credentials_admin_floor();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS user_credentials_admin_floor_trigger ON user_credentials;
DROP FUNCTION IF EXISTS user_credentials_admin_floor();
DROP TABLE IF EXISTS invites;
DROP INDEX IF EXISTS user_credentials_one_bootstrap;
ALTER TABLE user_credentials DROP COLUMN IF EXISTS invited_by;
CREATE UNIQUE INDEX user_credentials_one_admin
    ON user_credentials ((role))
    WHERE role = 'admin';
-- +goose StatementEnd
```

`translate()` (`internal/store/pg/pg.go:93-120`) gains one more case: a
unique violation on `user_credentials_one_bootstrap` translates to a new
`ErrBootstrapExists` sentinel, replacing `ErrAdminExists` (which described a
constraint that no longer exists — same shape, correct name).

## Store layer

New file `internal/store/pg/invites.go`, following the existing package's
shape:

```go
func (s *Store) CreateInvite(ctx context.Context, email, role, tokenHash string,
	invitedBy claim.PrincipalID, expiresAt time.Time) (string, error)

func (s *Store) InviteByTokenHash(ctx context.Context, tokenHash string, now time.Time) (
	email, role string, invitedBy claim.PrincipalID, err error)

func (s *Store) MarkInviteUsed(ctx context.Context, id string, now time.Time) error

func (s *Store) RevokeInvite(ctx context.Context, id string, now time.Time) error

func (s *Store) PendingInvites(ctx context.Context, now time.Time) ([]Invite, error)
```

- `CreateInvite` — plain insert, returns the new invite's `id`.
- `InviteByTokenHash` — `ErrNotFound` (translated) covers absent, expired,
  used, and revoked alike: the query filters `used_at IS NULL AND
  revoked_at IS NULL AND expires_at > now` itself, the same half-open-
  interval convention `SessionPrincipal` already uses. Indistinguishable
  failure modes here are deliberate — see API surface, below.
- `MarkInviteUsed` / `RevokeInvite` — set the respective timestamp.
- `PendingInvites` — `used_at IS NULL AND revoked_at IS NULL AND expires_at
  > now`, ordered by `created_at`.
- `CreateInvitedUser(ctx, email, passwordHash, role string, invitedBy
  claim.PrincipalID) (claim.PrincipalID, error)` — one transaction: insert
  into `principals` and `user_credentials` (with `invited_by` set), same
  shape as the existing `CreateUser` but for the invite path.
- `TeamMembers(ctx) ([]Member, error)` — `SELECT principal_id, email, role,
  created_at FROM user_credentials ORDER BY created_at`.

## API surface

New route group under `requireAdmin()` (`internal/api/api.go:47-49`
already has this middleware from sub-project 2):

```go
admin.POST("/team/invites", s.createInvite)
admin.GET("/team/invites", s.listInvites)
admin.DELETE("/team/invites/:id", s.revokeInvite)
admin.GET("/team/members", s.teamMembers)
```

- `POST /api/team/invites` — `{email, role}`. Generates a token the same
  way sessions already do: 32 bytes from `crypto/rand`, base64url-encoded;
  only the SHA-256 hash (`hashToken`, already in `internal/api/auth.go:43-46`)
  is stored. Response: `{email, role, expires_at, token}` — the raw token is
  visible exactly once, here, matching the convention that a raw bearer
  value is never written to the database or logged. The frontend builds the
  link client-side (`${origin}/register?invite=${token}`); the server never
  needs to know its own public URL.
- `GET /api/team/invites` — pending invites only (not used, not revoked, not
  expired). Used/expired/revoked history is not surfaced this pass.
- `DELETE /api/team/invites/:id` — sets `revoked_at`.
- `GET /api/team/members` — every account: principal, email, role, joined.

`RegisterRequest` (`internal/api/types.go`) gains one optional field,
`Invite string`. `register()` (`internal/api/auth.go:107-153`) branches on
its presence:

- **Present**: look up by token hash. Any failure — not found, expired,
  used, revoked, or the submitted email doesn't match the invite's target
  email — collapses to the same `400 {"error": "invalid or expired
  invite"}`. Distinguishing these would let a caller learn whether a token
  is real, used, or bound to a different email than the one they tried —
  the same existence-oracle failure class `login`'s identical 401 already
  avoids (`internal/api/auth.go:198-202`). On success: `CreateInvitedUser`
  with the invite's role, `MarkInviteUsed`, start a session — same response
  shape as today.
- **Absent**: unchanged bootstrap path (`UserCount() == 0` → admin,
  otherwise `403 "registration is closed"`), except the race-closing
  sentinel changes: `errors.Is(err, pg.ErrBootstrapExists)` replaces
  `errors.Is(err, pg.ErrAdminExists)`, still mapping to the same `403
  "registration is closed"` response (indistinguishable to the client from
  registration having already closed, same as today).

No new rate limiting: invite creation is behind `requireAdmin()`, not an
unauthenticated surface like login or register.

## Frontend

- `Team` route (`web/src/router.tsx`) changes from `<Placeholder
  title="Team" />` to a real `TeamPage`, following `ClaimsPage`'s existing
  `Table`-based list pattern (`web/src/pages/ClaimsPage.tsx`): a "Members"
  table (email, role, joined) and a "Pending invites" table (email, role,
  expires, a revoke action), plus an "Invite" action — a small form (email,
  role) that, on success, shows the generated link with a copy affordance.
- `RegisterPage` (`web/src/pages/RegisterPage.tsx:24-36`) currently gates
  entirely on `health.data?.registration_open`, showing "registration is
  closed" once any account exists. `registerRoute` gains a typed search
  param, `invite`, read from the URL and passed to `RegisterPage`. When
  present, the form renders regardless of `registration_open`, and the
  token rides through to `useRegister()`'s mutation as the optional
  `invite` field. When absent, today's behavior is unchanged.
- New frontend API layer additions (`routes.ts` → `client.ts` → `hooks.ts`
  → `index.ts`), following the exact layering sub-project 2 used for
  `/api/usage/org`: `getTeamMembers`, `getInvites`, `createInvite`,
  `revokeInvite` and their corresponding hooks.

## What's deliberately out of scope for this pass

- **Configurable invite lifetime.** Hardcoded 7 days, same precedent as the
  session's hardcoded 30 days.
- **Invite history.** Used/expired/revoked invites aren't listed, only
  pending ones. Re-inviting the same email after expiry or use is just
  creating a new invite.
- **Anything that demotes or removes an admin.** The floor-invariant
  trigger has no caller yet — scaffolding for a future capability, not
  something this sub-project exercises end-to-end.
- **MCP picking up invited identities.** Sub-project 4, unaffected here.
- **Email delivery of any kind.** The admin is the delivery mechanism —
  CRED only ever produces the link.

## Testing

- `internal/store/pg/integration_test.go`: invite CRUD round-trip (create →
  fetch by token hash → mark used), expiry rejection, revoked rejection,
  the new `user_credentials_one_bootstrap` race-closer (mirroring
  `TestCreateUserSecondAdminReturnsErrAdminExists`'s existing shape at
  `internal/store/pg/integration_test.go:526-543`), and the admin-floor
  trigger (attempt to demote or delete the last admin directly via SQL in
  the test, confirm Postgres rejects it).
- No per-handler Go test file, per this codebase's established
  `internal/api` precedent.
- Frontend: `TeamPage.test.tsx` (mocked client) and an extension to
  `Register.test.tsx` covering the invite-param path.
- Manual end-to-end verification: create an invite as the bootstrap admin,
  register with it in a second browser session, confirm the resulting role
  matches what was invited, confirm the token can't be reused, confirm an
  expired or revoked token is rejected with the generic error.
