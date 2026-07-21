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
