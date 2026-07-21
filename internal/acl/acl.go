// Package acl implements L5 — access control evaluated at recall, failing
// closed, as the intersection — as pure functions over domain types.
//
// This package must not import a database driver, database/sql, or pgx, and no
// function here may take a connection. depguard fails the build if that
// changes.
//
// L5 is never a SQL predicate. That costs performance: rows cross the wire
// that Postgres could have discarded. It is affordable at one instance per
// organization, and the alternative is the known silent-failure path —
// pgvector filtering under ACL selectivity returns 4 results where 40 were
// expected, with no error. Deciding in SQL is not merely harder to test.
package acl

import (
	"time"

	"github.com/canhta/cred/internal/claim"
)

// Active returns the principals a grant set admits at now.
//
// Expiry denies. It does not error: an error is an existence oracle, and
// unauthorized must be indistinguishable from nonexistent
// (testing-strategy adversarial cases 5 and 8).
func Active(a claim.ACL, now time.Time) map[claim.PrincipalID]struct{} {
	out := make(map[claim.PrincipalID]struct{}, len(a))
	for _, g := range a {
		if !g.ExpiresAt.IsZero() && !now.Before(g.ExpiresAt) {
			continue // TTL elapsed: deny, silently
		}
		out[g.Principal] = struct{}{}
	}
	return out
}

// Intersect returns the principals admitted by every one of the given sets.
//
// Intersection, never union. Union leaks private content to readers of the
// public source, and it is the natural implementation — merging two claims by
// keeping the survivor's ACL is the same bug wearing different clothes.
//
// Intersect of no sets is empty, not universal. A claim with no evidence is
// unreachable rather than public, which is L1 enforced through the permission
// channel.
func Intersect(sets []claim.ACL, now time.Time) map[claim.PrincipalID]struct{} {
	if len(sets) == 0 {
		return map[claim.PrincipalID]struct{}{}
	}
	out := Active(sets[0], now)
	for _, s := range sets[1:] {
		next := Active(s, now)
		for p := range out {
			if _, ok := next[p]; !ok {
				delete(out, p)
			}
		}
	}
	return out
}

// Permitted reports the principals who may read a claim: the claim's own ACL
// intersected with the intersection of its evidence ACLs.
//
//	claim.acl ⊆ ⋂(evidence_i.acl)
//
// A claim carrying no evidence is readable by nobody. That is L1 and it fails
// closed by construction rather than by a check that could be forgotten.
func Permitted(c claim.Claim, now time.Time) map[claim.PrincipalID]struct{} {
	if len(c.Evidence) == 0 {
		return map[claim.PrincipalID]struct{}{}
	}
	sets := make([]claim.ACL, 0, len(c.Evidence)+1)
	sets = append(sets, c.ACL)
	for _, e := range c.Evidence {
		sets = append(sets, e.ACL)
	}
	return Intersect(sets, now)
}

// CanRead reports whether p may read c at now.
func CanRead(c claim.Claim, p claim.PrincipalID, now time.Time) bool {
	_, ok := Permitted(c, now)[p]
	return ok
}

// Filter returns the claims p may read at now, preserving order.
//
// Recall calls this before budgeting, never after. Access control applied
// after a token budget leaks the existence of what was dropped.
func Filter(claims []claim.Claim, p claim.PrincipalID, now time.Time) []claim.Claim {
	out := make([]claim.Claim, 0, len(claims))
	for _, c := range claims {
		if CanRead(c, p, now) {
			out = append(out, c)
		}
	}
	return out
}
