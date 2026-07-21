package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/store/pg"
)

func runRemember(ctx context.Context, args []string, cfg config.Config, log *slog.Logger, out io.Writer) error {
	fs := flag.NewFlagSet("remember", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	kind := fs.String("kind", "", "claim kind: Convention, Decision, Constraint, RejectedApproach, Failure, Reference")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("%w: remember takes the claim text", ErrUsage)
	}
	statement := strings.Join(fs.Args(), " ")

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

	// Attestation is deterministic and key-free: the person asserting the claim
	// is its evidence (L1).
	id, err := newExecutor(st, emb, log).Attest(ctx, statement, *kind, claim.PrincipalID(cfg.Principal))
	if err != nil {
		return fmt.Errorf("remember: %w", err)
	}

	fmt.Fprintf(out, "remembered %s\n", id)
	fmt.Fprintf(out, "  See it:    cred log\n")
	fmt.Fprintf(out, "  Reverse:   cred forget %s\n", id)
	return nil
}

func runLog(ctx context.Context, args []string, cfg config.Config, out io.Writer) error {
	fs := flag.NewFlagSet("log", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	limit := fs.Int("limit", 20, "how many recent writes to show")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}

	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	entries, err := st.RecentWrites(ctx, *limit)
	if err != nil {
		return fmt.Errorf("log: %w", err)
	}
	if len(entries) == 0 {
		fmt.Fprintf(out, "No writes yet. Contribute one with `cred remember \"...\"`,\n")
		fmt.Fprintf(out, "or let the automatic path capture your work (see the hook example in the README).\n")
		return nil
	}

	// Every automatic write is visible here (D-016): what was stored, when, and
	// whether it is still live or was later superseded or forgotten.
	for _, e := range entries {
		status := "live"
		if e.SupersededAt != nil {
			status = e.SupersedeReason
			if status == "" {
				status = "superseded"
			}
		}
		fmt.Fprintf(out, "%s  %-10s  %-9s  %s\n",
			e.RecordedAt.Format(time.RFC3339), e.Kind, status, e.ID)
		fmt.Fprintf(out, "    %s\n", oneLine(e.Statement))
	}
	return nil
}

func runForget(ctx context.Context, args []string, cfg config.Config, out io.Writer) error {
	fs := flag.NewFlagSet("forget", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}
	if fs.NArg() != 1 {
		return fmt.Errorf("%w: forget takes exactly one claim id", ErrUsage)
	}
	id := fs.Arg(0)

	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	// Reversal expires the claim; it never deletes it. The record that it was
	// once believed survives, closed in transaction time.
	if err := st.ForgetClaim(ctx, id, time.Now().UTC()); err != nil {
		if errors.Is(err, pg.ErrNotFound) {
			return fmt.Errorf("forget: no live claim with id %s (already forgotten, superseded, or never existed)", id)
		}
		return fmt.Errorf("forget: %w", err)
	}
	fmt.Fprintf(out, "forgot %s\n", id)
	return nil
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len([]rune(s)) > 100 {
		s = string([]rune(s)[:99]) + "…"
	}
	return s
}
