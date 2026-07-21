package curate

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/store/pg"
	"github.com/canhta/cred/internal/temporal"
)

// DedupStore is the subset of the row store the reconciler needs. It hands back
// duplicate groups and applies a supersession that temporal already computed; it
// makes no decision itself.
type DedupStore interface {
	LiveDuplicateGroups(ctx context.Context) ([][]pg.DupMember, error)
	LoadClaimForSupersession(ctx context.Context, id string) (claim.Claim, error)
	ApplySupersession(ctx context.Context, c claim.Claim, reason string) error
}

// SupersedeReasonDuplicate is written when a claim is superseded because an
// identical statement already exists. It is a link, not a delete: both claims
// persist, one is marked superseded by the survivor, and an unmerge is a later
// update rather than an impossibility.
const SupersedeReasonDuplicate = "duplicate"

// Canonical picks the survivor of a duplicate group deterministically: the
// earliest-recorded claim, ties broken by id. It is pure so the reconciler is
// byte-identical across runs and unit-testable without a database.
//
// The earliest claim survives because it is the one whose evidence chain is
// oldest and most likely already referenced; the later duplicates are the churn
// automatic writes produce at every third turn, and they are the ones to fold
// away.
func Canonical(group []pg.DupMember) (survivor string, duplicates []string) {
	if len(group) == 0 {
		return "", nil
	}
	best := group[0]
	for _, m := range group[1:] {
		if earlier(m, best) {
			best = m
		}
	}
	for _, m := range group {
		if m.ID != best.ID {
			duplicates = append(duplicates, m.ID)
		}
	}
	return best.ID, duplicates
}

func earlier(a, b pg.DupMember) bool {
	if !a.RecordedAt.Equal(b.RecordedAt) {
		return a.RecordedAt.Before(b.RecordedAt)
	}
	return a.ID < b.ID
}

// Reconciler runs the exact-hash deduplication pass. It is deterministic and
// takes no model — dedup is a code decision, not a nomination. Contradiction
// reconciliation, which does need a nomination, is not built yet; see the README.
type Reconciler struct {
	store DedupStore
	log   *slog.Logger
}

// NewReconciler builds a Reconciler.
func NewReconciler(store DedupStore, log *slog.Logger) *Reconciler {
	return &Reconciler{store: store, log: log}
}

// Dedup folds exact-hash duplicate groups into their canonical survivor. It
// returns the number of claims superseded. Every supersession reuses
// internal/temporal.Supersede to close the incumbent's intervals — the
// bi-temporal machinery is not reimplemented here.
func (r *Reconciler) Dedup(ctx context.Context) (int, error) {
	groups, err := r.store.LiveDuplicateGroups(ctx)
	if err != nil {
		return 0, err
	}
	merged := 0
	now := time.Now().UTC()
	for _, group := range groups {
		survivor, duplicates := Canonical(group)
		for _, dupID := range duplicates {
			if err := r.supersede(ctx, dupID, survivor, now); err != nil {
				return merged, err
			}
			merged++
		}
	}
	if merged > 0 && r.log != nil {
		r.log.Info("dedup pass", slog.Int("superseded", merged))
	}
	return merged, nil
}

// supersede closes one duplicate in favour of the survivor, using the pure
// bi-temporal algebra to compute the closed intervals and the store only to
// persist them.
func (r *Reconciler) supersede(ctx context.Context, dupID, survivorID string, now time.Time) error {
	incumbent, err := r.store.LoadClaimForSupersession(ctx, dupID)
	if errors.Is(err, pg.ErrNotFound) {
		// Already superseded between the group scan and now: a concurrent pass
		// or a retried job got there first. Not an error — the outcome is what
		// we wanted.
		return nil
	}
	if err != nil {
		return err
	}
	superseded, err := temporal.Supersede(incumbent, survivorID, now)
	if err != nil {
		return err
	}
	return r.store.ApplySupersession(ctx, superseded, SupersedeReasonDuplicate)
}
