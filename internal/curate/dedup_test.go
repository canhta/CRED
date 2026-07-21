package curate

import (
	"testing"
	"time"

	"github.com/canhta/cred/internal/store/pg"
)

func member(id string, sec int) pg.DupMember {
	return pg.DupMember{ID: id, RecordedAt: time.Date(2026, 7, 21, 0, 0, sec, 0, time.UTC)}
}

func TestCanonicalKeepsEarliest(t *testing.T) {
	// The earliest-recorded claim survives; the later duplicates are the churn
	// automatic writes produce, and they fold into it.
	group := []pg.DupMember{member("c", 30), member("a", 10), member("b", 20)}
	survivor, dups := Canonical(group)
	if survivor != "a" {
		t.Fatalf("survivor = %q, want the earliest 'a'", survivor)
	}
	if len(dups) != 2 {
		t.Fatalf("duplicates = %v, want two", dups)
	}
}

func TestCanonicalBreaksTiesByID(t *testing.T) {
	// Equal timestamps must resolve to a deterministic total order, or the
	// reconciler is not byte-identical across runs (a v1 acceptance criterion).
	group := []pg.DupMember{member("z", 10), member("m", 10), member("a", 10)}
	survivor, dups := Canonical(group)
	if survivor != "a" {
		t.Fatalf("survivor = %q, want 'a' by id tie-break", survivor)
	}
	if len(dups) != 2 {
		t.Fatalf("duplicates = %v, want two", dups)
	}
}

func TestCanonicalSingletonHasNoDuplicates(t *testing.T) {
	survivor, dups := Canonical([]pg.DupMember{member("only", 1)})
	if survivor != "only" || len(dups) != 0 {
		t.Fatalf("survivor=%q dups=%v, want 'only' with no duplicates", survivor, dups)
	}
}

func TestCanonicalEmptyGroup(t *testing.T) {
	survivor, dups := Canonical(nil)
	if survivor != "" || dups != nil {
		t.Fatalf("empty group should yield no survivor")
	}
}
