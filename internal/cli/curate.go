package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/curate"
	"github.com/canhta/cred/internal/nominate"
)

// runCurate runs the background worker: it drains nomination jobs (crossing the
// LLM boundary, off the turn) and runs the exact-hash deduplication pass. It is
// the long-running half of the write path; `cred capture` feeds it.
//
// This is the one command that requires an API key, because it is the one that
// nominates. The read path and explicit `remember` need none.
func runCurate(ctx context.Context, args []string, cfg config.Config, log *slog.Logger, stderr io.Writer) error {
	fs := flag.NewFlagSet("curate", flag.ContinueOnError)
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

	nom, err := newNominator(cfg, st, log)
	if err != nil {
		return err
	}

	exec := newExecutor(st, emb, log)
	rec := curate.NewReconciler(st, log)
	// The section-8 write-path controls: the contribution-quota/cost-ceiling gate
	// and the scope-growth pruner, both over the resolved policy.
	limiter := curate.NewLimiter(st, cfg.Limits, log)
	pruner := curate.NewPruner(st, cfg.Limits, log)

	// The worker enqueues its own dedup and prune passes after a write, so it
	// needs an insert handle in addition to the worker client. Both share the pool.
	queue, err := st.RiverInsertClient()
	if err != nil {
		return fmt.Errorf("curate: %w", err)
	}

	workers := curate.Register(nom, exec, rec, pruner, limiter, queue, log)
	client, err := st.RiverWorkerClient(workers, log)
	if err != nil {
		return fmt.Errorf("curate: %w", err)
	}

	fmt.Fprintf(stderr, "cred curate  workers: nominate, dedup, prune  model %s\n", modelLabel(cfg))

	// ctx is cancelled by the signal handler in main. Start returns once the
	// client is running; the worker keeps going until ctx is done, then Stop
	// drains in-flight jobs. Whoever starts the goroutine owns its shutdown, and
	// that owner is River — we drive its lifecycle, not our own goroutines.
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("curate: start: %w", err)
	}
	<-ctx.Done()

	// A fresh context for shutdown: the signalled ctx is already done, and Stop
	// needs a live one to drain.
	if err := client.Stop(context.WithoutCancel(ctx)); err != nil {
		return fmt.Errorf("curate: stop: %w", err)
	}
	return nil
}

func modelLabel(cfg config.Config) string {
	if cfg.LLMModel != "" {
		if cfg.LLMBaseURL != "" {
			return cfg.LLMModel + " @ " + cfg.LLMBaseURL
		}
		return cfg.LLMModel
	}
	return nominate.DefaultModel + " (default)"
}
