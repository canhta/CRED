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
	"github.com/canhta/cred/internal/limit"
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

// startWorker builds and starts the curate worker with the given nominator and
// usage-limit policy. Pass limit.Defaults() when limits should not interfere.
func startWorker(t *testing.T, st *pg.Store, nom nominate.Nominator, cfg limit.Config) {
	t.Helper()
	emb := newEmbedder(t)
	exec := curate.NewExecutor(st, emb, testLogger())
	rec := curate.NewReconciler(st, testLogger())
	limiter := curate.NewLimiter(st, cfg, testLogger())
	pruner := curate.NewPruner(st, cfg, testLogger())
	q, err := st.RiverInsertClient()
	require.NoError(t, err)
	workers := curate.Register(nom, exec, rec, pruner, limiter, q, testLogger())
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

	startWorker(t, st, &nominate.Fake{Kind: claim.KindConvention}, limit.Defaults())

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

	startWorker(t, st, &nominate.Fake{Kind: claim.KindConstraint}, limit.Defaults())
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

// insertPrincipal adds a grantable principal so the write path's claim_acl FK is
// satisfied for a test-isolated identity. Using a fresh principal per test keeps
// the contribution-window count independent of the shared "local" principal that
// every other test writes as.
func insertPrincipal(t *testing.T, st *pg.Store, id string) {
	t.Helper()
	_, err := st.Pool().Exec(t.Context(),
		`INSERT INTO principals (id, kind, display_name) VALUES ($1, 'agent', $1)
		 ON CONFLICT (id) DO NOTHING`, id)
	require.NoError(t, err)
}

// TestContributionQuotaDeniesLoudly is the section-8 security control under the
// off-the-turn write path (D-017): once a principal is at its contribution
// quota, the next automatic write is refused — and the refusal is loud, not a
// silent drop. The refused claim never lands, and a 'denied' row is recorded so
// the exhaustion is queryable (`cred usage`). A silent drop here is exactly how
// a poisoning attempt would hide (L8), which is why this is a security test
// first and a capacity test second.
func TestContributionQuotaDeniesLoudly(t *testing.T) {
	st := openStore(t)
	marker := fmt.Sprintf("quota-int-%d", time.Now().UnixNano())
	principal := "quota-p-" + marker
	insertPrincipal(t, st, principal)

	lineA := "the alpha " + marker + " convention is to fail closed"
	lineB := "the bravo " + marker + " convention is to log identifiers only"

	// Quota of one accepted claim; every other control disabled so this test
	// isolates the contribution quota. The fake nominator calls no model, so no
	// inference cost is recorded and the cost ceiling is irrelevant regardless.
	cfg := limit.Config{
		ContributionQuota:  1,
		ContributionWindow: time.Hour,
	}
	startWorker(t, st, &nominate.Fake{Kind: claim.KindConvention}, cfg)

	q, err := st.RiverInsertClient()
	require.NoError(t, err)
	enqueue := func(line string) {
		require.NoError(t, curate.EnqueueNominate(t.Context(), q, curate.NominateArgs{
			Source: line + "\n", SourceKind: "document",
			Repo: "quota-repo-" + marker, Path: "notes.md", BaseLine: 1,
			ScopeKind: "repository", ScopeValue: "quota-scope-" + marker,
			Principals: []string{principal}, Trigger: "turn",
		}))
	}

	// First contribution lands (accepted == 0 < quota 1).
	enqueue(lineA)
	waitForWrite(t, st, "alpha "+marker)

	// Second contribution is over quota (accepted == 1 >= quota 1): it must be
	// denied and recorded, and its claim must never land.
	enqueue(lineB)

	since := time.Now().Add(-time.Hour)
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		denied, err := st.DeniedInWindow(t.Context(), claim.PrincipalID(principal), since)
		require.NoError(t, err)
		if denied >= 1 {
			// The denial is on the record. Now assert the refused write did not
			// land — the claim was denied, not merely deduped later.
			entries, err := st.RecentWrites(t.Context(), 200)
			require.NoError(t, err)
			for _, e := range entries {
				require.NotContains(t, e.Statement, "bravo "+marker,
					"a denied contribution must never reach the store")
			}
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("the over-quota contribution was not denied-and-recorded within the deadline")
}

// TestScopeGrowthPrunes exercises the scope-growth bound directly against the
// store: a scope over its ceiling is pruned back, more aggressively the further
// over it is, and the pruned claims are expired (not deleted) with reason
// 'pruned'. Growth is bounded by policy, not by hope.
func TestScopeGrowthPrunes(t *testing.T) {
	st := openStore(t)
	emb := newEmbedder(t)
	marker := fmt.Sprintf("prune-int-%d", time.Now().UnixNano())
	scope := claim.Scope{Kind: claim.ScopeRepo, Value: "prune-scope-" + marker}

	// Write five claims into one scope through the real executor, so the
	// embeddings are real and normalized and the rows are indistinguishable from
	// a production write.
	exec := curate.NewExecutor(st, emb, testLogger())
	var source string
	var cands []nominate.Candidate
	for i := 0; i < 5; i++ {
		line := fmt.Sprintf("prune line %d %s holds a distinct fact", i, marker)
		source += line + "\n"
		cands = append(cands, nominate.Candidate{
			Kind:      claim.KindReference,
			Statement: fmt.Sprintf("prune claim %d %s", i, marker),
			Quote:     line,
			// Confidence descends so the prune order (lowest first) is defined and
			// the survivors are the highest-confidence claims.
			Confidence: 0.5 + float64(i)*0.05,
		})
	}
	res, err := exec.WriteCandidates(t.Context(), nominate.Input{
		Source: source, SourceKind: claim.SourceDocument,
		Repo: "prune-repo-" + marker, Path: "facts.md", BaseLine: 1,
		Scope: scope, Principals: []claim.PrincipalID{"local"},
	}, cands)
	require.NoError(t, err)
	require.Len(t, res.Written, 5, "all five distinct claims should have been written")

	live, err := st.ScopeClaimCount(t.Context(), scope)
	require.NoError(t, err)
	require.Equal(t, 5, live)

	// Ceiling 3, aggressiveness 0.5: over=2, headroom=ceil(1.0)=1, target=3.
	// Pruning three leaves two — below the ceiling, with headroom.
	pruner := curate.NewPruner(st, limit.Config{ScopeClaimCeiling: 3, PruneAggressiveness: 0.5}, testLogger())
	rep, err := pruner.Prune(t.Context(), scope)
	require.NoError(t, err)
	require.Equal(t, 5, rep.Live)
	require.Equal(t, 3, rep.Pruned, "prune must cut the overage plus headroom")

	after, err := st.ScopeClaimCount(t.Context(), scope)
	require.NoError(t, err)
	require.Equal(t, 2, after, "the scope must be pruned back below its ceiling")

	// The pruned claims are expired with reason 'pruned', not deleted: they still
	// appear in the write log, closed.
	entries, err := st.RecentWrites(t.Context(), 200)
	require.NoError(t, err)
	var pruned int
	for _, e := range entries {
		if strings.Contains(e.Statement, marker) && e.SupersededAt != nil &&
			e.SupersedeReason == pg.SupersedeReasonPruned {
			pruned++
		}
	}
	require.Equal(t, 3, pruned, "pruned claims must be expired with reason 'pruned', not deleted")

	// A second pass with the scope now under the ceiling prunes nothing.
	rep2, err := pruner.Prune(t.Context(), scope)
	require.NoError(t, err)
	require.Zero(t, rep2.Pruned, "a scope under its ceiling is left alone")
}
