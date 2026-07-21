//go:build integration

// Package pg's integration suite. Build-tagged so `go test ./...` stays under
// the per-commit budget and this runs only where a database is expected.
package pg_test

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/acl"
	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/embed"
	"github.com/canhta/cred/internal/embed/wordpiece"
	"github.com/canhta/cred/internal/recall"
	"github.com/canhta/cred/internal/seed"
	"github.com/canhta/cred/internal/store/pg"
)

// requireDB is set in CI. Skipping is a local-developer affordance; in CI it
// is a failure, because a skipped test and a passing test are the same green
// check and silent degradation is this project's worst enemy.
func requireDB() bool { return os.Getenv("CRED_REQUIRE_DB") == "1" }

// ran counts the tests that actually executed. A skip that is supposed to be
// impossible should be loud when it happens, so TestMain fails on zero under
// CRED_REQUIRE_DB.
var ran int

func TestMain(m *testing.M) {
	code := m.Run()
	if code == 0 && requireDB() && ran == 0 {
		fmt.Fprintln(os.Stderr,
			"CRED_REQUIRE_DB=1 but no integration test ran: the suite passed vacuously")
		code = 1
	}
	os.Exit(code)
}

func databaseURL() string {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v
	}
	return config.DefaultDatabaseURL
}

// openTestStore connects and migrates into a scratch schema, so a run never
// disturbs a developer's own seeded database.
func openTestStore(t *testing.T) *pg.Store {
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
	ran++
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

// writeFixtureRepo builds a tiny repository with documentation to seed from.
func writeFixtureRepo(t *testing.T, agents string) string {
	t.Helper()
	root := t.TempDir()
	require.NoError(t, os.WriteFile(root+"/AGENTS.md", []byte(agents), 0o600))
	require.NoError(t, os.MkdirAll(root+"/docs", 0o750))
	require.NoError(t, os.WriteFile(root+"/docs/storage.md", []byte(
		"# Storage\n\nCRED runs on a single PostgreSQL database with pgvector.\n"+
			"Relational, vector, full-text and queueing all live there.\n"), 0o600))
	return root
}

const agentsV1 = `# Conventions

The retrieval pipeline fuses a dense vector arm and a lexical arm with
reciprocal rank fusion at k equal to sixty.

# Access control

Access control is evaluated at recall, fails closed, and takes the
intersection of the evidence permissions rather than the union.
`

func newService(t *testing.T, st *pg.Store, e *embed.BGE) *recall.Service {
	t.Helper()
	tok, err := wordpiece.New()
	require.NoError(t, err)
	return recall.New(st, e, func(s string) int { return len(tok.Encode(s)) })
}

func seedRepo(t *testing.T, st *pg.Store, e *embed.BGE, root string) seed.Report {
	t.Helper()
	rep, err := seed.New(st, e, testLogger(t)).Run(t.Context(), root,
		[]claim.PrincipalID{"local"})
	require.NoError(t, err)
	return rep
}

// TestSeedThenRecall is the end-to-end vertical: documentation in, ranked
// claims out, with the evidence pointing back at a real file and line range.
func TestSeedThenRecall(t *testing.T) {
	st := openTestStore(t)
	e := newEmbedder(t)
	root := writeFixtureRepo(t, agentsV1)

	rep := seedRepo(t, st, e, root)
	require.Positive(t, rep.Inserted, "seeding wrote nothing")
	require.Zero(t, rep.Superseded, "a first seed supersedes nothing")

	res, err := newService(t, st, e).Recall(t.Context(), recall.Request{
		Query:     "how are the two retrieval arms combined",
		Principal: "local",
		Limit:     3,
		Now:       time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Claims, "seeded content was not retrievable")

	top := res.Claims[0]
	require.NotEmpty(t, top.Claim.Evidence, "L1: a claim reached recall with no evidence")
	require.Contains(t, strings.ToLower(top.Claim.Evidence[0].ExtractedText), "rank fusion")
	require.Positive(t, top.Claim.Evidence[0].LineStart)
	require.GreaterOrEqual(t, top.Claim.Evidence[0].LineEnd, top.Claim.Evidence[0].LineStart)
	require.Positive(t, top.Score)
	require.NotEmpty(t, top.Contributions, "the score must be explainable")
	require.False(t, res.AsOf.IsZero(), "every response carries as_of")
}

// TestReseedingIsIdempotent — the content hash is the change detector, so an
// unchanged corpus must produce no writes at all.
func TestReseedingIsIdempotent(t *testing.T) {
	st := openTestStore(t)
	e := newEmbedder(t)
	root := writeFixtureRepo(t, agentsV1)

	first := seedRepo(t, st, e, root)
	require.Positive(t, first.Inserted)

	second := seedRepo(t, st, e, root)
	require.Zero(t, second.Inserted, "re-seeding unchanged content wrote rows")
	require.Zero(t, second.Superseded)
	require.Equal(t, first.Chunks, second.Unchanged,
		"every chunk should have been recognised as unchanged")
}

// TestChangedFileSupersedesRatherThanDuplicates — nothing is deleted; things
// expire. The old row stays, closed in transaction time, linked to its
// successor.
func TestChangedFileSupersedesRatherThanDuplicates(t *testing.T) {
	st := openTestStore(t)
	e := newEmbedder(t)
	root := writeFixtureRepo(t, agentsV1)

	seeded := seedRepo(t, st, e, root)
	// Scope the count to this test's repo, not st.Counts()'s global total: the
	// integration packages share one database and CI runs them concurrently
	// (go test -tags=integration ./... with default -p), so a global delta would
	// pick up another package's writes mid-test.
	beforeClaims := liveClaimsForRepo(t, st, seeded.Repo)

	changed := strings.Replace(agentsV1,
		"reciprocal rank fusion at k equal to sixty",
		"reciprocal rank fusion at k equal to sixty, one-based", 1)
	require.NotEqual(t, agentsV1, changed)
	require.NoError(t, os.WriteFile(root+"/AGENTS.md", []byte(changed), 0o600))

	rep := seedRepo(t, st, e, root)
	require.Positive(t, rep.Superseded, "a changed chunk did not supersede its predecessor")

	afterClaims := liveClaimsForRepo(t, st, seeded.Repo)
	require.Equal(t, beforeClaims, afterClaims,
		"live claim count must not grow when a chunk is replaced")
}

// liveClaimsForRepo counts live claims whose evidence belongs to repo. It reads
// the pool directly — a test asserting store state, scoped so a concurrent
// package sharing the database cannot perturb the count.
func liveClaimsForRepo(t *testing.T, st *pg.Store, repo string) int {
	t.Helper()
	var n int
	err := st.Pool().QueryRow(t.Context(), `
		SELECT count(*) FROM claims c
		 WHERE c.superseded_at IS NULL
		   AND EXISTS (SELECT 1 FROM claim_evidence ce
		                 JOIN evidence e ON e.id = ce.evidence_id
		                WHERE ce.claim_id = c.id AND e.source_repo = $1)`, repo).Scan(&n)
	require.NoError(t, err)
	return n
}

// TestBothArmsContribute — if one arm never appears, the system is doing
// single-arm search with the latency of two, and RRF is decoration.
func TestBothArmsContribute(t *testing.T) {
	st := openTestStore(t)
	e := newEmbedder(t)
	seedRepo(t, st, e, writeFixtureRepo(t, agentsV1))

	res, err := newService(t, st, e).Recall(t.Context(), recall.Request{
		Query:     "PostgreSQL pgvector database",
		Principal: "local",
		Limit:     5,
		Now:       time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, res.Claims)

	seen := map[recall.Arm]bool{}
	for _, c := range res.Claims {
		for _, contrib := range c.Contributions {
			seen[contrib.Arm] = true
		}
	}
	require.True(t, seen[recall.ArmDense], "the dense arm contributed nothing")
	require.True(t, seen[recall.ArmLexical], "the lexical arm contributed nothing")
}

// TestUnauthorizedPrincipalSeesNothing — L5 fails closed. A principal with no
// grant gets an empty result, not an error: an error is an existence oracle.
func TestUnauthorizedPrincipalSeesNothing(t *testing.T) {
	st := openTestStore(t)
	e := newEmbedder(t)
	seedRepo(t, st, e, writeFixtureRepo(t, agentsV1))

	svc := newService(t, st, e)
	const query = "how are the two retrieval arms combined"

	authorized, err := svc.Recall(t.Context(), recall.Request{
		Query: query, Principal: "local", Limit: 5, Now: time.Now().UTC(),
	})
	require.NoError(t, err)
	require.NotEmpty(t, authorized.Claims)

	stranger, err := svc.Recall(t.Context(), recall.Request{
		Query: query, Principal: "nobody", Limit: 5, Now: time.Now().UTC(),
	})
	require.NoError(t, err, "denial must not surface as an error")
	require.Empty(t, stranger.Claims)
	require.Zero(t, stranger.Authorized)
	// The candidate count is identical, which is the point: the filtering
	// happens in Go over rows Postgres returned either way.
	require.Equal(t, authorized.Candidates, stranger.Candidates)
}

// TestStoreReturnsRowsNotDecisions — L5 is computed in Go over what the store
// handed back. This asserts the store never applied it: the ACL rows come back
// intact and internal/acl reaches the same verdict from them alone.
func TestStoreReturnsRowsNotDecisions(t *testing.T) {
	st := openTestStore(t)
	e := newEmbedder(t)
	seedRepo(t, st, e, writeFixtureRepo(t, agentsV1))

	vec, err := e.Embed(t.Context(), []string{"reciprocal rank fusion"})
	require.NoError(t, err)

	modelID, _, _, err := st.PresentModel(t.Context())
	require.NoError(t, err)

	hits, err := st.DenseSearch(t.Context(), modelID, vec[0], 5)
	require.NoError(t, err)
	require.NotEmpty(t, hits)
	require.Equal(t, 1, hits[0].Rank, "ranks are 1-based")

	ids := []string{hits[0].ClaimID}
	claims, err := st.LoadClaims(t.Context(), ids)
	require.NoError(t, err)
	require.Len(t, claims, 1)

	evidence, err := st.LoadEvidence(t.Context(), ids)
	require.NoError(t, err)
	claimACLs, err := st.LoadClaimACLs(t.Context(), ids)
	require.NoError(t, err)
	require.NotEmpty(t, claimACLs[ids[0]], "the store dropped the grants")

	c := claims[0]
	c.ACL = claimACLs[c.ID]
	c.Evidence = evidence[c.ID]
	require.NotEmpty(t, c.Evidence)

	evidenceIDs := []string{c.Evidence[0].ID}
	evidenceACLs, err := st.LoadEvidenceACLs(t.Context(), evidenceIDs)
	require.NoError(t, err)
	c.Evidence[0].ACL = evidenceACLs[c.Evidence[0].ID]

	now := time.Now().UTC()
	require.True(t, acl.CanRead(c, "local", now))
	require.False(t, acl.CanRead(c, "nobody", now))
}

// TestSchemaRejectsAnOpenSupersessionEdge — the constraint that keeps a claim
// from being current and replaced at the same time.
func TestSchemaRejectsAnOpenSupersessionEdge(t *testing.T) {
	st := openTestStore(t)
	_, err := st.Pool().Exec(t.Context(), `
		INSERT INTO claims (kind, statement, scope_kind, scope_value,
		                    valid_from, recorded_at, confidence,
		                    source_repo, extracted_by_model, superseded_by)
		VALUES ('Reference','x','path','x', now(), now(), 0.5, 'r', '',
		        '00000000-0000-0000-0000-000000000000')`)
	require.Error(t, err, "a supersession edge without a closed interval was accepted")
}

// TestSchemaRejectsAnUnnormalizedVector — the norm CHECK catches provider
// drift at write time rather than as an unexplained decline in recall.
func TestSchemaRejectsAnUnnormalizedVector(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()

	var claimID string
	err := st.Pool().QueryRow(ctx, `
		INSERT INTO claims (kind, statement, scope_kind, scope_value,
		                    valid_from, recorded_at, confidence,
		                    source_repo, extracted_by_model)
		VALUES ('Reference','norm probe','path','x', now(), now(), 0.5, 'probe', '')
		RETURNING id`).Scan(&claimID)
	require.NoError(t, err)

	var b strings.Builder
	b.WriteByte('[')
	for i := range embed.Dimensions {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString("1") // norm is sqrt(384), far outside tolerance
	}
	b.WriteByte(']')

	_, err = st.Pool().Exec(ctx, `
		INSERT INTO claim_embeddings (embedding_model_id, claim_id, embedding)
		VALUES (1, $1, $2::halfvec)`, claimID, b.String())
	require.Error(t, err, "an unnormalized vector was accepted")
	require.Contains(t, err.Error(), "is_normalized")
}

// testLogger discards output. The seeder logs identifiers and counts only —
// never chunk text, because ingested content is untrusted and a test log is
// read by the same tools as a production one.
func testLogger(t *testing.T) *slog.Logger {
	t.Helper()
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
