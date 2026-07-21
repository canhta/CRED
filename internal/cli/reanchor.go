package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/curate"
)

// runReanchor re-resolves every live claim's anchor against the current files
// under a root and expires the ones whose evidence no longer holds (semantic
// change or ambiguous), leaving formatting-only changes untouched.
//
// It is a CLI subcommand rather than a curate worker on purpose. Re-anchoring is
// triggered by a file changing on disk — an external event a background daemon
// cannot observe without watching the filesystem, which is a dependency and a
// footgun this project does not want. `cred reanchor <root>` is the deterministic
// primitive a CI step or a post-edit git hook invokes, the same shape as `cred
// seed`. It needs no API key and no model: invalidation is deterministic, so
// nothing here crosses the LLM boundary or embeds.
func runReanchor(ctx context.Context, args []string, cfg config.Config, log *slog.Logger, out io.Writer) error {
	fs := flag.NewFlagSet("reanchor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	quiet := fs.Bool("quiet", false, "print only the summary line, not each expiry")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}
	root := "."
	if fs.NArg() == 1 {
		root = fs.Arg(0)
	} else if fs.NArg() > 1 {
		return fmt.Errorf("%w: reanchor takes at most one path", ErrUsage)
	}

	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	rep, err := curate.NewReanchorer(st, log).Reanchor(ctx, root)
	if err != nil {
		return fmt.Errorf("reanchor: %w", err)
	}

	if !*quiet {
		// One line per expiry: the reason a claim died is a diff, not a score, so
		// it is worth printing.
		for _, d := range rep.Details {
			fmt.Fprintf(out, "%s\n", d)
		}
	}
	fmt.Fprintf(out, "checked %d  valid %d  expired %d", rep.Checked, rep.Valid, rep.Expired)
	if rep.MissingFile > 0 {
		fmt.Fprintf(out, "  missing-file %d (left untouched)", rep.MissingFile)
	}
	fmt.Fprintf(out, "  in %s\n", ms(rep.Duration))
	if rep.Checked == 0 {
		fmt.Fprintf(out, "\nNothing anchored under %s. Seed first with `cred seed %s`.\n", rep.Repo, root)
	}
	return nil
}
