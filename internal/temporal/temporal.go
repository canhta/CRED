// Package temporal implements CRED's bi-temporal algebra as pure functions
// over domain types.
//
// This package must not import a database driver, database/sql, or pgx, and no
// function here may take a connection. depguard fails the build if that
// changes. The reason is not taste: a temporal predicate expressed in SQL is a
// predicate that cannot be unit-tested, and Postgres is here to store and
// filter, not to decide.
//
// Every interval is half-open, [From, Until). That single choice makes "no
// gaps and no overlaps" a single equality and removes the boundary off-by-one
// class outright.
package temporal

import (
	"errors"
	"time"

	"github.com/canhta/cred/internal/claim"
)

// ErrNotWellFormed reports an interval whose end does not follow its start.
var ErrNotWellFormed = errors.New("temporal: interval end must follow its start")

// Forever is the open-ended interval starting at t.
func Forever(t time.Time) claim.Interval { return claim.Interval{From: t} }

// IsOpen reports whether an interval has no end.
func IsOpen(i claim.Interval) bool { return i.Until.IsZero() }

// WellFormed reports whether From strictly precedes Until. An open interval is
// well-formed by construction, which is why intervals are generated as
// (start, duration >= 1) rather than as two independent instants — no shrinker
// can then break the invariant.
func WellFormed(i claim.Interval) bool {
	return IsOpen(i) || i.From.Before(i.Until)
}

// Contains reports whether t falls in [From, Until). The upper bound is
// excluded; an instant exactly at Until belongs to the next interval.
func Contains(i claim.Interval, t time.Time) bool {
	if t.Before(i.From) {
		return false
	}
	return IsOpen(i) || t.Before(i.Until)
}

// Overlaps reports whether two half-open intervals share any instant.
// Adjacent intervals — one ending exactly where the next begins — do not
// overlap.
func Overlaps(a, b claim.Interval) bool {
	if !IsOpen(a) && !a.Until.After(b.From) {
		return false
	}
	if !IsOpen(b) && !b.Until.After(a.From) {
		return false
	}
	return true
}

// Meets reports whether a ends exactly where b begins: no gap, no overlap.
func Meets(a, b claim.Interval) bool {
	return !IsOpen(a) && a.Until.Equal(b.From)
}

// Close ends an open interval at t. Closing an already-closed interval, or
// closing before the interval started, is an error rather than a silent
// clamp — a silently clamped interval is a claim that was never true.
func Close(i claim.Interval, t time.Time) (claim.Interval, error) {
	if !IsOpen(i) {
		return claim.Interval{}, ErrNotWellFormed
	}
	if !t.After(i.From) {
		return claim.Interval{}, ErrNotWellFormed
	}
	return claim.Interval{From: i.From, Until: t}, nil
}

// CurrentAt returns the claims that were true in the world at validAt and
// known to the system at txAt.
//
// Both dimensions are required. Filtering on only one is the most common
// bi-temporal error, and it reads as correct because the results are
// plausible.
func CurrentAt(claims []claim.Claim, validAt, txAt time.Time) []claim.Claim {
	out := make([]claim.Claim, 0, len(claims))
	for _, c := range claims {
		if Contains(c.Valid, validAt) && Contains(c.Recorded, txAt) {
			out = append(out, c)
		}
	}
	return out
}

// NoValidTimeOverlap reports whether a set of intervals is pairwise
// non-overlapping. Within one transaction slice, per kind, at most one claim
// may be valid at any instant.
func NoValidTimeOverlap(intervals []claim.Interval) bool {
	for i := range intervals {
		for j := i + 1; j < len(intervals); j++ {
			if Overlaps(intervals[i], intervals[j]) {
				return false
			}
		}
	}
	return true
}

// Supersede closes the incumbent's transaction-time interval at t and links it
// to its successor.
//
// It refuses to close an interval that is already closed, and refuses a
// self-edge. Cycle rejection over longer chains belongs with the reconciler,
// which this slice does not ship; the one-step case is enforced here so the
// gap is a missing feature rather than a silent hole.
func Supersede(incumbent claim.Claim, successorID string, t time.Time) (claim.Claim, error) {
	if successorID == "" || successorID == incumbent.ID {
		return claim.Claim{}, ErrNotWellFormed
	}
	recorded, err := Close(incumbent.Recorded, t)
	if err != nil {
		return claim.Claim{}, err
	}
	incumbent.Recorded = recorded
	incumbent.SupersededBy = successorID
	if IsOpen(incumbent.Valid) {
		valid, err := Close(incumbent.Valid, t)
		if err != nil {
			return claim.Claim{}, err
		}
		incumbent.Valid = valid
	}
	return incumbent, nil
}
