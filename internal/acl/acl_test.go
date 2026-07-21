package acl_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/acl"
	"github.com/canhta/cred/internal/claim"
)

// These tests take no database and no connection. If one ever needs either,
// the boundary this package exists to hold has already been violated.

var (
	now   = time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	past  = now.Add(-time.Hour)
	later = now.Add(time.Hour)

	alice = claim.PrincipalID("alice")
	bob   = claim.PrincipalID("bob")
	carol = claim.PrincipalID("carol")
)

func grants(ids ...claim.PrincipalID) claim.ACL {
	out := make(claim.ACL, 0, len(ids))
	for _, id := range ids {
		out = append(out, claim.Grant{Principal: id})
	}
	return out
}

func withEvidence(c claim.Claim, acls ...claim.ACL) claim.Claim {
	for i, a := range acls {
		c.Evidence = append(c.Evidence, claim.Evidence{
			ID:  string(rune('a' + i)),
			ACL: a,
		})
	}
	return c
}

// TestPermittedIsIntersectionNeverUnion guards the whole point of this
// package: union leaks private content to readers of the public source.
func TestPermittedIsIntersectionNeverUnion(t *testing.T) {
	c := withEvidence(
		claim.Claim{ACL: grants(alice, bob, carol)},
		grants(alice, bob),
		grants(bob, carol),
	)
	got := acl.Permitted(c, now)
	require.Len(t, got, 1)
	require.Contains(t, got, bob)
	require.NotContains(t, got, alice, "union would have admitted alice")
	require.NotContains(t, got, carol, "union would have admitted carol")
}

// TestClaimACLCannotWidenBeyondEvidence — the claim's own ACL is an upper
// bound that intersection may only narrow, never widen.
func TestClaimACLCannotWidenBeyondEvidence(t *testing.T) {
	c := withEvidence(
		claim.Claim{ACL: grants(alice, bob, carol)},
		grants(alice),
	)
	got := acl.Permitted(c, now)
	require.Len(t, got, 1)
	require.Contains(t, got, alice)
}

// TestEmptyACLIsReachableByNobody — treating an empty ACL as public is the
// fail-open bug.
func TestEmptyACLIsReachableByNobody(t *testing.T) {
	tests := []struct {
		name string
		c    claim.Claim
	}{
		{"empty claim ACL", withEvidence(claim.Claim{ACL: nil}, grants(alice))},
		{"empty evidence ACL", withEvidence(claim.Claim{ACL: grants(alice)}, nil)},
		{"both empty", withEvidence(claim.Claim{}, nil)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Empty(t, acl.Permitted(tc.c, now))
			require.False(t, acl.CanRead(tc.c, alice, now))
		})
	}
}

// TestClaimWithoutEvidenceIsUnreachable — an orphan claim is not merely
// invalid, it is unreadable, enforced through the permission channel.
func TestClaimWithoutEvidenceIsUnreachable(t *testing.T) {
	c := claim.Claim{ACL: grants(alice, bob)}
	require.Empty(t, c.Evidence)
	require.Empty(t, acl.Permitted(c, now))
	require.False(t, acl.CanRead(c, alice, now))
}

// TestExpiredGrantDeniesRatherThanErrors — an error would be an existence
// oracle.
func TestExpiredGrantDeniesRatherThanErrors(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		want      bool
	}{
		{"no expiry", time.Time{}, true},
		{"expires later", later, true},
		{"expired", past, false},
		{"expires exactly now", now, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := claim.ACL{{Principal: alice, ExpiresAt: tc.expiresAt}}
			c := withEvidence(claim.Claim{ACL: g}, g)
			require.Equal(t, tc.want, acl.CanRead(c, alice, now))
		})
	}
}

// TestStalePermissionDataDenies — the TTL is evaluated against the clock the
// caller supplies, so a grant that was valid at seed time is not valid
// forever.
func TestStalePermissionDataDenies(t *testing.T) {
	g := claim.ACL{{Principal: alice, ExpiresAt: now}}
	c := withEvidence(claim.Claim{ACL: g}, g)
	require.True(t, acl.CanRead(c, alice, past))
	require.False(t, acl.CanRead(c, alice, later))
}

func TestIntersectOfNoSetsIsEmptyNotUniversal(t *testing.T) {
	require.Empty(t, acl.Intersect(nil, now))
	require.Empty(t, acl.Intersect([]claim.ACL{}, now))
}

func TestIntersectIsOrderIndependent(t *testing.T) {
	a, b, c := grants(alice, bob), grants(bob, carol), grants(alice, bob, carol)
	forward := acl.Intersect([]claim.ACL{a, b, c}, now)
	reverse := acl.Intersect([]claim.ACL{c, b, a}, now)
	require.Equal(t, forward, reverse)
}

// TestUnauthorizedIsIndistinguishableFromNonexistent, in the form this package
// can assert: Filter drops rather than marks, so a denied claim leaves no trace
// in the output for a caller to count.
func TestUnauthorizedIsIndistinguishableFromNonexistent(t *testing.T) {
	visible := withEvidence(claim.Claim{ID: "v", ACL: grants(alice)}, grants(alice))
	hidden := withEvidence(claim.Claim{ID: "h", ACL: grants(bob)}, grants(bob))

	withHidden := acl.Filter([]claim.Claim{visible, hidden}, alice, now)
	withoutHidden := acl.Filter([]claim.Claim{visible}, alice, now)
	require.Equal(t, withoutHidden, withHidden)
}

func TestFilterPreservesOrder(t *testing.T) {
	mk := func(id string, p claim.PrincipalID) claim.Claim {
		return withEvidence(claim.Claim{ID: id, ACL: grants(p)}, grants(p))
	}
	in := []claim.Claim{mk("1", alice), mk("2", bob), mk("3", alice)}
	got := acl.Filter(in, alice, now)
	require.Len(t, got, 2)
	require.Equal(t, "1", got[0].ID)
	require.Equal(t, "3", got[1].ID)
}
