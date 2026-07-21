package limit

import (
	"testing"
	"time"
)

func TestWindowStartIsHalfOpen(t *testing.T) {
	now := time.Date(2026, 7, 21, 12, 0, 0, 0, time.UTC)
	start := WindowStart(now, time.Hour)
	if !start.Equal(now.Add(-time.Hour)) {
		t.Fatalf("WindowStart = %v, want %v", start, now.Add(-time.Hour))
	}
	// The convention the store relies on: `recorded_at > WindowStart`. An event
	// exactly at the boundary is out of window (aged out), an instant later is
	// in. This mirrors internal/temporal's half-open intervals.
	if start.After(now.Add(-time.Hour)) || start.Before(now.Add(-time.Hour)) {
		t.Fatal("boundary must be exactly one window before now")
	}
}

func TestContributionBoundary(t *testing.T) {
	cfg := Config{ContributionQuota: 3}
	tests := []struct {
		accepted      int
		wantAllowed   bool
		wantRemaining int
	}{
		{0, true, 3},
		{2, true, 1},   // one below the quota: allowed, one left
		{3, false, 0},  // exactly at the quota: the next is denied
		{4, false, 0},  // over (should not occur, but must still deny)
		{99, false, 0}, // far over
	}
	for _, tt := range tests {
		got := Contribution(tt.accepted, cfg)
		if got.Allowed != tt.wantAllowed {
			t.Errorf("Contribution(%d) allowed = %v, want %v", tt.accepted, got.Allowed, tt.wantAllowed)
		}
		if got.Remaining != tt.wantRemaining {
			t.Errorf("Contribution(%d) remaining = %d, want %d", tt.accepted, got.Remaining, tt.wantRemaining)
		}
		if !tt.wantAllowed && got.Reason != ReasonContributionQuota {
			t.Errorf("Contribution(%d) reason = %q, want %q", tt.accepted, got.Reason, ReasonContributionQuota)
		}
	}
}

func TestContributionDisabledIsUnlimited(t *testing.T) {
	for _, quota := range []int{0, -1} {
		got := Contribution(1_000_000, Config{ContributionQuota: quota})
		if !got.Allowed || got.Remaining != -1 {
			t.Fatalf("quota %d must disable the control: got %+v", quota, got)
		}
	}
}

func TestCostCeilingEitherDimensionBinds(t *testing.T) {
	cfg := Config{MaxInferenceCalls: 10, MaxInputTokens: 1000}

	if d := Cost(9, 999, cfg); !d.Allowed {
		t.Fatalf("just under both ceilings must be allowed: %+v", d)
	}
	// Calls exactly at the ceiling: denied on calls.
	if d := Cost(10, 0, cfg); d.Allowed || d.Reason != ReasonCostCeiling {
		t.Fatalf("calls at ceiling must deny: %+v", d)
	}
	// Calls fine, tokens exactly at the ceiling: denied on tokens.
	if d := Cost(0, 1000, cfg); d.Allowed || d.Reason != ReasonCostCeiling {
		t.Fatalf("tokens at ceiling must deny: %+v", d)
	}
}

func TestRecallRateBoundary(t *testing.T) {
	cfg := Config{RecallRate: 2}
	if d := RecallRate(1, cfg); !d.Allowed || d.Remaining != 1 {
		t.Fatalf("one below rate must be allowed with one remaining: %+v", d)
	}
	if d := RecallRate(2, cfg); d.Allowed || d.Reason != ReasonRecallRate {
		t.Fatalf("at the rate must deny: %+v", d)
	}
}

func TestPruneTargetGrowsWithOverage(t *testing.T) {
	cfg := Config{ScopeClaimCeiling: 100, PruneAggressiveness: 0.5}

	// Under and exactly at the ceiling: prune nothing.
	if n := PruneTarget(99, cfg); n != 0 {
		t.Fatalf("under ceiling must prune 0, got %d", n)
	}
	if n := PruneTarget(100, cfg); n != 0 {
		t.Fatalf("at ceiling must prune 0, got %d", n)
	}

	// Just over: prune the overage plus ceil(overage*aggressiveness).
	// over=1 -> headroom ceil(0.5)=1 -> target 2.
	if n := PruneTarget(101, cfg); n != 2 {
		t.Fatalf("one over must prune 2, got %d", n)
	}
	// over=10 -> headroom 5 -> target 15.
	if n := PruneTarget(110, cfg); n != 15 {
		t.Fatalf("ten over must prune 15, got %d", n)
	}

	// Aggressiveness is monotonic in overage: further over prunes strictly more.
	prev := 0
	for live := 101; live <= 200; live++ {
		n := PruneTarget(live, cfg)
		if n < prev {
			t.Fatalf("prune target must not decrease as scope grows: live=%d n=%d prev=%d", live, n, prev)
		}
		prev = n
	}
}

func TestPruneTargetBoundedByLiveCount(t *testing.T) {
	// A tiny ceiling with a huge aggressiveness must never ask to prune more than
	// exists.
	cfg := Config{ScopeClaimCeiling: 1, PruneAggressiveness: 10}
	if n := PruneTarget(5, cfg); n != 5 {
		t.Fatalf("prune target must be capped at the live count, got %d", n)
	}
}

func TestPruneTargetDisabled(t *testing.T) {
	if n := PruneTarget(10_000, Config{ScopeClaimCeiling: 0}); n != 0 {
		t.Fatal("a non-positive ceiling disables pruning")
	}
}

func TestDefaultsAreUsable(t *testing.T) {
	cfg := Defaults()
	if !Contribution(0, cfg).Allowed {
		t.Fatal("a fresh principal must be allowed to contribute under defaults")
	}
	if !Cost(0, 0, cfg).Allowed {
		t.Fatal("a fresh principal must be allowed to nominate under defaults")
	}
	if !RecallRate(0, cfg).Allowed {
		t.Fatal("a fresh principal must be allowed to recall under defaults")
	}
	if PruneTarget(cfg.ScopeClaimCeiling, cfg) != 0 {
		t.Fatal("a scope at the default ceiling prunes nothing")
	}
}
