// Package cli implements CRED's subcommands.
//
// The CLI is not developer convenience. The most reliable failure mode across
// every surveyed memory system is silent write acceptance followed by empty
// reads — thousands of documents visible in a UI and zero results from search,
// with no way to tell why. A CLI that shows what was seeded, what matched, and
// what each arm contributed is the instrument those users did not have.
package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"

	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/obs"
)

// ErrUsage reports a command-line mistake, as distinct from a failure.
var ErrUsage = errors.New("usage")

const usage = `cred — evidence-governed memory for AI agents.

A claim lives only while its evidence does.

Usage:
  cred <command> [flags]

Commands:
  migrate            Apply database migrations
  seed <path>        Seed claims from a repository's documentation
  recall <query>     Retrieve claims, showing why each one ranked
  serve              Run the MCP server over stdio
  doctor             Check the installation and name the fix for anything broken

Environment:
  DATABASE_URL                 Postgres connection string
  CRED_MODEL_DIR               Directory holding model.onnx
  CRED_ALLOW_MODEL_DOWNLOAD    Fetch the model on first run (default true)
  CRED_PRINCIPAL               Identity recall is evaluated against
  CRED_LOG_LEVEL               debug, info, warn, error
  CRED_LOG_FORMAT              text or json

This build is read-only. There is no write path and no remember tool.
`

// Run dispatches a subcommand and returns a process exit code.
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) < 2 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(stderr, "cred: %v\n", err)
		return 1
	}

	// The MCP server speaks JSON-RPC on stdout, so every log line must go to
	// stderr. Writing a log line to stdout there corrupts the protocol stream
	// and presents as an unexplained client disconnect.
	log := obs.NewLogger(stderr, cfg.LogLevel, cfg.LogFormat)

	cmd, rest := args[1], args[2:]
	err = dispatch(ctx, cmd, rest, cfg, log, stdout, stderr)

	switch {
	case err == nil:
		return 0
	case errors.Is(err, ErrUsage):
		fmt.Fprintf(stderr, "cred: %v\n\n", err)
		fmt.Fprint(stderr, usage)
		return 2
	case errors.Is(err, errCheckFailed):
		// doctor already printed each failure and its fix.
		return 1
	case errors.Is(err, flag.ErrHelp):
		fmt.Fprint(stderr, usage)
		return 2
	default:
		fmt.Fprintf(stderr, "cred: %v\n", err)
		return 1
	}
}

func dispatch(ctx context.Context, cmd string, args []string, cfg config.Config,
	log *slog.Logger, stdout, stderr io.Writer,
) error {
	switch cmd {
	case "migrate":
		return runMigrate(ctx, args, cfg, stdout)
	case "seed":
		return runSeed(ctx, args, cfg, log, stdout)
	case "recall":
		return runRecall(ctx, args, cfg, stdout)
	case "serve":
		return runServe(ctx, args, cfg, log, stderr)
	case "doctor":
		return runDoctor(ctx, args, cfg, stdout)
	case "help", "-h", "--help":
		fmt.Fprint(stdout, usage)
		return nil
	default:
		return fmt.Errorf("%w: unknown command %q", ErrUsage, cmd)
	}
}
