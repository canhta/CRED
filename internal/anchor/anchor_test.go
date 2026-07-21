package anchor_test

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/anchor"
	"github.com/canhta/cred/internal/claim"
)

// These tests take no database and no connection. internal/anchor is pure, like
// internal/temporal and internal/acl; if a test here needs either, the boundary
// this package exists to hold has already been broken. They are the L3 law:
// formatting churn survives, a real edit expires, an insertion above does not
// silently re-anchor, and ambiguity expires rather than guesses.

// baseDoc is a small decision-log-shaped document. The claim under test is
// anchored to the "D-010 > Reasoning" section.
const baseDoc = `# Decisions

## D-010

The reranker decision.

### Reasoning

The cross-encoder is cut because it is too slow on CPU at every model size.

### Cost

MaxSim costs storage.

## D-011

Sovereignty is a tiebreaker, not a wedge.
`

func rawHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// anchorFor computes the anchor for the line containing marker in doc, as the
// ingest path would.
func anchorFor(t *testing.T, doc, marker string) anchor.Anchor {
	t.Helper()
	line := 0
	for i, l := range strings.Split(doc, "\n") {
		if strings.Contains(l, marker) {
			line = i + 1
			break
		}
	}
	require.Positive(t, line, "marker %q not found", marker)
	span := anchor.Span{LineStart: line, LineEnd: line, ByteHash: rawHash(marker)}
	return anchor.TextAnchorer{}.Compute(anchor.Source{Text: doc, Kind: claim.SourceDocument}, span)
}

func resolve(doc string, a anchor.Anchor) anchor.Verdict {
	return anchor.TextAnchorer{}.Resolve(a, anchor.Source{Text: doc, Kind: claim.SourceDocument})
}

func TestComputeProducesHeadingPath(t *testing.T) {
	a := anchorFor(t, baseDoc, "cross-encoder is cut")
	require.Equal(t, "Decisions > D-010 > Reasoning", a.SymbolPath, "tier 1 is the full heading path from the top")
	require.NotEmpty(t, a.NodeHash, "tier 2 hashes the enclosing section")
	require.NotEmpty(t, a.WindowHash, "tier 3 hashes a window")
	require.True(t, a.Anchored())
}

func TestFormattingChurnDoesNotExpire(t *testing.T) {
	// Tier 4 changes (every byte of the section is reflowed and re-spaced) while
	// tiers 1 and 2 hold. A pure-formatting commit must expire zero claims —
	// PRD acceptance criterion 4, and the whole reason the ladder exists.
	a := anchorFor(t, baseDoc, "cross-encoder is cut")

	reformatted := strings.NewReplacer(
		// reflow the Reasoning body across lines and squeeze/expand whitespace;
		// the words are identical, only the byte layout changes.
		"The cross-encoder is cut because it is too slow on CPU at every model size.",
		"The   cross-encoder is cut\nbecause it is too slow on CPU\nat every model size.",
		"## D-010", "##    D-010",
	).Replace(baseDoc)
	require.NotEqual(t, baseDoc, reformatted, "the fixture must actually reformat")

	v := resolve(reformatted, a)
	require.Equal(t, anchor.Valid, v.Kind, "formatting churn holds tiers 1 and 2: %s", v.Reason)
	require.False(t, v.Kind.Expires())
}

func TestSemanticEditExpires(t *testing.T) {
	// The words in the anchored section change. Tier 1 still resolves (same
	// heading) but tier 2 differs — a genuine semantic change, which expires.
	a := anchorFor(t, baseDoc, "cross-encoder is cut")

	edited := strings.Replace(baseDoc,
		"The cross-encoder is cut because it is too slow on CPU at every model size.",
		"The cross-encoder is KEPT because a GPU is now assumed.", 1)
	require.NotEqual(t, baseDoc, edited)

	v := resolve(edited, a)
	require.Equal(t, anchor.SemanticChange, v.Kind, v.Reason)
	require.True(t, v.Kind.Expires())
}

func TestInsertionAboveDoesNotSilentlyReAnchor(t *testing.T) {
	// The failure L3 exists to prevent. Insert a whole new section above the
	// anchored one. Every line of "D-010 > Reasoning" now sits at a different
	// line number, so a tier-4 line-range hash would be looking at other text.
	// The ladder resolves by heading path instead: the claim stays anchored to
	// the SAME section it was about, and stays valid.
	a := anchorFor(t, baseDoc, "cross-encoder is cut")

	inserted := strings.Replace(baseDoc, "## D-010",
		"## D-005\n\nCompete head-on with the incumbents.\n\n### Reasoning\n\nThe market is validated by funded competitors.\n\n## D-010", 1)
	require.Greater(t, strings.Count(inserted, "\n"), strings.Count(baseDoc, "\n"),
		"the fixture must actually shift lines down")

	// Prove the resolved node is the real D-010 > Reasoning, not whatever now sits
	// at the original line numbers: the node hash is unchanged from ingest.
	reResolved := anchorFor(t, inserted, "cross-encoder is cut")
	require.Equal(t, a.NodeHash, reResolved.NodeHash,
		"the anchored section's tier-2 hash must survive an insertion above it")

	v := resolve(inserted, a)
	require.Equal(t, anchor.Valid, v.Kind,
		"an insertion above must not expire and must not re-anchor: %s", v.Reason)
}

func TestAmbiguousResolutionExpires(t *testing.T) {
	// Two sections now carry the identical heading path "D-010 > Reasoning". The
	// resolver cannot tell which one the claim meant, so it expires rather than
	// guessing — never guesses is the rule.
	a := anchorFor(t, baseDoc, "cross-encoder is cut")

	dup := baseDoc + "\n## D-010\n\nA second, unrelated D-010.\n\n### Reasoning\n\nDifferent body entirely.\n"
	v := resolve(dup, a)
	require.Equal(t, anchor.Ambiguous, v.Kind, v.Reason)
	require.True(t, v.Kind.Expires())
}

func TestRemovedSectionExpires(t *testing.T) {
	// The anchored heading is gone entirely. Tier 1 fails to resolve: a semantic
	// change (the section was deleted or renamed), which expires.
	a := anchorFor(t, baseDoc, "cross-encoder is cut")

	removed := strings.Replace(baseDoc,
		"### Reasoning\n\nThe cross-encoder is cut because it is too slow on CPU at every model size.\n\n", "", 1)
	v := resolve(removed, a)
	require.Equal(t, anchor.SemanticChange, v.Kind, v.Reason)
	require.True(t, v.Kind.Expires())
}

func TestUnanchoredEvidenceIsNeverExpired(t *testing.T) {
	// A pre-existing tier-4-only row, or an attestation: no tier-1/2 anchor. The
	// ladder does not apply. It must never expire on tier 4 alone — that is the
	// exact over-expiry L3 forbids.
	tierFourOnly := anchor.Anchor{ByteHash: rawHash("some bytes")}
	require.False(t, tierFourOnly.Anchored())

	v := resolve(baseDoc, tierFourOnly)
	require.Equal(t, anchor.Unanchored, v.Kind, v.Reason)
	require.False(t, v.Kind.Expires(), "unanchored evidence is left untouched, never expired")
}

func TestEditElsewhereDoesNotExpire(t *testing.T) {
	// An edit to a *different* section (D-011) must not touch a claim anchored to
	// D-010 > Reasoning. This is the "insertions and edits elsewhere in the file"
	// property of tier 1.
	a := anchorFor(t, baseDoc, "cross-encoder is cut")

	elsewhere := strings.Replace(baseDoc,
		"Sovereignty is a tiebreaker, not a wedge.",
		"Sovereignty is now the headline wedge, reversing the earlier call.", 1)
	v := resolve(elsewhere, a)
	require.Equal(t, anchor.Valid, v.Kind, v.Reason)
}
