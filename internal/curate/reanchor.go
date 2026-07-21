package curate

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/canhta/cred/internal/anchor"
	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/store/pg"
)

// ReanchorStore is the subset of the row store re-anchoring needs. It hands back
// live anchored evidence and applies an expiry the reanchorer already decided;
// it makes no invalidation decision itself, for the same reason the reconciler
// does not: the deterministic decision lives in internal/anchor (pure), and the
// store only persists the close.
type ReanchorStore interface {
	LiveAnchoredEvidence(ctx context.Context, repo string) ([]pg.AnchoredEvidence, error)
	ExpireClaim(ctx context.Context, id, reason string, now time.Time) error
}

// Reanchorer runs the anchor ladder's deterministic invalidation: for each live claim anchored
// to a file under a repository root, it re-resolves the stored anchor against
// the current file and expires the claim when the verdict is a semantic change
// or ambiguous. Formatting churn — tier 4 changed while tiers 1 and 2 hold —
// expires nothing, which is the property the ladder exists to provide.
//
// It reuses the existing expiry machinery (pg.ExpireClaim, the same close
// `cred forget` performs) rather than reinventing one: invalidation is
// deterministic, and the determinism lives in the pure anchorer, not here.
type Reanchorer struct {
	store ReanchorStore
	log   *slog.Logger
}

// NewReanchorer builds a Reanchorer.
func NewReanchorer(store ReanchorStore, log *slog.Logger) *Reanchorer {
	return &Reanchorer{store: store, log: log}
}

// ReanchorReport is what one re-anchoring pass did. Kept legible on purpose: the
// whole value of the anchor ladder is that the reason a claim survived or expired
// is inspectable.
type ReanchorReport struct {
	Repo        string
	Checked     int      // anchored evidence rows resolved
	Valid       int      // survived — tiers 1 and 2 agreed (formatting churn included)
	Expired     int      // expired — semantic change or ambiguous
	Unanchored  int      // tier-4-only rows the ladder does not apply to (should be 0 here)
	MissingFile int      // anchored files no longer on disk; left untouched, not expired
	Details     []string // one line per expiry, for `cred reanchor` output
	Duration    time.Duration
}

// Reanchor resolves every live anchored claim under root against the current
// files and expires the ones whose evidence no longer holds. root is the
// repository root the evidence was seeded or written against (its absolute path
// is the source_repo). Files are read fresh from disk, so this is the operation
// a CI step or a post-edit hook runs: a pure-formatting commit expires zero
// claims; a semantic change expires exactly the right ones.
func (r *Reanchorer) Reanchor(ctx context.Context, root string) (ReanchorReport, error) {
	started := time.Now()

	abs, err := filepath.Abs(root)
	if err != nil {
		return ReanchorReport{}, fmt.Errorf("resolve %s: %w", root, err)
	}
	repo := filepath.ToSlash(abs)

	rows, err := r.store.LiveAnchoredEvidence(ctx, repo)
	if err != nil {
		return ReanchorReport{}, fmt.Errorf("load anchored evidence: %w", err)
	}

	rep := ReanchorReport{Repo: repo}
	// One read per file, not per row: many claims anchor into the same file. A nil
	// entry records a known-missing file so it is not stat-ed twice.
	files := map[string]*string{}
	now := time.Now().UTC()

	for _, ev := range rows {
		src, ok, err := readFile(files, abs, ev.Path)
		if err != nil {
			return rep, err
		}
		if !ok {
			// The file is gone. Deleting a file is a source change, but it is also
			// how a repository is reorganized, and expiring on a missing file would
			// make `cred reanchor` on a partial checkout destroy live memory. Left
			// untouched deliberately; a re-seed is the deliberate act that reconciles
			// deletions.
			rep.MissingFile++
			continue
		}

		rep.Checked++
		a, dispatchOK := anchor.For(claim.SourceKind(ev.SourceKind))
		if !dispatchOK {
			rep.Unanchored++
			continue
		}
		stored := anchor.Anchor{
			SymbolPath: ev.SymbolPath,
			NodeHash:   ev.NodeHash,
			WindowHash: ev.WindowHash,
			ByteHash:   ev.ByteHash,
		}
		v := a.Resolve(stored, anchor.Source{Text: src, Kind: claim.SourceKind(ev.SourceKind), Path: ev.Path})

		switch {
		case v.Kind == anchor.Unanchored:
			rep.Unanchored++
		case v.Kind.Expires():
			if err := r.expire(ctx, ev.ClaimID, now); err != nil {
				return rep, err
			}
			rep.Expired++
			rep.Details = append(rep.Details,
				fmt.Sprintf("expired %s  %s:%d-%d  %s", ev.ClaimID, ev.Path, ev.LineStart, ev.LineEnd, v.Reason))
		default: // Valid
			rep.Valid++
		}
	}

	rep.Duration = time.Since(started)
	if r.log != nil {
		// Identifiers and counts only — never source text.
		r.log.Info("reanchor pass",
			slog.String("repo", repo),
			slog.Int("checked", rep.Checked),
			slog.Int("valid", rep.Valid),
			slog.Int("expired", rep.Expired),
			slog.Int("missing_file", rep.MissingFile))
	}
	return rep, nil
}

// expire closes the claim with the stale-anchor reason. A row already expired
// between the scan and now (a concurrent forget, or another reanchor) is not an
// error: the outcome is what we wanted.
func (r *Reanchorer) expire(ctx context.Context, claimID string, now time.Time) error {
	err := r.store.ExpireClaim(ctx, claimID, pg.SupersedeReasonStaleAnchor, now)
	if errors.Is(err, pg.ErrNotFound) {
		return nil
	}
	return err
}

// readFile reads path (relative to root) once and caches it. The bool is false
// when the file does not exist; a genuine read error propagates. A cached nil
// pointer records a known-missing file.
func readFile(cache map[string]*string, root, rel string) (string, bool, error) {
	full := filepath.Join(root, filepath.FromSlash(rel))
	if s, ok := cache[full]; ok {
		if s == nil {
			return "", false, nil
		}
		return *s, true, nil
	}
	b, err := os.ReadFile(full)
	if errors.Is(err, os.ErrNotExist) {
		cache[full] = nil
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("read %s: %w", rel, err)
	}
	s := string(b)
	cache[full] = &s
	return s, true, nil
}
