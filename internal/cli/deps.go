package cli

import (
	"context"
	"fmt"

	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/embed"
	"github.com/canhta/cred/internal/embed/wordpiece"
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
