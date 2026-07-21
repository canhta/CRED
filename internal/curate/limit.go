package curate

import (
	"context"
	"log/slog"
	"time"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/limit"
	"github.com/canhta/cred/internal/nominate"
	"github.com/canhta/cred/internal/obs"
	"github.com/canhta/cred/internal/store/pg"
)

// This file is the store-backed enforcement of the usage limits on the write
// side: the cost-attribution recorder wired at the nominate boundary, the
// contribution-quota and cost-ceiling gate on the automatic write path, and the
// scope-growth pruner. The decision in each is internal/limit's (pure); the
// counters and the persistence are the store's. This package is the only side
// of the LLM boundary that may reach the store, which is why the glue lives
// here and not in internal/nominate.

// firstPrincipal is the contributing principal a job is attributed to. This
// slice ships one principal per contribution; the helper is the single place
// that assumption lives, so widening it later is one change.
func firstPrincipal(ps []claim.PrincipalID) claim.PrincipalID {
	if len(ps) == 0 {
		return ""
	}
	return ps[0]
}

// UsageStore is the ledger-append surface the cost recorder needs.
type UsageStore interface {
	RecordUsage(ctx context.Context, e pg.UsageEvent) error
}

// UsageRecorder implements nominate.UsageSink: it writes one inference event to
// the cost ledger per model call, attributed to the Input's principal and scope.
// This is the cost-attribution half of the usage limits — inference calls,
// tokens, and wall-clock, per principal and per scope.
type UsageRecorder struct {
	store UsageStore
	log   *slog.Logger
	now   func() time.Time
}

// NewUsageRecorder builds a UsageRecorder.
func NewUsageRecorder(store UsageStore, log *slog.Logger) *UsageRecorder {
	return &UsageRecorder{store: store, log: log, now: func() time.Time { return time.Now().UTC() }}
}

// Record implements nominate.UsageSink.
func (r *UsageRecorder) Record(ctx context.Context, in nominate.Input, u nominate.Usage) {
	err := r.store.RecordUsage(ctx, pg.UsageEvent{
		Principal:     firstPrincipal(in.Principals),
		Scope:         in.Scope,
		Kind:          "inference",
		InferenceCall: true,
		InputTokens:   u.InputTokens,
		OutputTokens:  u.OutputTokens,
		WallMS:        u.Wall.Milliseconds(),
		Now:           r.now(),
	})
	if err != nil && r.log != nil {
		// Best-effort — the tokens are already spent, so failing the nomination
		// over a ledger write helps nobody. But log it: the cost ceiling reads
		// this ledger, and a silently dropped record weakens the ceiling.
		r.log.Warn("usage record failed", slog.String("error", err.Error()))
	}
}

var _ nominate.UsageSink = (*UsageRecorder)(nil)

// LimitStore is what the contribution/cost gate counts and records through.
type LimitStore interface {
	ContributionsInWindow(ctx context.Context, principal claim.PrincipalID, since time.Time) (int, error)
	InferenceCostInWindow(ctx context.Context, principal claim.PrincipalID, since time.Time) (calls, tokens int, err error)
	RecordUsage(ctx context.Context, e pg.UsageEvent) error
}

// Limiter gates the automatic write path: before a captured source is nominated
// and written, it checks the principal's cost ceiling and contribution quota. On
// exhaustion it denies — and the denial is loud, never a silent no-op: it is
// recorded to the ledger and logged at Warn with the machine reason. Under the
// off-the-turn write path that loudness is the whole point. A silent
// drop of a write there is indistinguishable from a suppressed poisoning attempt,
// so exhaustion is made observable rather than swallowed.
type Limiter struct {
	store LimitStore
	cfg   limit.Config
	log   *slog.Logger
	now   func() time.Time
}

// NewLimiter builds a Limiter over cfg.
func NewLimiter(store LimitStore, cfg limit.Config, log *slog.Logger) *Limiter {
	return &Limiter{store: store, cfg: cfg, log: log, now: func() time.Time { return time.Now().UTC() }}
}

// Admit reports whether one automatic contribution by principal may proceed. It
// checks the cost ceiling first — so a principal already over it never spends
// another token — then the contribution quota. A denial is recorded and logged;
// Admit returns false with a nil error. A non-nil error is a transient counting
// failure the worker should retry, not a denial.
func (l *Limiter) Admit(ctx context.Context, principal claim.PrincipalID, scope claim.Scope) (bool, error) {
	now := l.now()

	calls, tokens, err := l.store.InferenceCostInWindow(ctx, principal, limit.WindowStart(now, l.cfg.CostWindow))
	if err != nil {
		return false, err
	}
	if d := limit.Cost(calls, tokens, l.cfg); !d.Allowed {
		l.deny(ctx, principal, scope, d.Reason, now)
		return false, nil
	}

	accepted, err := l.store.ContributionsInWindow(ctx, principal, limit.WindowStart(now, l.cfg.ContributionWindow))
	if err != nil {
		return false, err
	}
	if d := limit.Contribution(accepted, l.cfg); !d.Allowed {
		l.deny(ctx, principal, scope, d.Reason, now)
		return false, nil
	}
	return true, nil
}

// deny makes an exhaustion loud: a structured Warn (exported through obs'
// attribute names) and a recorded 'denied' ledger row queryable via `cred
// usage`. Never the source or the statement — only identifiers, the scope,
// and the machine reason.
func (l *Limiter) deny(ctx context.Context, principal claim.PrincipalID, scope claim.Scope, reason limit.Reason, now time.Time) {
	if l.log != nil {
		l.log.Warn("contribution denied",
			slog.String(obs.AttrPrincipalID, string(principal)),
			slog.String(obs.AttrScopeKind, string(scope.Kind)),
			slog.String(obs.AttrScopeValue, scope.Value),
			slog.String(obs.AttrDeniedReason, string(reason)))
	}
	err := l.store.RecordUsage(ctx, pg.UsageEvent{
		Principal:    principal,
		Scope:        scope,
		Kind:         "denied",
		DeniedReason: string(reason),
		Now:          now,
	})
	if err != nil && l.log != nil {
		// A denial the ledger failed to record is still loud in the log, but the
		// gap matters — say so at Error.
		l.log.Error("recording contribution denial failed", slog.String("error", err.Error()))
	}
}

// PruneStore is what the scope-growth pruner counts and expires through.
type PruneStore interface {
	ScopeClaimCount(ctx context.Context, scope claim.Scope) (int, error)
	PruneScope(ctx context.Context, scope claim.Scope, n int, now time.Time) (int, error)
}

// PruneReport is what one prune pass over a scope did.
type PruneReport struct {
	Scope  claim.Scope
	Live   int
	Pruned int
}

// Pruner enforces the scope-growth bound: when a scope exceeds its ceiling,
// pruning cuts it back — more aggressively the further over it is (internal/limit
// decides how many), rather than letting the scope grow without limit. Context
// grows roughly four times faster than it shrinks; the shrink is bounded by
// policy here, not left to hope.
type Pruner struct {
	store PruneStore
	cfg   limit.Config
	log   *slog.Logger
	now   func() time.Time
}

// NewPruner builds a Pruner over cfg.
func NewPruner(store PruneStore, cfg limit.Config, log *slog.Logger) *Pruner {
	return &Pruner{store: store, cfg: cfg, log: log, now: func() time.Time { return time.Now().UTC() }}
}

// Prune expires a scope's lowest-value live claims when it is over its ceiling.
// It makes no decision of its own: the count comes from the store, PruneTarget
// (pure) decides how many, and the store closes exactly that many.
func (p *Pruner) Prune(ctx context.Context, scope claim.Scope) (PruneReport, error) {
	live, err := p.store.ScopeClaimCount(ctx, scope)
	if err != nil {
		return PruneReport{}, err
	}
	rep := PruneReport{Scope: scope, Live: live}

	target := limit.PruneTarget(live, p.cfg)
	if target <= 0 {
		return rep, nil
	}
	n, err := p.store.PruneScope(ctx, scope, target, p.now())
	if err != nil {
		return rep, err
	}
	rep.Pruned = n
	if p.log != nil && n > 0 {
		// Identifiers and counts only — never claim text.
		p.log.Info("pruned scope",
			slog.String(obs.AttrScopeKind, string(scope.Kind)),
			slog.String(obs.AttrScopeValue, scope.Value),
			slog.Int(obs.AttrScopeLive, live),
			slog.Int(obs.AttrPrunedCount, n))
	}
	return rep, nil
}
