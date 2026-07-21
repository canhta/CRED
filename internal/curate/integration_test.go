//go:build integration

// Package curate's integration suite. Build-tagged so `go test ./...` stays
// under the per-commit budget and this runs only where a database is expected.
//
// It exercises the write path end to end against a real Postgres and a real
// River queue, but with the fake nominator — so no API key is needed even
// though the production write path requires one. That is the property the whole
// package is arranged to preserve: the read path's zero-config guarantee is not
// regressed by the tests for the write path.
package curate_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/curate"
	"github.com/canhta/cred/internal/embed"
	"github.com/canhta/cred/internal/nominate"
	"github.com/canhta/cred/internal/store/pg"
)

func requireDB() bool { return os.Getenv("CRED_REQUIRE_DB") == "1" }

func databaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return config.DefaultDatabaseURL
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func openStore(t *testing.T) *pg.Store {
	t.Helper()
	st, err := pg.Open(t.Context(), databaseURL())
	if err != nil {
		if requireDB() {
			t.Fatalf("CRED_REQUIRE_DB=1 and Postgres is unreachable: %v", err)
		}
		t.Skipf("Postgres is not reachable, skipping: %v", err)
	}
	t.Cleanup(st.Close)
	if _, err := st.Migrate(t.Context()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := st.MigrateRiver(t.Context()); err != nil {
		t.Fatalf("river migrate: %v", err)
	}
	return st
}

func newEmbedder(t *testing.T) *embed.BGE {
	t.Helper()
	path, err := embed.ModelPath(t.Context(), "", true)
	if err != nil {
		if requireDB() {
			t.Fatalf("CRED_REQUIRE_DB=1 and the model is unavailable: %v", err)
		}
		t.Skipf("embedding model unavailable, skipping: %v", err)
	}
	e, err := embed.NewBGE(path)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, e.Close()) })
	return e
}

// startWorker builds and starts the curate worker with the given nominator.
func startWorker(t *testing.T, st *pg.Store, nom nominate.Nominator) {
	t.Helper()
	emb := newEmbedder(t)
	exec := curate.NewExecutor(st, emb, testLogger())
	rec := curate.NewReconciler(st, testLogger())
	q, err := st.RiverInsertClient()
	require.NoError(t, err)
	workers := curate.Register(nom, exec, rec, q, testLogger())
	client, err := st.RiverWorkerClient(workers, testLogger())
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	require.NoError(t, client.Start(ctx))
	t.Cleanup(func() {
		cancel()
		_ = client.Stop(context.Background())
	})
}

// waitForWrite polls the write log until a claim whose statement contains marker
// appears, or the deadline passes.
func waitForWrite(t *testing.T, st *pg.Store, marker string) pg.WriteEntry {
	t.Helper()
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := st.RecentWrites(t.Context(), 100)
		require.NoError(t, err)
		for _, e := range entries {
			if strings.Contains(e.Statement, marker) {
				return e
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("no write containing %q appeared within the deadline", marker)
	return pg.WriteEntry{}
}

// TestEnqueueNominateWritesClaimWithEvidence is the write-path vertical: a
// trigger enqueues captured material, the worker extracts it off the turn with
// the fake nominator, and a claim lands with evidence pointing back at the
// source span (L1).
func TestEnqueueNominateWritesClaimWithEvidence(t *testing.T) {
	st := openStore(t)
	marker := fmt.Sprintf("curate-int-%d", time.Now().UnixNano())
	line := "the deploy step " + marker + " runs migrations before serving"

	startWorker(t, st, &nominate.Fake{Kind: claim.KindConvention})

	q, err := st.RiverInsertClient()
	require.NoError(t, err)
	err = curate.EnqueueNominate(t.Context(), q, curate.NominateArgs{
		Source:     "# notes\n" + line + "\n",
		SourceKind: "document",
		Repo:       "r", Path: "notes.md", BaseLine: 1,
		ScopeKind: "repository", ScopeValue: "r",
		Principals: []string{"local"},
		Trigger:    "tool_result",
	})
	require.NoError(t, err)

	entry := waitForWrite(t, st, marker)
	require.Equal(t, "Convention", entry.Kind)
	require.Nil(t, entry.SupersededAt, "a fresh write is live")

	// The claim reached the store with evidence, and the evidence is the source
	// span the fake quoted — not free text (L1, L8).
	ev, err := st.LoadEvidence(t.Context(), []string{entry.ID})
	require.NoError(t, err)
	require.NotEmpty(t, ev[entry.ID], "L1: a written claim reached the store with no evidence")
	require.Contains(t, ev[entry.ID][0].ExtractedText, marker)
	require.Positive(t, ev[entry.ID][0].LineStart)
}

// TestDedupSupersedesDuplicateWrites — automatic writes at every third turn
// produce duplicates by construction (D-017). The dedup pass folds an exact
// restatement into its predecessor via the bi-temporal machinery, leaving one
// live claim and one superseded, linked.
func TestDedupSupersedesDuplicateWrites(t *testing.T) {
	st := openStore(t)
	marker := fmt.Sprintf("dedup-int-%d", time.Now().UnixNano())
	// Same normalized statement, different surface whitespace/case.
	stmtA := "the cache TTL " + marker + " is five minutes"
	stmtB := "the   cache ttl " + marker + " IS five minutes"

	startWorker(t, st, &nominate.Fake{Kind: claim.KindConstraint})
	q, err := st.RiverInsertClient()
	require.NoError(t, err)

	enqueue := func(line string) {
		require.NoError(t, curate.EnqueueNominate(t.Context(), q, curate.NominateArgs{
			Source: line + "\n", SourceKind: "document",
			Repo: "r", Path: "conf.md", BaseLine: 1,
			ScopeKind: "repository", ScopeValue: "r",
			Principals: []string{"local"}, Trigger: "turn",
		}))
	}
	enqueue(stmtA)
	waitForWrite(t, st, marker)
	enqueue(stmtB)

	// Wait until exactly one of the two is superseded with reason 'duplicate'.
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		entries, err := st.RecentWrites(t.Context(), 100)
		require.NoError(t, err)
		var live, deduped int
		for _, e := range entries {
			if !strings.Contains(e.Statement, marker) {
				continue
			}
			if e.SupersededAt == nil {
				live++
			} else if e.SupersedeReason == curate.SupersedeReasonDuplicate {
				deduped++
			}
		}
		if live == 1 && deduped == 1 {
			return // dedup collapsed the pair, keeping one, linking the other
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("dedup did not collapse the duplicate pair within the deadline")
}
