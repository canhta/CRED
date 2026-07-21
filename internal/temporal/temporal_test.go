package temporal_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/temporal"
)

// These tests take no database and no connection. If one ever needs either,
// the boundary this package exists to hold has already been violated.
//
// Timestamps are drawn from a small pool so boundary collisions are common —
// the generator technique testing-strategy.md rates above any single
// invariant.
var (
	t0 = time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	t1 = t0.Add(1 * time.Hour)
	t2 = t0.Add(2 * time.Hour)
	t3 = t0.Add(3 * time.Hour)
)

func iv(from, until time.Time) claim.Interval {
	return claim.Interval{From: from, Until: until}
}

func TestContainsIsHalfOpen(t *testing.T) {
	tests := []struct {
		name string
		i    claim.Interval
		at   time.Time
		want bool
	}{
		{"lower bound is included", iv(t1, t2), t1, true},
		{"upper bound is excluded", iv(t1, t2), t2, false},
		{"interior", iv(t1, t2), t1.Add(time.Minute), true},
		{"before", iv(t1, t2), t0, false},
		{"open interval has no upper bound", temporal.Forever(t1), t3, true},
		{"open interval still excludes the past", temporal.Forever(t1), t0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, temporal.Contains(tc.i, tc.at))
		})
	}
}

func TestAdjacentIntervalsDoNotOverlap(t *testing.T) {
	tests := []struct {
		name string
		a, b claim.Interval
		want bool
	}{
		{"adjacent", iv(t0, t1), iv(t1, t2), false},
		{"adjacent, reversed", iv(t1, t2), iv(t0, t1), false},
		{"overlapping", iv(t0, t2), iv(t1, t3), true},
		{"identical", iv(t0, t2), iv(t0, t2), true},
		{"contained", iv(t0, t3), iv(t1, t2), true},
		{"disjoint with a gap", iv(t0, t1), iv(t2, t3), false},
		{"open overlaps everything after its start", temporal.Forever(t0), iv(t2, t3), true},
		{"open does not overlap what ends before it", temporal.Forever(t2), iv(t0, t1), false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, temporal.Overlaps(tc.a, tc.b))
			require.Equal(t, tc.want, temporal.Overlaps(tc.b, tc.a),
				"overlap must be symmetric")
		})
	}
}

// TestMeetsIsGaplessAndNonOverlapping is the property half-open intervals buy:
// "no gaps and no overlaps" collapses to one equality.
func TestMeetsIsGaplessAndNonOverlapping(t *testing.T) {
	a, b := iv(t0, t1), iv(t1, t2)
	require.True(t, temporal.Meets(a, b))
	require.False(t, temporal.Overlaps(a, b))
	require.True(t, temporal.Contains(a, t0))
	require.True(t, temporal.Contains(b, t1))
	require.False(t, temporal.Contains(a, t1), "t1 belongs to exactly one of them")
}

func TestWellFormed(t *testing.T) {
	require.True(t, temporal.WellFormed(iv(t0, t1)))
	require.True(t, temporal.WellFormed(temporal.Forever(t0)))
	require.False(t, temporal.WellFormed(iv(t1, t0)), "end before start")
	require.False(t, temporal.WellFormed(iv(t1, t1)), "zero-width is not well-formed")
}

func TestCloseRefusesToClampSilently(t *testing.T) {
	got, err := temporal.Close(temporal.Forever(t0), t1)
	require.NoError(t, err)
	require.Equal(t, iv(t0, t1), got)

	_, err = temporal.Close(iv(t0, t1), t2)
	require.ErrorIs(t, err, temporal.ErrNotWellFormed, "already closed")

	_, err = temporal.Close(temporal.Forever(t1), t0)
	require.ErrorIs(t, err, temporal.ErrNotWellFormed, "closing before the start")

	_, err = temporal.Close(temporal.Forever(t1), t1)
	require.ErrorIs(t, err, temporal.ErrNotWellFormed, "zero-width close")
}

// TestCurrentAtRequiresBothDimensions is the error that reads as correct:
// filtering on valid time alone returns a plausible, wrong answer.
func TestCurrentAtRequiresBothDimensions(t *testing.T) {
	knownThenRetracted := claim.Claim{
		ID:       "retracted",
		Valid:    temporal.Forever(t0),
		Recorded: iv(t0, t1),
	}
	current := claim.Claim{
		ID:       "current",
		Valid:    temporal.Forever(t0),
		Recorded: temporal.Forever(t1),
	}
	all := []claim.Claim{knownThenRetracted, current}

	got := temporal.CurrentAt(all, t2, t2)
	require.Len(t, got, 1)
	require.Equal(t, "current", got[0].ID)

	// Rewinding transaction time brings the retracted claim back and hides
	// the one the system had not yet learned.
	got = temporal.CurrentAt(all, t2, t0)
	require.Len(t, got, 1)
	require.Equal(t, "retracted", got[0].ID)
}

func TestNoValidTimeOverlap(t *testing.T) {
	require.True(t, temporal.NoValidTimeOverlap([]claim.Interval{
		iv(t0, t1), iv(t1, t2), iv(t2, t3),
	}))
	require.False(t, temporal.NoValidTimeOverlap([]claim.Interval{
		iv(t0, t2), iv(t1, t3),
	}))
	require.True(t, temporal.NoValidTimeOverlap(nil))
}

func TestSupersedeClosesBothDimensions(t *testing.T) {
	incumbent := claim.Claim{
		ID:       "old",
		Valid:    temporal.Forever(t0),
		Recorded: temporal.Forever(t0),
	}
	got, err := temporal.Supersede(incumbent, "new", t1)
	require.NoError(t, err)
	require.Equal(t, "new", got.SupersededBy)
	require.Equal(t, iv(t0, t1), got.Recorded)
	require.Equal(t, iv(t0, t1), got.Valid)
	require.False(t, temporal.Contains(got.Recorded, t1),
		"the successor owns t1, and exactly one of them does")
}

func TestSupersedeRejectsSelfEdge(t *testing.T) {
	incumbent := claim.Claim{ID: "same", Recorded: temporal.Forever(t0)}
	_, err := temporal.Supersede(incumbent, "same", t1)
	require.ErrorIs(t, err, temporal.ErrNotWellFormed)

	_, err = temporal.Supersede(incumbent, "", t1)
	require.ErrorIs(t, err, temporal.ErrNotWellFormed)
}

// TestSupersedeIsMonotoneInTransactionTime — invariant 5. Superseding twice at
// the same instant, or backwards, must fail rather than produce a chain whose
// transaction times go the wrong way.
func TestSupersedeIsMonotoneInTransactionTime(t *testing.T) {
	incumbent := claim.Claim{ID: "old", Recorded: temporal.Forever(t1)}
	_, err := temporal.Supersede(incumbent, "new", t0)
	require.ErrorIs(t, err, temporal.ErrNotWellFormed)

	first, err := temporal.Supersede(incumbent, "new", t2)
	require.NoError(t, err)
	_, err = temporal.Supersede(first, "newer", t3)
	require.ErrorIs(t, err, temporal.ErrNotWellFormed,
		"an already-superseded claim cannot be superseded again")
}
