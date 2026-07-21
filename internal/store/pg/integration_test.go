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
	"github.com/canhta/cred/internal/limit"
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
	require.NotEmpty(t, top.Claim.Evidence, "a claim reached recall with no evidence")
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

// TestUnauthorizedPrincipalSeesNothing — access control fails closed. A principal with no
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

// TestStoreReturnsRowsNotDecisions — the ACL intersection is computed in Go over what the store
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

// TestRecallBudgetDeniesAndRecords is the read-path half of usage limits: an agent
// calling recall in a loop is rate-limited per principal, protecting tail
// latency, and the denial is loud — a synchronous, typed BudgetError, since
// recall is on the turn (unlike the write path's recorded 'denied' event). Every
// admitted recall is recorded for cost attribution, so the counter the limit
// decides over is the same one the recall wrote.
func TestRecallBudgetDeniesAndRecords(t *testing.T) {
	st := openTestStore(t)
	e := newEmbedder(t)
	tok, err := wordpiece.New()
	require.NoError(t, err)
	count := func(s string) int { return len(tok.Encode(s)) }

	// A fresh principal so the recall window is this test's alone. Recall's
	// counter lives in usage_events, which has no principals FK, so the principal
	// need not be a grantable identity here.
	principal := claim.PrincipalID(fmt.Sprintf("recall-budget-%d", time.Now().UnixNano()))
	cfg := limit.Config{
		RecallRate:       2,
		RecallWindow:     time.Hour,
		MaxPackageClaims: 5,
	}
	svc := recall.New(st, e, count).WithLimits(cfg, st)

	now := time.Now().UTC()
	req := recall.Request{Query: "anything at all", Principal: principal, Limit: 3, Now: now}

	// Two recalls are admitted (rate 2). They may return nothing — the store need
	// not hold matching claims for the budget to apply.
	for i := 0; i < 2; i++ {
		_, admitErr := svc.Recall(t.Context(), req)
		require.NoError(t, admitErr, "recall %d was within budget and must be admitted", i)
	}

	// Both were recorded for cost attribution.
	recalls, err := st.RecallsInWindow(t.Context(), principal, limit.WindowStart(now, cfg.RecallWindow))
	require.NoError(t, err)
	require.Equal(t, 2, recalls, "each admitted recall must be recorded for cost attribution")

	// The third is over the rate: a loud, typed denial, not a silent empty result.
	_, denyErr := svc.Recall(t.Context(), req)
	require.Error(t, denyErr, "the over-budget recall must be denied, not silently served empty")
	be, ok := recall.AsBudgetError(denyErr)
	require.True(t, ok, "a budget denial must be a typed BudgetError the caller can surface")
	require.Equal(t, limit.ReasonRecallRate, be.Reason)
}

// uniqueEmail returns an email address that has never been used by another
// test in this run. Tests in this package share one database and CI runs
// them concurrently, so a fixed literal would collide across tests via the
// email UNIQUE constraint.
func uniqueEmail(t *testing.T, tag string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d@example.test", tag, time.Now().UnixNano())
}

// uniqueTokenHash returns a value that has never been used as a
// sessions.token_hash by another test run against this database -- the
// column is UNIQUE, and this suite's database persists across runs (a
// developer's own manual testing included), so a fixed literal collides.
func uniqueTokenHash(t *testing.T, tag string) string {
	t.Helper()
	return fmt.Sprintf("%s-%d", tag, time.Now().UnixNano())
}

// TestCreateUserSessionPrincipalRoundTrip covers the path register() and
// login() both depend on: a created user's principal can start a session,
// and that session resolves back to the same principal and role.
func TestCreateUserSessionPrincipalRoundTrip(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	inviterID := insertThrowawayPrincipal(t, st)

	principal, err := st.CreateInvitedUser(ctx, uniqueEmail(t, "roundtrip"), "hash", "member", claim.PrincipalID(inviterID))
	require.NoError(t, err)
	require.NotEmpty(t, principal)

	tokenHash := uniqueTokenHash(t, "roundtrip")
	require.NoError(t, st.CreateSession(ctx, principal, tokenHash, time.Now().UTC().Add(time.Hour)))

	gotPrincipal, gotRole, err := st.SessionPrincipal(ctx, tokenHash, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, principal, gotPrincipal)
	require.Equal(t, "member", gotRole)
}

// TestSessionPrincipalRejectsExpiredSession -- an expired session must be
// indistinguishable from an absent one, the same as an authorization failure
// and a nonexistent row are indistinguishable elsewhere in this codebase.
func TestSessionPrincipalRejectsExpiredSession(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	inviterID := insertThrowawayPrincipal(t, st)

	principal, err := st.CreateInvitedUser(ctx, uniqueEmail(t, "expired"), "hash", "member", claim.PrincipalID(inviterID))
	require.NoError(t, err)

	tokenHash := uniqueTokenHash(t, "expired")
	past := time.Now().UTC().Add(-time.Hour)
	require.NoError(t, st.CreateSession(ctx, principal, tokenHash, past))

	_, _, err = st.SessionPrincipal(ctx, tokenHash, time.Now().UTC())
	require.ErrorIs(t, err, pg.ErrNotFound)
}

// TestFailedLoginsInWindowCountsOnlyFailuresInWindow -- the login rate limit
// counts failures within a window; a success must never count against it, and
// a failure outside the window must not either. now is passed explicitly to
// RecordLoginAttempt so the window boundary is exact rather than approximated
// with a sleep.
func TestFailedLoginsInWindowCountsOnlyFailuresInWindow(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	email := uniqueEmail(t, "loginwindow")

	windowStart := time.Now().UTC()
	beforeWindow := windowStart.Add(-time.Hour)
	insideWindow1 := windowStart.Add(time.Minute)
	insideWindow2 := windowStart.Add(2 * time.Minute)

	require.NoError(t, st.RecordLoginAttempt(ctx, email, false, beforeWindow)) // too old: excluded
	require.NoError(t, st.RecordLoginAttempt(ctx, email, true, insideWindow1)) // succeeded: excluded
	require.NoError(t, st.RecordLoginAttempt(ctx, email, false, insideWindow1))
	require.NoError(t, st.RecordLoginAttempt(ctx, email, false, insideWindow2))

	failed, err := st.FailedLoginsInWindow(ctx, email, windowStart)
	require.NoError(t, err)
	require.Equal(t, 2, failed, "only the two in-window failures should count")
}

// TestCreateUserDuplicateEmailReturnsErrEmailTaken -- the same-email race is
// caught by user_credentials_email_key.
func TestCreateUserDuplicateEmailReturnsErrEmailTaken(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	email := uniqueEmail(t, "dup")
	inviterID := insertThrowawayPrincipal(t, st)

	_, err := st.CreateInvitedUser(ctx, email, "hash", "member", claim.PrincipalID(inviterID))
	require.NoError(t, err)

	_, err = st.CreateInvitedUser(ctx, email, "hash", "member", claim.PrincipalID(inviterID))
	require.ErrorIs(t, err, pg.ErrEmailTaken)
}

// insertThrowawayPrincipal inserts a bare principals row (kind='agent') for
// tests that need a valid invited_by/principal_id foreign-key target
// without depending on shared bootstrap-admin state.
func insertThrowawayPrincipal(t *testing.T, st *pg.Store) string {
	t.Helper()
	var id string
	err := st.Pool().QueryRow(t.Context(), `
		INSERT INTO principals (id, kind, display_name)
		VALUES (gen_random_uuid()::text, 'agent', $1)
		RETURNING id`, uniqueEmail(t, "throwaway")).Scan(&id)
	require.NoError(t, err)
	return id
}

// TestCreateUserSecondBootstrapReturnsErrBootstrapExists proves the
// replacement for the registration TOCTOU race that
// user_credentials_one_admin used to close: two concurrent bootstrap
// registrations (CreateUser, invited_by always NULL) must never both
// succeed, now that admins are no longer capped and something else has to
// close this race.
//
// This suite shares one database across runs (including a developer's own
// manual testing of the register endpoint), so a bootstrap row may already
// exist here -- the test only creates a first one when none is present,
// then asserts the constraint rejects an additional one either way.
func TestCreateUserSecondBootstrapReturnsErrBootstrapExists(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()

	var alreadyHasBootstrap bool
	err := st.Pool().QueryRow(ctx,
		`SELECT EXISTS (SELECT 1 FROM user_credentials WHERE invited_by IS NULL)`,
	).Scan(&alreadyHasBootstrap)
	require.NoError(t, err)

	if !alreadyHasBootstrap {
		_, createErr := st.CreateUser(ctx, uniqueEmail(t, "bootstrap1"), "hash", "admin")
		require.NoError(t, createErr)
	}

	_, err = st.CreateUser(ctx, uniqueEmail(t, "bootstrap2"), "hash", "admin")
	require.ErrorIs(t, err, pg.ErrBootstrapExists)
}

// TestCreateInvitedUserAllowsSecondAdmin proves the role model this sub-
// project adds: an admin may invite another admin, which the dropped
// user_credentials_one_admin index used to forbid outright. Both accounts
// are created via CreateInvitedUser (invited_by set), so this is
// independent of whatever bootstrap state the shared database already has.
func TestCreateInvitedUserAllowsSecondAdmin(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	inviterID := insertThrowawayPrincipal(t, st)

	_, err := st.CreateInvitedUser(ctx, uniqueEmail(t, "invited-admin-1"), "hash", "admin", claim.PrincipalID(inviterID))
	require.NoError(t, err)

	_, err = st.CreateInvitedUser(ctx, uniqueEmail(t, "invited-admin-2"), "hash", "admin", claim.PrincipalID(inviterID))
	require.NoError(t, err, "an invited account must be able to hold the admin role even when one already exists")
}

// TestAdminFloorTriggerRejectsRemovingTheLastAdmin proves the floor
// invariant: user_credentials_admin_floor_trigger has no application caller
// yet (nothing demotes or removes an account), but the constraint itself is
// exercised directly here so it's proven correct before anything depends on
// it. Runs inside a transaction that always rolls back, so it never
// disturbs the other admin rows this shared test database already has.
func TestAdminFloorTriggerRejectsRemovingTheLastAdmin(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	email := uniqueEmail(t, "floortrigger")
	inviterID := insertThrowawayPrincipal(t, st)

	tx, err := st.Pool().Begin(ctx)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback(ctx) }()

	var principalID string
	err = tx.QueryRow(ctx, `
		INSERT INTO principals (id, kind, display_name)
		VALUES (gen_random_uuid()::text, 'user', $1)
		RETURNING id`, email).Scan(&principalID)
	require.NoError(t, err)

	// invited_by is set (to a throwaway principal, not NULL) so this insert
	// doesn't itself collide with user_credentials_one_bootstrap when a
	// bootstrap row already exists elsewhere in this shared database -- this
	// test only cares about the floor trigger, not bootstrap semantics.
	_, err = tx.Exec(ctx, `
		INSERT INTO user_credentials (principal_id, email, password_hash, role, invited_by)
		VALUES ($1, $2, 'hash', 'admin', $3)`, principalID, email, inviterID)
	require.NoError(t, err)

	// Demote every other existing admin, so the trigger's count-of-admins
	// check has exactly one row (the one just inserted above) to reason
	// about -- visible only inside this transaction, undone by the deferred
	// rollback. Doing this after the insert, not before, matters: demoting
	// every admin before this test's own row exists would drive the count to
	// zero itself when more than one admin already exists in this shared
	// database, tripping the trigger before the test's own setup finishes.
	_, err = tx.Exec(ctx,
		`UPDATE user_credentials SET role = 'member' WHERE role = 'admin' AND principal_id != $1`, principalID)
	require.NoError(t, err)

	_, err = tx.Exec(ctx,
		`UPDATE user_credentials SET role = 'member' WHERE principal_id = $1`, principalID)
	require.Error(t, err, "demoting the only remaining admin must be rejected")
}

// TestClaimInviteRoundTrip covers the path register()'s invite branch
// depends on: a created invite can be claimed once with the right email,
// and a second claim of the same token fails.
func TestClaimInviteRoundTrip(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	email := uniqueEmail(t, "claim")
	tokenHash := uniqueTokenHash(t, "claim")
	inviterID := insertThrowawayPrincipal(t, st)

	_, err := st.CreateInvite(ctx, email, "member", tokenHash, claim.PrincipalID(inviterID),
		time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	role, invitedBy, err := st.ClaimInvite(ctx, tokenHash, email, time.Now().UTC())
	require.NoError(t, err)
	require.Equal(t, "member", role)
	require.Equal(t, claim.PrincipalID(inviterID), invitedBy)

	_, _, err = st.ClaimInvite(ctx, tokenHash, email, time.Now().UTC())
	require.ErrorIs(t, err, pg.ErrNotFound, "an already-used invite must not be claimable again")
}

// TestClaimInviteRejectsEmailMismatch proves a mismatched email neither
// succeeds nor burns the invite -- the real invitee can still claim it
// afterward with the correct email.
func TestClaimInviteRejectsEmailMismatch(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	email := uniqueEmail(t, "mismatch")
	tokenHash := uniqueTokenHash(t, "mismatch")
	inviterID := insertThrowawayPrincipal(t, st)

	_, err := st.CreateInvite(ctx, email, "member", tokenHash, claim.PrincipalID(inviterID),
		time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)

	_, _, err = st.ClaimInvite(ctx, tokenHash, "someone-else@example.test", time.Now().UTC())
	require.ErrorIs(t, err, pg.ErrNotFound)

	role, _, err := st.ClaimInvite(ctx, tokenHash, email, time.Now().UTC())
	require.NoError(t, err, "the mismatched attempt must not have burned the invite")
	require.Equal(t, "member", role)
}

// TestClaimInviteRejectsExpiredInvite and TestRevokeInviteThenClaimFails
// cover the other two collapsed failure modes register() treats
// identically to a not-found token.
func TestClaimInviteRejectsExpiredInvite(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	email := uniqueEmail(t, "expired")
	tokenHash := uniqueTokenHash(t, "expired")
	inviterID := insertThrowawayPrincipal(t, st)

	_, err := st.CreateInvite(ctx, email, "member", tokenHash, claim.PrincipalID(inviterID),
		time.Now().UTC().Add(-time.Hour))
	require.NoError(t, err)

	_, _, err = st.ClaimInvite(ctx, tokenHash, email, time.Now().UTC())
	require.ErrorIs(t, err, pg.ErrNotFound)
}

func TestRevokeInviteThenClaimFails(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	email := uniqueEmail(t, "revoked")
	tokenHash := uniqueTokenHash(t, "revoked")
	inviterID := insertThrowawayPrincipal(t, st)

	id, err := st.CreateInvite(ctx, email, "member", tokenHash, claim.PrincipalID(inviterID),
		time.Now().UTC().Add(time.Hour))
	require.NoError(t, err)
	require.NoError(t, st.RevokeInvite(ctx, id, time.Now().UTC()))

	_, _, err = st.ClaimInvite(ctx, tokenHash, email, time.Now().UTC())
	require.ErrorIs(t, err, pg.ErrNotFound)
}

// TestPendingInvitesExcludesUsedExpiredAndRevoked proves the Team page's
// pending-invites list never shows an invite that can no longer be
// redeemed.
func TestPendingInvitesExcludesUsedExpiredAndRevoked(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	inviterID := insertThrowawayPrincipal(t, st)
	now := time.Now().UTC()

	pendingEmail := uniqueEmail(t, "pending")
	_, err := st.CreateInvite(ctx, pendingEmail, "member", uniqueTokenHash(t, "pending"),
		claim.PrincipalID(inviterID), now.Add(time.Hour))
	require.NoError(t, err)

	usedEmail := uniqueEmail(t, "used")
	usedTokenHash := uniqueTokenHash(t, "used")
	_, err = st.CreateInvite(ctx, usedEmail, "member", usedTokenHash, claim.PrincipalID(inviterID), now.Add(time.Hour))
	require.NoError(t, err)
	_, _, err = st.ClaimInvite(ctx, usedTokenHash, usedEmail, now)
	require.NoError(t, err)

	expiredEmail := uniqueEmail(t, "expired-list")
	_, err = st.CreateInvite(ctx, expiredEmail, "member", uniqueTokenHash(t, "expired-list"),
		claim.PrincipalID(inviterID), now.Add(-time.Hour))
	require.NoError(t, err)

	revokedEmail := uniqueEmail(t, "revoked-list")
	revokedID, err := st.CreateInvite(ctx, revokedEmail, "member", uniqueTokenHash(t, "revoked-list"),
		claim.PrincipalID(inviterID), now.Add(time.Hour))
	require.NoError(t, err)
	require.NoError(t, st.RevokeInvite(ctx, revokedID, now))

	pending, err := st.PendingInvites(ctx, now)
	require.NoError(t, err)

	var emails []string
	for _, inv := range pending {
		emails = append(emails, inv.Email)
	}
	require.Contains(t, emails, pendingEmail)
	require.NotContains(t, emails, usedEmail)
	require.NotContains(t, emails, expiredEmail)
	require.NotContains(t, emails, revokedEmail)
}

// TestTeamMembersListsEveryAccount covers the Team page's members table.
func TestTeamMembersListsEveryAccount(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	email := uniqueEmail(t, "member-list")
	inviterID := insertThrowawayPrincipal(t, st)

	principal, err := st.CreateInvitedUser(ctx, email, "hash", "member", claim.PrincipalID(inviterID))
	require.NoError(t, err)

	members, err := st.TeamMembers(ctx)
	require.NoError(t, err)

	var found bool
	for _, m := range members {
		if m.PrincipalID == string(principal) {
			found = true
			require.Equal(t, email, m.Email)
			require.Equal(t, "member", m.Role)
		}
	}
	require.True(t, found, "the created member must appear in TeamMembers")
}

// TestRoleForPrincipalResolvesRealRoleOrEmpty covers both callers this
// backs: authenticate()'s session path already gets role from
// SessionPrincipal, but the header/default-principal path calls this
// directly. A principal with a user_credentials row resolves its real role;
// a principal with none (a team/org/agent principal, or a header/default
// value with no console account) resolves "" rather than an error -- having
// no console account is the normal case for most principals, not a failure.
func TestRoleForPrincipalResolvesRealRoleOrEmpty(t *testing.T) {
	st := openTestStore(t)
	ctx := t.Context()
	inviterID := insertThrowawayPrincipal(t, st)

	principal, err := st.CreateInvitedUser(ctx, uniqueEmail(t, "roleforprincipal"), "hash", "member", claim.PrincipalID(inviterID))
	require.NoError(t, err)

	role, err := st.RoleForPrincipal(ctx, principal)
	require.NoError(t, err)
	require.Equal(t, "member", role)

	role, err = st.RoleForPrincipal(ctx, claim.PrincipalID("no-such-principal"))
	require.NoError(t, err)
	require.Empty(t, role)
}
