package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/seed"
)

func runSeed(ctx context.Context, args []string, cfg config.Config,
	log *slog.Logger, out io.Writer,
) error {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: seed takes exactly one path", ErrUsage)
	}
	root := fs.Arg(0)

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

	fmt.Fprintf(out, "seeding    scanning %s\n", root)

	rep, err := seed.New(st, emb, log).Run(ctx, root,
		[]claim.PrincipalID{claim.PrincipalID(cfg.Principal)})
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}

	fmt.Fprintf(out, "seeding    %d files, %d chunks -> %d written, %d superseded, %d unchanged (%s)\n",
		rep.Files, rep.Chunks, rep.Inserted, rep.Superseded, rep.Unchanged,
		rep.Duration.Round(1e6))

	claims, evidence, err := st.Counts(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "store      %d live claims, %d live evidence rows\n", claims, evidence)
	fmt.Fprintf(out, "\n  Try it:\n    cred recall \"what are the design laws\"\n")
	return nil
}
