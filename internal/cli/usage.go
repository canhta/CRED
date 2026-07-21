package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/limit"
)

// runUsage shows the usage state: the principal's remaining headroom on each
// limit before it is hit, and the per-scope cost that answers "which teams
// actually use this". The numbers come from the same store counts and the same
// internal/limit policy the enforcement path uses, so what a user sees here is
// exactly what the worker and recall paths decide over — quota state visible
// before it is hit, not a separate estimate that could disagree.
func runUsage(ctx context.Context, args []string, cfg config.Config, out io.Writer) error {
	fs := flag.NewFlagSet("usage", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	topN := fs.Int("scopes", 10, "how many scopes to list for cost and growth")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}

	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	principal := claim.PrincipalID(cfg.Principal)
	now := time.Now().UTC()
	lc := cfg.Limits

	state, err := st.PrincipalWindowState(ctx, principal,
		limit.WindowStart(now, lc.ContributionWindow),
		limit.WindowStart(now, lc.CostWindow),
		limit.WindowStart(now, lc.RecallWindow))
	if err != nil {
		return fmt.Errorf("usage: %w", err)
	}

	contribution := limit.Contribution(state.Contributions, lc)
	cost := limit.Cost(state.InferenceCall, state.InputTokens, lc)
	recall := limit.RecallRate(state.Recalls, lc)

	fmt.Fprintf(out, "usage for principal %q\n\n", cfg.Principal)

	tw := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintf(tw, "LIMIT\tWINDOW\tUSED\tCEILING\tREMAINING\n")
	fmt.Fprintf(tw, "contribution\t%s\t%d\t%s\t%s\n",
		lc.ContributionWindow, state.Contributions,
		ceilingLabel(lc.ContributionQuota), remainingLabel(contribution))
	fmt.Fprintf(tw, "inference calls\t%s\t%d\t%s\t%s\n",
		lc.CostWindow, state.InferenceCall,
		ceilingLabel(lc.MaxInferenceCalls), remainingLabel(cost))
	fmt.Fprintf(tw, "inference tokens\t%s\t%d\t%s\t%s\n",
		lc.CostWindow, state.InputTokens, ceilingLabel(lc.MaxInputTokens), "-")
	fmt.Fprintf(tw, "recall\t%s\t%d\t%s\t%s\n",
		lc.RecallWindow, state.Recalls,
		ceilingLabel(lc.RecallRate), remainingLabel(recall))
	_ = tw.Flush()

	// Exhaustion is loud even here: if any limit is already hit, say so plainly
	// rather than leaving the reader to compare columns.
	for _, d := range []limit.Decision{contribution, cost, recall} {
		if !d.Allowed {
			fmt.Fprintf(out, "\nEXHAUSTED: %s. Further operations are denied until the window rolls.\n", d.Reason)
			break
		}
	}

	// Denials already recorded in the contribution window — the on-the-record
	// half of "never a silent drop". A non-zero count here is a write the quota
	// or cost ceiling refused off the turn.
	denied, err := st.DeniedInWindow(ctx, principal, limit.WindowStart(now, lc.ContributionWindow))
	if err != nil {
		return fmt.Errorf("usage: %w", err)
	}
	if denied > 0 {
		fmt.Fprintf(out, "\n%d contribution(s) denied and recorded in the last %s.\n", denied, lc.ContributionWindow)
	}

	// "Which teams actually use this": inference cost grouped by scope.
	costs, err := st.UsageByScope(ctx, limit.WindowStart(now, lc.CostWindow))
	if err != nil {
		return fmt.Errorf("usage: %w", err)
	}
	fmt.Fprintf(out, "\ncost by scope (last %s):\n", lc.CostWindow)
	if len(costs) == 0 {
		fmt.Fprintf(out, "  none recorded\n")
	} else {
		tw = tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
		fmt.Fprintf(tw, "  SCOPE\tCALLS\tINPUT TOKENS\tOUTPUT TOKENS\n")
		for i, c := range costs {
			if i >= *topN {
				break
			}
			fmt.Fprintf(tw, "  %s=%s\t%d\t%d\t%d\n",
				c.Scope.Kind, c.Scope.Value, c.Calls, c.InputTokens, c.OutputTokens)
		}
		_ = tw.Flush()
	}

	// Scope growth: the largest scopes against the ceiling, so a scope
	// approaching its bound is visible before pruning has to bite.
	sizes, err := st.ScopeSizes(ctx, *topN)
	if err != nil {
		return fmt.Errorf("usage: %w", err)
	}
	fmt.Fprintf(out, "\nscope growth (ceiling %s):\n", ceilingLabel(lc.ScopeClaimCeiling))
	if len(sizes) == 0 {
		fmt.Fprintf(out, "  no live claims\n")
	} else {
		tw = tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
		fmt.Fprintf(tw, "  SCOPE\tLIVE CLAIMS\tNEXT PRUNE\n")
		for _, s := range sizes {
			fmt.Fprintf(tw, "  %s=%s\t%d\t%d\n",
				s.Scope.Kind, s.Scope.Value, s.Live, limit.PruneTarget(s.Live, lc))
		}
		_ = tw.Flush()
	}
	return nil
}

// ceilingLabel renders a ceiling, showing a disabled control as "off" rather
// than a misleading 0.
func ceilingLabel(n int) string {
	if n <= 0 {
		return "off"
	}
	return fmt.Sprintf("%d", n)
}

// remainingLabel renders a decision's remaining headroom, with "unlimited" for a
// disabled control and "0 (denied)" at exhaustion.
func remainingLabel(d limit.Decision) string {
	switch {
	case d.Remaining < 0:
		return "unlimited"
	case !d.Allowed:
		return "0 (denied)"
	default:
		return fmt.Sprintf("%d", d.Remaining)
	}
}
