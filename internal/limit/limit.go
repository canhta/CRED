// Package limit is CRED's usage-and-limits policy, as pure functions over
// counters and configuration.
//
// Every limit here is a security control first and a capacity concern second.
// Shared memory with unbounded per-principal write access is a poisoning
// vector: dedup has a threshold, and a threshold can always be approached from
// below, so the contribution quota is the backstop dedup cannot be.
//
// This package is pure. It imports no database driver, no database/sql, and no
// pgx, and takes no connection in any signature — depguard fails the build if
// that changes, exactly as it does for internal/temporal and internal/acl. The
// reason is the same: a limit expressed as a SQL predicate is a limit that
// cannot be unit-tested at its boundaries, and Postgres is here to store the
// counters, not to decide. The store counts rows; this package decides.
package limit

import (
	"math"
	"time"
)

// Reason names why an operation was denied. It is machine-stable — the CLI, the
// logs, and the usage ledger all key on it — and carries no content, so it is
// safe to export through telemetry and surface to a caller.
type Reason string

const (
	ReasonNone              Reason = ""
	ReasonContributionQuota Reason = "contribution_quota"
	ReasonCostCeiling       Reason = "cost_ceiling"
	ReasonRecallRate        Reason = "recall_rate"
	ReasonLoginAttempts     Reason = "login_attempts"
)

// Config is the resolved policy. Every field has a working default (see
// Defaults), because a limit that requires configuration to exist is a limit
// that is off on first run — and zero-config must not regress. A non-positive
// value on any ceiling disables that control, which is how an operator opts out
// of one limit without disabling the rest.
type Config struct {
	// Contribution quota: accepted claims per principal per window. The
	// sybil/repetition defence.
	ContributionQuota  int
	ContributionWindow time.Duration

	// Cost ceiling: inference calls and tokens per principal per window. The
	// hard cost ceiling enforced in code.
	MaxInferenceCalls int
	MaxInputTokens    int
	CostWindow        time.Duration

	// Recall budget: recall rate per principal per window, and the assembled
	// package's per-recall claim cap. Protects tail latency from a recall loop.
	RecallRate       int
	RecallWindow     time.Duration
	MaxPackageClaims int

	// Login attempts: failed logins per email per window. The brute-force /
	// credential-stuffing defence on the one unauthenticated write path.
	MaxLoginAttempts int
	LoginWindow      time.Duration

	// Scope growth bound: the per-scope live-claim ceiling, and how aggressively
	// pruning cuts back once it is exceeded. Context grows roughly four times
	// faster than it shrinks; growth is bounded by policy, not by hope.
	ScopeClaimCeiling   int
	PruneAggressiveness float64
}

// Defaults returns a policy sized so a single developer never hits a limit in
// ordinary use, while a tight loop is capped within one window. The numbers are
// deliberately generous: a false denial on a real contribution is worse than a
// late one on an attack, because the attack is also bounded by cost and by
// dedup below it. They are overridable through CRED_* configuration.
func Defaults() Config {
	return Config{
		// Automatic write fires at every third turn. A busy hour of real
		// work is a few dozen accepted claims; 120/hour leaves headroom while
		// still capping a repetition loop to one window's worth.
		ContributionQuota:  120,
		ContributionWindow: time.Hour,

		// The hard cost ceiling. One nomination is one call of up to ~2k output
		// tokens; 500 calls/hour and 2M input tokens/hour bound a runaway worker
		// well below a surprising bill.
		MaxInferenceCalls: 500,
		MaxInputTokens:    2_000_000,
		CostWindow:        time.Hour,

		// A recall loop is the tail-latency threat. 120/minute is two sustained
		// per second, far above interactive use and far below a hot loop.
		RecallRate:       120,
		RecallWindow:     time.Minute,
		MaxPackageClaims: 50,

		// Ten failed attempts per email per fifteen minutes: well above a
		// human mistyping a password twice, well below a scripted attempt
		// loop.
		MaxLoginAttempts: 10,
		LoginWindow:      15 * time.Minute,

		// A scope holding thousands of live claims is past the point where recall
		// stays sharp; prune back toward the ceiling once it is crossed.
		ScopeClaimCeiling:   5000,
		PruneAggressiveness: 0.5,
	}
}

// Decision is the outcome of a limit check. Remaining is how many more
// operations the principal may perform before the control binds; it is what
// makes quota state visible before it is hit. Remaining is -1 when the
// control is disabled (unlimited).
type Decision struct {
	Allowed   bool
	Reason    Reason
	Remaining int
}

// WindowStart is the half-open lower bound of a limit window: an event is in
// window iff its timestamp is strictly after WindowStart(now, window). Half-open
// matches internal/temporal's interval convention, so an event exactly one
// window old has aged out rather than being double-counted on the boundary. The
// store's count query uses `recorded_at > WindowStart(...)`.
func WindowStart(now time.Time, window time.Duration) time.Time {
	return now.Add(-window)
}

// Contribution decides whether a principal may have one more claim accepted,
// given how many it has had accepted in the current window. This is the
// backstop dedup cannot be: a near-duplicate flood sitting just under the
// dedup threshold is still one accepted claim each, and this counts them.
func Contribution(acceptedInWindow int, cfg Config) Decision {
	return atMost(acceptedInWindow, cfg.ContributionQuota, ReasonContributionQuota)
}

// Cost decides whether a principal may make one more inference call, given the
// calls and input tokens it has already spent in the window. Either ceiling
// binds; the call ceiling is reported as Remaining because it is the one a
// human reasons about, but tokens are checked too and reported by `cred usage`.
func Cost(callsInWindow, inputTokensInWindow int, cfg Config) Decision {
	calls := atMost(callsInWindow, cfg.MaxInferenceCalls, ReasonCostCeiling)
	if !calls.Allowed {
		return calls
	}
	if cfg.MaxInputTokens > 0 && inputTokensInWindow >= cfg.MaxInputTokens {
		return Decision{Allowed: false, Reason: ReasonCostCeiling, Remaining: 0}
	}
	return calls
}

// RecallRate decides whether a principal may run one more recall, given how many
// it has run in the window. A denial here is a loud, synchronous error to the
// caller — recall is on the turn, unlike the write path.
func RecallRate(recallsInWindow int, cfg Config) Decision {
	return atMost(recallsInWindow, cfg.RecallRate, ReasonRecallRate)
}

// LoginAttempts decides whether one more login attempt for this email may
// proceed, given how many have failed in the current window. Only failures
// count toward the ceiling -- a user who logs in correctly every day must
// never be at risk of tripping a limit meant for attackers.
func LoginAttempts(failedInWindow int, cfg Config) Decision {
	return atMost(failedInWindow, cfg.MaxLoginAttempts, ReasonLoginAttempts)
}

// PackageCap is the per-recall assembled-package claim cap, enforced
// server-side regardless of what the client requests. Zero or negative means no
// cap.
func PackageCap(cfg Config) int { return cfg.MaxPackageClaims }

// PruneTarget is how many of a scope's lowest-value live claims to expire, given
// its current live count. Under the ceiling it prunes nothing. Over the ceiling
// it prunes the whole overage plus extra headroom, and the headroom grows with
// the overage — so a scope that keeps growing is cut back harder each pass
// rather than tracked just under its limit forever. The result is bounded by the
// live count.
//
// This is the "growth bounded by policy" function: the aggressiveness is a
// constant an operator sets, not an emergent property of how fast writes arrive.
func PruneTarget(liveInScope int, cfg Config) int {
	if cfg.ScopeClaimCeiling <= 0 {
		return 0 // control disabled
	}
	over := liveInScope - cfg.ScopeClaimCeiling
	if over <= 0 {
		return 0
	}
	headroom := int(math.Ceil(float64(over) * cfg.PruneAggressiveness))
	target := over + headroom
	if target > liveInScope {
		target = liveInScope
	}
	return target
}

// atMost is the shared "count strictly below ceiling" decision. A non-positive
// ceiling disables the control (unlimited, Remaining -1). At the ceiling exactly
// the next operation is denied; one below, it is allowed with Remaining 1.
func atMost(inWindow, ceiling int, reason Reason) Decision {
	if ceiling <= 0 {
		return Decision{Allowed: true, Reason: ReasonNone, Remaining: -1}
	}
	remaining := ceiling - inWindow
	if remaining <= 0 {
		return Decision{Allowed: false, Reason: reason, Remaining: 0}
	}
	return Decision{Allowed: true, Reason: ReasonNone, Remaining: remaining}
}
