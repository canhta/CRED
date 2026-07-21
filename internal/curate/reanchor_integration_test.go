//go:build integration

package curate_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/curate"
	"github.com/canhta/cred/internal/seed"
	"github.com/canhta/cred/internal/store/pg"
)

// claimStatus returns (supersededAt, reason) for the live-or-superseded claim
// whose statement contains marker. It reads the pool directly — this is a test
// asserting store state, not production code deciding anything.
func claimStatus(t *testing.T, st *pg.Store, marker string) (superseded bool, reason string) {
	t.Helper()
	var at *time.Time
	var r *string
	err := st.Pool().QueryRow(t.Context(), `
		SELECT superseded_at, supersede_reason
		  FROM claims
		 WHERE statement LIKE '%' || $1 || '%'
		 ORDER BY recorded_at DESC
		 LIMIT 1`, marker).Scan(&at, &r)
	require.NoError(t, err, "no claim found for marker %q", marker)
	if r != nil {
		reason = *r
	}
	return at != nil, reason
}

// TestReanchorExpiresSemanticChangeNotFormatting is the L3 vertical: seed a doc,
// change one section's formatting only and another section's words, re-anchor,
// and verify exactly the semantically-changed claim expired while the
// formatting-only one survived. This is PRD acceptance criterion 4 end to end.
func TestReanchorExpiresSemanticChangeNotFormatting(t *testing.T) {
	st := openStore(t)
	emb := newEmbedder(t)

	marker := fmt.Sprintf("anc%d", time.Now().UnixNano())
	alpha := "alpha-" + marker
	beta := "beta-" + marker

	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	require.NoError(t, os.MkdirAll(docs, 0o755))
	docPath := filepath.Join(docs, "anchortest.md")

	// Two sections. Alpha's body will be reformatted only; Beta's words change.
	original := fmt.Sprintf(`# Anchor test %s

## Section %s

The alpha section states that the cache is warmed on startup for fast reads.

## Section %s

The beta section states that retries use a fixed backoff of one second.
`, marker, alpha, beta)
	require.NoError(t, os.WriteFile(docPath, []byte(original), 0o644))

	// Seed.
	rep, err := seed.New(st, emb, testLogger()).Run(t.Context(), root, []claim.PrincipalID{"local"})
	require.NoError(t, err)
	require.Positive(t, rep.Inserted, "seeding wrote at least the two section chunks")

	// Both sections start live and unsuperseded.
	for _, m := range []string{alpha, beta} {
		superseded, _ := claimStatus(t, st, m)
		require.False(t, superseded, "section %s must be live after seeding", m)
	}

	// Mutate on disk WITHOUT re-seeding: Alpha reflowed (same words, new bytes),
	// Beta's words genuinely changed.
	mutated := fmt.Sprintf(`# Anchor test %s

## Section %s

The alpha section states that the cache is warmed
on startup

for   fast reads.

## Section %s

The beta section states that retries use exponential backoff capped at thirty seconds.
`, marker, alpha, beta)
	require.NoError(t, os.WriteFile(docPath, []byte(mutated), 0o644))

	// Re-anchor.
	rrep, err := curate.NewReanchorer(st, testLogger()).Reanchor(t.Context(), root)
	require.NoError(t, err)
	require.Positive(t, rrep.Checked, "reanchor resolved the seeded anchors")

	// Alpha: formatting churn — tiers 1 and 2 held — must NOT expire.
	alphaSuperseded, _ := claimStatus(t, st, alpha)
	require.False(t, alphaSuperseded,
		"a pure-formatting change must expire zero claims (L3, acceptance criterion 4)")

	// Beta: a real edit — tier 2 changed — must expire with the stale-anchor reason.
	betaSuperseded, betaReason := claimStatus(t, st, beta)
	require.True(t, betaSuperseded, "a semantic change must expire the claim")
	require.Equal(t, pg.SupersedeReasonStaleAnchor, betaReason,
		"the expiry reason distinguishes an L3 invalidation from a dedup or a forget")

	require.Equal(t, 1, rrep.Expired, "exactly the beta claim expired")
	require.Positive(t, rrep.Valid, "the alpha claim (and the title chunk) survived")
}

// TestReanchorLeavesUnchangedFilesAlone verifies the no-op case: re-anchoring a
// file that has not changed expires nothing. A re-anchor that expired claims on
// an untouched file would be worse than useless.
func TestReanchorLeavesUnchangedFilesAlone(t *testing.T) {
	st := openStore(t)
	emb := newEmbedder(t)

	marker := fmt.Sprintf("noop%d", time.Now().UnixNano())
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	require.NoError(t, os.MkdirAll(docs, 0o755))
	doc := fmt.Sprintf("# Stable %s\n\n## Kept %s\n\nThis section is never touched after seeding.\n", marker, marker)
	require.NoError(t, os.WriteFile(filepath.Join(docs, "stable.md"), []byte(doc), 0o644))

	_, err := seed.New(st, emb, testLogger()).Run(t.Context(), root, []claim.PrincipalID{"local"})
	require.NoError(t, err)

	rrep, err := curate.NewReanchorer(st, testLogger()).Reanchor(t.Context(), root)
	require.NoError(t, err)
	require.Zero(t, rrep.Expired, "an unchanged file must expire nothing")

	superseded, _ := claimStatus(t, st, marker)
	require.False(t, superseded)
}
