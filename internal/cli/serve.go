package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/mcpsrv"
	"github.com/canhta/cred/internal/recall"
)

func runServe(ctx context.Context, args []string, cfg config.Config,
	log *slog.Logger, stderr io.Writer,
) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}

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

	// The startup banner goes to stderr. stdout carries JSON-RPC frames, and
	// one stray line there presents to the client as an unexplained
	// disconnect rather than as a parse error.
	claims, evidence, err := st.Counts(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(stderr, "cred %s  mcp stdio  1 tool (recall, read-only)  %d claims, %d evidence\n",
		mcpsrv.Version, claims, evidence)

	svc := recall.New(st, emb, count)
	srv := mcpsrv.New(svc, claim.PrincipalID(cfg.Principal), log)

	// ctx is cancelled by the signal handler in main. Run returns when the
	// client disconnects or the context is done; the deferred Close calls
	// above are the shutdown path, in reverse construction order.
	if err := srv.ServeStdio(ctx); err != nil {
		return fmt.Errorf("mcp server: %w", err)
	}
	return nil
}
