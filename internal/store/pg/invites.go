package pg

import (
	"context"
	"time"

	"github.com/canhta/cred/internal/claim"
)

// This file backs the invite flow: an admin creates an invite naming an
// email and a role, and whoever redeems it with a matching email gets the
// account CreateInvitedUser makes, with invited_by set to the admin who
// created it.

// Invite is one row from PendingInvites -- what the Team page's pending-
// invites table renders.
type Invite struct {
	ID        string
	Email     string
	Role      string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// CreateInvite inserts a new invite row and returns its id.
func (s *Store) CreateInvite(ctx context.Context, email, role, tokenHash string,
	invitedBy claim.PrincipalID, expiresAt time.Time,
) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
		INSERT INTO invites (email, role, token_hash, invited_by, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`, email, role, tokenHash, string(invitedBy), expiresAt).Scan(&id)
	return id, translate(err)
}

// ClaimInvite atomically marks an invite used and returns what it named --
// or ErrNotFound (translated) if it was already used, revoked, expired,
// never existed, or the email doesn't match. A single UPDATE ... RETURNING,
// not a separate lookup-then-mark, closes the same kind of race the
// bootstrap path has (two concurrent redemptions of one single-use token
// must not both succeed) and never burns the invite on a mismatched-email
// attempt: a WHERE clause that doesn't match affects zero rows and changes
// nothing.
func (s *Store) ClaimInvite(ctx context.Context, tokenHash, email string, now time.Time) (
	role string, invitedBy claim.PrincipalID, err error,
) {
	var invitedByStr string
	err = s.pool.QueryRow(ctx, `
		UPDATE invites SET used_at = $3
		 WHERE token_hash = $1 AND email = $2
		   AND used_at IS NULL AND revoked_at IS NULL AND expires_at > $3
		RETURNING role, invited_by`,
		tokenHash, email, now).Scan(&role, &invitedByStr)
	if err != nil {
		return "", "", translate(err)
	}
	return role, claim.PrincipalID(invitedByStr), nil
}

// RevokeInvite marks a pending invite as revoked. Revoking an already-used,
// already-expired, or already-revoked invite is a silent no-op -- there is
// nothing left to protect against for a token that can no longer be
// redeemed.
func (s *Store) RevokeInvite(ctx context.Context, id string, now time.Time) error {
	_, err := s.pool.Exec(ctx, `UPDATE invites SET revoked_at = $2 WHERE id = $1`, id, now)
	return translate(err)
}

// PendingInvites lists invites that are still redeemable, oldest first.
func (s *Store) PendingInvites(ctx context.Context, now time.Time) ([]Invite, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, email, role, created_at, expires_at FROM invites
		 WHERE used_at IS NULL AND revoked_at IS NULL AND expires_at > $1
		 ORDER BY created_at`, now)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	var out []Invite
	for rows.Next() {
		var inv Invite
		if err := rows.Scan(&inv.ID, &inv.Email, &inv.Role, &inv.CreatedAt, &inv.ExpiresAt); err != nil {
			return nil, translate(err)
		}
		out = append(out, inv)
	}
	return out, translate(rows.Err())
}

// CreateInvitedUser inserts a new principal and its credentials with
// invited_by set -- the invite-redemption twin of CreateUser's open
// bootstrap path. A separate method rather than a shared invitedBy
// parameter on CreateUser: the two paths have different callers in
// different packages, and this keeps CreateUser's existing signature (and
// every existing call site) untouched.
func (s *Store) CreateInvitedUser(ctx context.Context, email, passwordHash, role string,
	invitedBy claim.PrincipalID,
) (claim.PrincipalID, error) {
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
		INSERT INTO user_credentials (principal_id, email, password_hash, role, invited_by)
		VALUES ($1, $2, $3, $4, $5)`, principalID, email, passwordHash, role, string(invitedBy))
	if err != nil {
		return "", translate(err)
	}

	return claim.PrincipalID(principalID), translate(tx.Commit(ctx))
}

// TeamMember is one row from TeamMembers -- every account, regardless of
// how it was created.
type TeamMember struct {
	PrincipalID string
	Email       string
	Role        string
	CreatedAt   time.Time
}

// TeamMembers lists every account, oldest first.
func (s *Store) TeamMembers(ctx context.Context) ([]TeamMember, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT principal_id, email, role, created_at FROM user_credentials
		 ORDER BY created_at`)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	var out []TeamMember
	for rows.Next() {
		var m TeamMember
		if err := rows.Scan(&m.PrincipalID, &m.Email, &m.Role, &m.CreatedAt); err != nil {
			return nil, translate(err)
		}
		out = append(out, m)
	}
	return out, translate(rows.Err())
}
