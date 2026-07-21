package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/curate"
	"github.com/canhta/cred/internal/embed"
	"github.com/canhta/cred/internal/embed/wordpiece"
	"github.com/canhta/cred/internal/nominate"
	"github.com/canhta/cred/internal/store/pg"
)

// openStore connects to Postgres.
func openStore(ctx context.Context, cfg config.Config) (*pg.Store, error) {
	st, err := pg.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("%w\n\n  Is Postgres running? `docker compose up -d db` starts one.\n"+
			"  DATABASE_URL is currently %s", err, redact(cfg.DatabaseURL))
	}
	return st, nil
}

// openEmbedder loads the local model.
//
// No API key is required and none is accepted. That is the whole point of the
// pure-Go path: nothing in this category ships config-free except the systems
// that never call a model, and the read path is where that advantage is either
// realised or wasted.
func openEmbedder(ctx context.Context, cfg config.Config) (*embed.BGE, error) {
	path, err := embed.ModelPath(ctx, cfg.ModelDir, cfg.AllowModelDownload)
	if err != nil {
		return nil, fmt.Errorf("%w\n\n  Set CRED_MODEL_DIR to a directory holding model.onnx,\n"+
			"  or allow the first-run download with CRED_ALLOW_MODEL_DOWNLOAD=true", err)
	}
	e, err := embed.NewBGE(path)
	if err != nil {
		return nil, fmt.Errorf("load embedding model: %w", err)
	}
	return e, nil
}

// newExecutor builds the deterministic write executor. It needs a store and an
// embedder — no API key, because attestation and the deterministic write of a
// validated candidate never call a model. The caller owns closing the embedder.
func newExecutor(st *pg.Store, emb *embed.BGE, log *slog.Logger) *curate.Executor {
	return curate.NewExecutor(st, emb, log)
}

// newNominator builds the LLM boundary for the curate worker. This is the one
// place a key is required; if none is configured, the worker cannot nominate and
// says so loudly rather than degrading silently.
func newNominator(cfg config.Config, log *slog.Logger) (nominate.Nominator, error) {
	model, err := nominate.NewAnthropicModel(cfg.LLMAPIKey, nominate.WithModel(cfg.LLMModel))
	if err != nil {
		return nil, fmt.Errorf("%w\n\n  The automatic write path calls a model. Set CRED_LLM_API_KEY\n"+
			"  (or ANTHROPIC_API_KEY), or run the worker with CRED_AUTO_CAPTURE=false\n"+
			"  and use `cred remember` for explicit, key-free contribution", err)
	}
	return nominate.NewExtractor(model, nil, log), nil
}

// tokenCounter returns an exact model-token count, so the recall budget is
// measured rather than estimated.
func tokenCounter() (func(string) int, error) {
	tok, err := wordpiece.New()
	if err != nil {
		return nil, fmt.Errorf("tokenizer: %w", err)
	}
	return func(s string) int { return len(tok.Encode(s)) }, nil
}

// redact removes the password from a connection string before it reaches a
// terminal or a log.
func redact(url string) string {
	at := -1
	for i := len(url) - 1; i >= 0; i-- {
		if url[i] == '@' {
			at = i
			break
		}
	}
	if at < 0 {
		return url
	}
	scheme := 0
	for i := 0; i+2 < len(url); i++ {
		if url[i] == ':' && url[i+1] == '/' && url[i+2] == '/' {
			scheme = i + 3
			break
		}
	}
	if scheme == 0 || scheme >= at {
		return url
	}
	return url[:scheme] + "***@" + url[at+1:]
}
