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
  migrate            Apply database migrations (CRED schema + River tables)
  seed <path>        Seed claims from a repository's documentation
  recall <query>     Retrieve claims, showing why each one ranked
  reanchor [path]    Re-check anchors (L3); expire claims whose source changed
  remember <text>    Contribute a claim by attestation (no API key needed)
  capture            Enqueue material for automatic extraction (hook entry point)
  curate             Run the background worker: nominate off the turn, dedup
  log                Show recent writes (visible, per D-016)
  forget <id>        Reverse a write by expiring its claim (per D-016)
  usage              Show per-principal quota state and per-scope cost (section 8)
  serve              Run the MCP server over stdio (recall + remember)
  doctor             Check the installation and name the fix for anything broken

Environment:
  DATABASE_URL                 Postgres connection string
  CRED_MODEL_DIR               Directory holding model.onnx
  CRED_ALLOW_MODEL_DOWNLOAD    Fetch the model on first run (default true)
  CRED_PRINCIPAL               Identity recall is evaluated against
  CRED_LOG_LEVEL               debug, info, warn, error
  CRED_LOG_FORMAT              text or json
  CRED_AUTO_CAPTURE            Automatic nomination on capture (default true)
  CRED_LLM_API_KEY             Model key for the curate worker (or ANTHROPIC_API_KEY)
  CRED_LLM_MODEL               Model id for nomination (default claude-opus-4-8)
  CRED_CONTRIBUTION_QUOTA      Accepted claims per principal per window (default 120)
  CRED_COST_MAX_CALLS          Inference calls per principal per window (default 500)
  CRED_COST_MAX_TOKENS         Input tokens per principal per window (default 2000000)
  CRED_RECALL_RATE             Recalls per principal per window (default 120)
  CRED_SCOPE_CLAIM_CEILING     Live claims per scope before pruning (default 5000)

The usage limits (section 8) ship on by default with working ceilings; a
non-positive override disables that one control. See them with cred usage.

Reads and explicit remember need no API key. Only curate — the automatic
nomination worker — does. Every write is visible (cred log) and reversible
(cred forget).
`

// Run dispatches a subcommand and returns a process exit code.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
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
	err = dispatch(ctx, cmd, rest, cfg, log, stdin, stdout, stderr)

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
	log *slog.Logger, stdin io.Reader, stdout, stderr io.Writer,
) error {
	switch cmd {
	case "migrate":
		return runMigrate(ctx, args, cfg, stdout)
	case "seed":
		return runSeed(ctx, args, cfg, log, stdout)
	case "recall":
		return runRecall(ctx, args, cfg, stdout)
	case "reanchor":
		return runReanchor(ctx, args, cfg, log, stdout)
	case "remember":
		return runRemember(ctx, args, cfg, log, stdout)
	case "capture":
		return runCapture(ctx, args, cfg, stdin, stdout)
	case "curate":
		return runCurate(ctx, args, cfg, log, stderr)
	case "log":
		return runLog(ctx, args, cfg, stdout)
	case "forget":
		return runForget(ctx, args, cfg, stdout)
	case "usage":
		return runUsage(ctx, args, cfg, stdout)
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
