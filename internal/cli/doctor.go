package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/embed"
	"github.com/canhta/cred/internal/store/pg"
)

// errCheckFailed reports that at least one check failed. Run returns exit
// code 1 for it without printing anything further, because doctor has already
// printed each failure and its fix.
var errCheckFailed = errors.New("one or more checks failed")

// check is one diagnostic. Every check names its fix; a check that reports a
// problem without naming the fix has moved the work rather than done it.
type check struct {
	name   string
	detail string
	fix    string
	ok     bool
}

func runDoctor(ctx context.Context, args []string, cfg config.Config, out io.Writer) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}

	var checks []check
	add := func(c check) { checks = append(checks, c) }

	add(check{
		name:   "runtime",
		detail: fmt.Sprintf("%s %s/%s, cgo off", runtime.Version(), runtime.GOOS, runtime.GOARCH),
		ok:     true,
	})

	// Model, before the database: it is the check that can fail with no
	// network and it is the slower one to fix.
	modelPath, err := embed.ModelPath(ctx, cfg.ModelDir, cfg.AllowModelDownload)
	switch {
	case err != nil:
		add(check{
			name:   "model",
			detail: err.Error(),
			fix: "Place model.onnx in CRED_MODEL_DIR, or set\n" +
				"CRED_ALLOW_MODEL_DOWNLOAD=true to fetch it once (~127 MiB).",
		})
	default:
		add(check{
			name:   "model",
			detail: fmt.Sprintf("%s (384d, local, no API key) at %s", embed.ModelName, modelPath),
			ok:     true,
		})

		start := time.Now()
		e, eerr := embed.NewBGE(modelPath)
		if eerr != nil {
			add(check{
				name:   "embedding",
				detail: eerr.Error(),
				fix:    "The model file is present but did not load. Delete it and let doctor refetch it.",
			})
		} else {
			vecs, verr := e.Embed(ctx, []string{"cred doctor"})
			_ = e.Close()
			switch {
			case verr != nil:
				add(check{name: "embedding", detail: verr.Error(),
					fix: "The forward pass failed. Report this with the message above."})
			case len(vecs) != 1 || len(vecs[0]) != embed.Dimensions:
				add(check{name: "embedding", detail: "unexpected vector shape",
					fix: "The model file does not match the expected architecture."})
			default:
				add(check{
					name:   "embedding",
					detail: fmt.Sprintf("forward pass ok, %d dimensions, %s", embed.Dimensions, time.Since(start).Round(time.Millisecond)),
					ok:     true,
				})
			}
		}
	}

	st, err := pg.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		add(check{
			name:   "database",
			detail: fmt.Sprintf("%v (%s)", err, redact(cfg.DatabaseURL)),
			fix: "Start Postgres with `docker compose up -d db`, or point\n" +
				"DATABASE_URL at an existing PostgreSQL 17 instance.",
		})
		return report(out, checks)
	}
	defer st.Close()

	version, err := st.Version(ctx)
	if err != nil {
		add(check{name: "database", detail: err.Error(), fix: "Check DATABASE_URL and the server logs."})
	} else {
		add(check{name: "database", detail: shortVersion(version), ok: true})
	}

	ext, err := st.VectorExtension(ctx)
	switch {
	case errors.Is(err, pg.ErrExtensionMissing):
		add(check{
			name:   "extension",
			detail: "pgvector is not installed in this database",
			// pgvector is not a "trusted" extension, so CREATE EXTENSION
			// needs superuser. On managed Postgres it is never created
			// automatically; the exact command is printed instead.
			fix: "pgvector is not a \"trusted\" extension, so CREATE EXTENSION\n" +
				"requires superuser. Ask a DBA to run:\n\n" +
				"    CREATE EXTENSION IF NOT EXISTS vector;",
		})
	case err != nil:
		add(check{name: "extension", detail: err.Error(), fix: "Check database permissions."})
	default:
		add(check{name: "extension", detail: "vector " + ext + " ready", ok: true})
	}

	schemaVersion, err := st.SchemaVersion(ctx)
	if err != nil {
		add(check{name: "schema", detail: err.Error(), fix: "Run `cred migrate`."})
	} else {
		pending, perr := st.PendingMigrations(ctx)
		switch {
		case perr != nil:
			add(check{name: "schema", detail: perr.Error(), fix: "Run `cred migrate`."})
		case pending > 0:
			add(check{
				name:   "schema",
				detail: fmt.Sprintf("at version %d, %d migrations pending", schemaVersion, pending),
				fix:    "Run `cred migrate`.",
			})
		default:
			add(check{
				name:   "schema",
				detail: fmt.Sprintf("version %d, up to date", schemaVersion),
				ok:     true,
			})
		}
	}

	if id, name, dims, merr := st.PresentModel(ctx); merr != nil {
		add(check{
			name:   "embedding model",
			detail: merr.Error(),
			fix:    "Run `cred migrate` to register the embedding model.",
		})
	} else {
		c := check{
			name:   "embedding model",
			detail: fmt.Sprintf("id %d, %s, %d dimensions, status PRESENT", id, name, dims),
			ok:     name == embed.ModelName && dims == embed.Dimensions,
		}
		if !c.ok {
			c.fix = fmt.Sprintf(
				"The database expects %s at %d dimensions but this binary ships %s at %d.\n"+
					"Reads filter on the model, so they would silently return nothing.",
				name, dims, embed.ModelName, embed.Dimensions)
		}
		add(c)
	}

	if claims, evidence, cerr := st.Counts(ctx); cerr != nil {
		add(check{name: "content", detail: cerr.Error(), fix: "Run `cred migrate`."})
	} else if claims == 0 {
		add(check{
			name:   "content",
			detail: "no claims stored",
			fix:    "Seed a repository: `cred seed /path/to/repo`.",
		})
	} else {
		add(check{
			name:   "content",
			detail: fmt.Sprintf("%d live claims, %d live evidence rows", claims, evidence),
			ok:     true,
		})
	}

	return report(out, checks)
}

func report(out io.Writer, checks []check) error {
	failed := 0
	for _, c := range checks {
		mark := "✘"
		if c.ok {
			mark = "✔"
		} else {
			failed++
		}
		fmt.Fprintf(out, "  %s %-17s %s\n", mark, c.name, c.detail)
		if !c.ok && c.fix != "" {
			for _, line := range strings.Split(c.fix, "\n") {
				fmt.Fprintf(out, "  %-19s %s\n", "", line)
			}
			fmt.Fprintln(out)
		}
	}
	if failed > 0 {
		fmt.Fprintf(out, "\n%d of %d checks failed.\n", failed, len(checks))
		return errCheckFailed
	}
	fmt.Fprintf(out, "\nAll %d checks passed. Connect an agent:\n", len(checks))
	fmt.Fprintf(out, "  claude mcp add cred -- %s serve\n", executable())
	return nil
}

func shortVersion(v string) string {
	if i := strings.Index(v, " on "); i > 0 {
		return v[:i]
	}
	return v
}

func executable() string {
	p, err := os.Executable()
	if err != nil {
		return "cred"
	}
	return p
}
