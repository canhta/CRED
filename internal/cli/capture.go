package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/canhta/cred/internal/anchor"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/curate"
)

// runCapture is the automatic write path's entry point — the command an agent
// hook calls. It captures material (a tool result, a transcript slice, a file
// span) and enqueues a nomination job, then returns immediately. Extraction
// happens off the turn in `cred curate`, so the agent is never blocked.
//
// It reads the material from --text or, if absent, from stdin, so a hook can
// pipe a tool result straight in.
func runCapture(ctx context.Context, args []string, cfg config.Config, stdin io.Reader, out io.Writer) error {
	fs := flag.NewFlagSet("capture", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	text := fs.String("text", "", "material to extract from; if empty, read from stdin")
	repo := fs.String("repo", "", "repository the material came from")
	path := fs.String("path", "", "file path the material came from")
	baseLine := fs.Int("base-line", 1, "1-based line number of the material's first line")
	scopeKind := fs.String("scope-kind", "repository", "organization, repository, path, or service")
	scopeValue := fs.String("scope-value", "", "the scope value; defaults to --repo")
	sourceKind := fs.String("source-kind", "", "document, code, or attestation; inferred from --path when empty")
	trigger := fs.String("trigger", "tool_result", "what fired this: turn, tool_result, session_end")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}

	// Opt-out honoured here too, so a hook that forgets to check the env still
	// does nothing when the operator has turned auto-capture off.
	if !cfg.AutoCapture {
		fmt.Fprintf(out, "capture skipped: CRED_AUTO_CAPTURE is off\n")
		return nil
	}

	source := *text
	if source == "" {
		b, err := io.ReadAll(stdin)
		if err != nil {
			return fmt.Errorf("read material from stdin: %w", err)
		}
		source = string(b)
	}
	if strings.TrimSpace(source) == "" {
		fmt.Fprintf(out, "capture skipped: no material\n")
		return nil
	}

	sv := *scopeValue
	if sv == "" {
		sv = *repo
	}

	// An unset kind is read off the file extension, so a captured .ts span routes
	// to the code anchorer and a .md span to the text one without the hook
	// declaring it.
	sk := *sourceKind
	if sk == "" {
		sk = string(anchor.KindForPath(*path))
	}

	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	// The insert-only client is never started and needs no worker — enqueue and
	// return. It requires only the River tables, which `cred migrate` created.
	q, err := st.RiverInsertClient()
	if err != nil {
		return fmt.Errorf("capture: %w", err)
	}

	err = curate.EnqueueNominate(ctx, q, curate.NominateArgs{
		Source:     source,
		SourceKind: sk,
		Repo:       *repo,
		Path:       *path,
		BaseLine:   *baseLine,
		ScopeKind:  *scopeKind,
		ScopeValue: sv,
		Principals: []string{cfg.Principal},
		Trigger:    *trigger,
	})
	if err != nil {
		return fmt.Errorf("capture: enqueue: %w", err)
	}

	fmt.Fprintf(out, "captured for extraction (trigger=%s). Run `cred curate` to process the queue.\n", *trigger)
	return nil
}
