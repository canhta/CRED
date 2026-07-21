package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/recall"
)

func runRecall(ctx context.Context, args []string, cfg config.Config, out io.Writer) error {
	fs := flag.NewFlagSet("recall", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", 5, "maximum claims to return")
	depth := fs.Int("depth", recall.DefaultCandidateDepth, "candidates fetched per arm")
	budget := fs.Int("budget", recall.DefaultTokenBudget, "token ceiling for the assembled package")
	full := fs.Bool("full", false, "print each claim's full evidence text")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("%w: recall takes a query", ErrUsage)
	}
	query := strings.Join(fs.Args(), " ")

	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	emb, err := openEmbedder(ctx, cfg)
	if err != nil {
		return err
	}
	defer func() { _ = emb.Close() }()

	count, err := tokenCounter()
	if err != nil {
		return err
	}

	res, err := recall.New(st, emb, count).WithLimits(cfg.Limits, st).Recall(ctx, recall.Request{
		Query:     query,
		Principal: claim.PrincipalID(cfg.Principal),
		Limit:     *limit,
		Depth:     *depth,
		Budget:    *budget,
		Now:       time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("recall: %w", err)
	}

	printResult(out, res, *full)
	return nil
}

func printResult(out io.Writer, res *recall.Result, full bool) {
	if len(res.Claims) == 0 {
		fmt.Fprintf(out, "No claims matched.\n\n")
		fmt.Fprintf(out, "  %d candidates were retrieved and %d survived access control.\n",
			res.Candidates, res.Authorized)
		if res.Candidates > 0 && res.Authorized == 0 {
			fmt.Fprintf(out, "  Everything retrieved was denied. Check CRED_PRINCIPAL.\n")
		}
		if res.Candidates == 0 {
			fmt.Fprintf(out, "  Nothing was retrieved. Has anything been seeded? `cred seed .`\n")
		}
		return
	}

	for i, s := range res.Claims {
		fmt.Fprintf(out, "%d. %s\n", i+1, s.Claim.Statement)
		for _, e := range s.Claim.Evidence {
			// Show the tier-1 anchor when present: it is why the claim survives
			// edits elsewhere in the file, and it makes the anchor legible rather
			// than an opaque line range.
			if e.AnchorSymbolPath != "" {
				fmt.Fprintf(out, "   evidence  %s:%d-%d  [%s]\n",
					e.Path, e.LineStart, e.LineEnd, e.AnchorSymbolPath)
			} else {
				fmt.Fprintf(out, "   evidence  %s:%d-%d\n", e.Path, e.LineStart, e.LineEnd)
			}
		}
		fmt.Fprintf(out, "   score     %.6f  = %s\n", s.Score, explain(s.Contributions))
		fmt.Fprintf(out, "   kind      %s   scope %s=%s   tokens %d\n",
			s.Claim.Kind, s.Claim.Scope.Kind, s.Claim.Scope.Value, s.Tokens)
		if full {
			for _, e := range s.Claim.Evidence {
				fmt.Fprintf(out, "   ---\n%s\n   ---\n", indent(e.ExtractedText, "   | "))
			}
		}
		fmt.Fprintln(out)
	}

	fmt.Fprintf(out, "%d candidates -> %d authorized -> %d returned, %d omitted\n",
		res.Candidates, res.Authorized, len(res.Claims), res.OmittedForBudget)
	fmt.Fprintf(out, "tokens %d / %d   as_of %s   staleness %.0fs\n",
		res.TokensUsed, res.TokenBudget,
		res.AsOf.Format(time.RFC3339), res.StalenessSeconds)
	fmt.Fprintf(out, "latency total %s = embed %s + dense %s + lexical %s + load %s\n",
		ms(res.Timings.Total), ms(res.Timings.Embed), ms(res.Timings.Dense),
		ms(res.Timings.Lexical), ms(res.Timings.Load))

	// Above 0.95 the system is doing single-arm search with the latency of
	// two. Worth saying out loud rather than leaving in a metric nobody reads.
	if res.DominantShare > 0.95 {
		fmt.Fprintf(out, "\nwarning: %.0f%% of the top results came from the %s arm alone.\n",
			res.DominantShare*100, res.DominantArm)
		fmt.Fprintf(out, "         That is single-arm search paying for two.\n")
	}
}

func explain(cs []recall.Contribution) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = fmt.Sprintf("%s(rank %d, raw %.4f) %.6f", c.Arm, c.Rank, c.Raw, c.Score)
	}
	return strings.Join(parts, " + ")
}

func ms(d time.Duration) string {
	return fmt.Sprintf("%.1fms", float64(d.Microseconds())/1000.0)
}

func indent(s, prefix string) string {
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = prefix + l
	}
	return strings.Join(lines, "\n")
}
