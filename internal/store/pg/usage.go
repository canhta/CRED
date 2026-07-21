package pg

import (
	"context"
	"time"

	"github.com/canhta/cred/internal/claim"
)

// This file is the storage half of PRD section 8. It stores and counts; it does
// not decide. Every "is this over the limit" and "how aggressively to prune"
// question is answered by internal/limit (pure); the methods here return the
// counters that package decides over, exactly as the store returns rows for
// internal/acl to intersect. No method here compares a count to a ceiling.

// SupersedeReasonPruned is written when the scope-growth bound expires a claim:
// the scope exceeded its ceiling and this was among its lowest-value live
// claims. It is expiry with no successor, like a stale-anchor expiry.
const SupersedeReasonPruned = "pruned"

// UsageEvent is one row of the cost-attribution ledger. The store writes it; the
// caller decides what it means. Kind is 'inference', 'recall', or 'denied'.
type UsageEvent struct {
	Principal     claim.PrincipalID
	Scope         claim.Scope
	Kind          string
	InferenceCall bool
	InputTokens   int
	OutputTokens  int
	WallMS        int64
	PackageClaims int
	DeniedReason  string // set only when Kind == "denied"
	Now           time.Time
}

// RecordUsage appends one event to the ledger. It is deliberately a plain
// INSERT with no read-modify-write: the ledger is append-only, so concurrent
// worker and recall paths never contend on a row.
func (s *Store) RecordUsage(ctx context.Context, e UsageEvent) error {
	calls := 0
	if e.InferenceCall {
		calls = 1
	}
	var reason any
	if e.DeniedReason != "" {
		reason = e.DeniedReason
	}
	now := e.Now
	if now.IsZero() {
		now = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO usage_events (principal_id, scope_kind, scope_value, kind,
		                          inference_calls, input_tokens, output_tokens,
		                          wall_ms, package_claims, denied_reason, recorded_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		string(e.Principal), string(e.Scope.Kind), e.Scope.Value, e.Kind,
		calls, e.InputTokens, e.OutputTokens, e.WallMS, e.PackageClaims, reason, now)
	return translate(err)
}

// ContributionsInWindow counts the claims a principal has had accepted since
// the cutoff — the number the contribution quota decides over. It counts
// superseded rows too: a claim that was accepted and then deduplicated was still
// accepted, and the quota is the flood backstop dedup is not, so it must not be
// discounted by the very dedup it backs up.
func (s *Store) ContributionsInWindow(ctx context.Context, principal claim.PrincipalID, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM claims
		 WHERE contributed_by = $1 AND recorded_at > $2`,
		string(principal), since).Scan(&n)
	return n, translate(err)
}

// InferenceCostInWindow sums a principal's inference calls and input tokens
// since the cutoff — what the hard cost ceiling decides over.
func (s *Store) InferenceCostInWindow(ctx context.Context, principal claim.PrincipalID, since time.Time) (calls, inputTokens int, err error) {
	err = s.pool.QueryRow(ctx, `
		SELECT coalesce(sum(inference_calls), 0), coalesce(sum(input_tokens), 0)
		  FROM usage_events
		 WHERE principal_id = $1 AND kind = 'inference' AND recorded_at > $2`,
		string(principal), since).Scan(&calls, &inputTokens)
	return calls, inputTokens, translate(err)
}

// RecallsInWindow counts a principal's recalls since the cutoff — what the
// recall-rate budget decides over.
func (s *Store) RecallsInWindow(ctx context.Context, principal claim.PrincipalID, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM usage_events
		 WHERE principal_id = $1 AND kind = 'recall' AND recorded_at > $2`,
		string(principal), since).Scan(&n)
	return n, translate(err)
}

// RecordRecall is the recall-path convenience over RecordUsage: it records a
// completed recall's wall-clock and package size, attributed to the principal.
// Recall spans scopes rather than belonging to one, so the scope is left empty —
// the per-scope attribution the PRD asks for is on the write (inference) side.
func (s *Store) RecordRecall(ctx context.Context, principal claim.PrincipalID, wallMS int64, packageClaims int, now time.Time) error {
	return s.RecordUsage(ctx, UsageEvent{
		Principal: principal, Kind: "recall",
		WallMS: wallMS, PackageClaims: packageClaims, Now: now,
	})
}

// DeniedInWindow counts a principal's denied contributions since the cutoff. A
// non-zero count is the queryable, on-the-record half of "exhaustion is loud,
// never a silent drop": even when the denial happened off the turn in a
// background worker (D-017), it left a row here.
func (s *Store) DeniedInWindow(ctx context.Context, principal claim.PrincipalID, since time.Time) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM usage_events
		 WHERE principal_id = $1 AND kind = 'denied' AND recorded_at > $2`,
		string(principal), since).Scan(&n)
	return n, translate(err)
}

// ScopeClaimCount returns how many live claims a scope holds — what the
// scope-growth bound decides over.
func (s *Store) ScopeClaimCount(ctx context.Context, scope claim.Scope) (int, error) {
	var n int
	err := s.pool.QueryRow(ctx, `
		SELECT count(*) FROM claims
		 WHERE scope_kind = $1 AND scope_value = $2 AND superseded_at IS NULL`,
		string(scope.Kind), scope.Value).Scan(&n)
	return n, translate(err)
}

// PruneScope expires the n lowest-value live claims in a scope, returning how
// many it closed. "Lowest value" is the lowest confidence, then the oldest, then
// by id for determinism — the v1 proxy for "claims that never surface", since
// per-claim surface counts (rescore) are not built yet. It is expiry, not
// supersession: nothing replaces a pruned claim, so superseded_by stays NULL,
// and nothing is deleted — the rows remain, closed in transaction time, so the
// record that the scope once held them survives.
//
// The decision of how many to prune is internal/limit's (PruneTarget); this
// method is handed the number and only closes that many. The ORDER BY is the
// tie-break policy, kept in one place and deterministic so a prune pass is
// reproducible.
func (s *Store) PruneScope(ctx context.Context, scope claim.Scope, n int, now time.Time) (int, error) {
	if n <= 0 {
		return 0, nil
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE claims
		   SET superseded_at = $3, valid_until = $3, supersede_reason = $4
		 WHERE id IN (
		     SELECT id FROM claims
		      WHERE scope_kind = $1 AND scope_value = $2 AND superseded_at IS NULL
		      ORDER BY confidence ASC, recorded_at ASC, id ASC
		      LIMIT $5)`,
		string(scope.Kind), scope.Value, now, SupersedeReasonPruned, n)
	if err != nil {
		return 0, translate(err)
	}
	return int(tag.RowsAffected()), nil
}

// ContributionState is one principal's window counts, for `cred usage`. It is
// the counters, not the decision — the CLI passes them to internal/limit to
// render remaining headroom, so the number a user sees is the number the
// enforcement path computes.
type ContributionState struct {
	Contributions int
	InferenceCall int
	InputTokens   int
	Recalls       int
}

// ScopeCost is per-scope inference cost, for the "which teams actually use
// this" report — the founder-facing question no competitor exposes.
type ScopeCost struct {
	Scope        claim.Scope
	Calls        int
	InputTokens  int
	OutputTokens int
}

// ScopeSize is a scope's live-claim count, for the scope-growth report.
type ScopeSize struct {
	Scope claim.Scope
	Live  int
}

// UsageByScope aggregates inference cost per scope since the cutoff, most
// expensive first.
func (s *Store) UsageByScope(ctx context.Context, since time.Time) ([]ScopeCost, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT scope_kind, scope_value,
		       coalesce(sum(inference_calls), 0),
		       coalesce(sum(input_tokens), 0),
		       coalesce(sum(output_tokens), 0)
		  FROM usage_events
		 WHERE kind = 'inference' AND recorded_at > $1
		 GROUP BY scope_kind, scope_value
		 ORDER BY sum(input_tokens) DESC, scope_kind, scope_value`, since)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	var out []ScopeCost
	for rows.Next() {
		var c ScopeCost
		if err := rows.Scan(&c.Scope.Kind, &c.Scope.Value, &c.Calls, &c.InputTokens, &c.OutputTokens); err != nil {
			return nil, translate(err)
		}
		out = append(out, c)
	}
	return out, translate(rows.Err())
}

// ScopeSizes returns the live-claim count per scope, largest first, so `cred
// usage` can show which scopes are near their growth ceiling.
func (s *Store) ScopeSizes(ctx context.Context, limit int) ([]ScopeSize, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT scope_kind, scope_value, count(*)
		  FROM claims
		 WHERE superseded_at IS NULL
		 GROUP BY scope_kind, scope_value
		 ORDER BY count(*) DESC, scope_kind, scope_value
		 LIMIT $1`, limit)
	if err != nil {
		return nil, translate(err)
	}
	defer rows.Close()

	var out []ScopeSize
	for rows.Next() {
		var sz ScopeSize
		if err := rows.Scan(&sz.Scope.Kind, &sz.Scope.Value, &sz.Live); err != nil {
			return nil, translate(err)
		}
		out = append(out, sz)
	}
	return out, translate(rows.Err())
}

// PrincipalWindowState gathers the three windowed per-principal counts in one
// call, each over its own window cutoff. The caller (internal/limit) turns them
// into remaining headroom.
func (s *Store) PrincipalWindowState(ctx context.Context, principal claim.PrincipalID,
	contributionSince, costSince, recallSince time.Time,
) (ContributionState, error) {
	var st ContributionState
	var err error
	if st.Contributions, err = s.ContributionsInWindow(ctx, principal, contributionSince); err != nil {
		return st, err
	}
	if st.InferenceCall, st.InputTokens, err = s.InferenceCostInWindow(ctx, principal, costSince); err != nil {
		return st, err
	}
	if st.Recalls, err = s.RecallsInWindow(ctx, principal, recallSince); err != nil {
		return st, err
	}
	return st, nil
}
