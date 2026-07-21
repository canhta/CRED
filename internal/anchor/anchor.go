// Package anchor implements L3's fingerprint ladder: the semantic anchor that
// decides whether a claim's evidence still holds after its source file changes.
//
// The ladder has four tiers (PRD section 4, L3):
//
//	tier 1  symbol path            survives line moves, reformatting, insertions above
//	tier 2  normalized node hash   survives formatting and headers
//	tier 3  context-window hash    survives small local edits
//	tier 4  raw byte hash          survives nothing — diagnostic only
//
// The law is exact and lives in Resolve: an anchor is valid **iff tiers 1 and 2
// agree**. Tier 4 changing while 1 and 2 hold is formatting churn and expires
// nothing. Tiers 1 and 2 disagreeing is a genuine semantic change. Ambiguous
// resolution expires the claim; it never guesses. A byte hash of a line range
// fails in both directions — a formatting commit erases a module's memory, and
// an insertion above silently re-anchors a claim onto different code that then
// validates. A confidently wrong claim is worse than no claim.
//
// This package is pure. Like internal/temporal and internal/acl it holds
// functions over domain types, imports no database driver, and takes no
// connection in any signature; depguard fails the build if that changes. The
// invalidation *decision* is deterministic and testable here; persisting the
// expiry it implies lives in internal/curate, on the store side of the boundary.
package anchor

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/canhta/cred/internal/claim"
)

// Anchor is the fingerprint ladder for one evidence span. Every hash is hex.
//
// SymbolPath and NodeHash are the only two tiers the law consults. WindowHash
// and ByteHash are retained for diagnostics and for a future tier-3 fallback —
// they never decide validity, because deciding on them is the failure L3 exists
// to prevent.
type Anchor struct {
	SymbolPath string // tier 1: heading path (text) or tree-sitter symbol path (code)
	NodeHash   string // tier 2: sha256 of the enclosing node's normalized text
	WindowHash string // tier 3: sha256 of a normalized window around the span
	ByteHash   string // tier 4: sha256 of the raw span bytes (== evidence.content_sha256)
}

// Anchored reports whether an anchor carries the two tiers the law needs. A
// zero SymbolPath or NodeHash means the ladder cannot be applied — a
// pre-existing tier-4-only row, or an attestation whose evidence is a person
// rather than a file span. Such evidence is never expired by re-anchoring,
// because expiring on tier 4 alone is exactly the mistake L3 forbids.
func (a Anchor) Anchored() bool {
	return a.SymbolPath != "" && a.NodeHash != ""
}

// Source is the current, trusted text of a file, plus what kind of file it is.
// Anchoring reads the whole file so tier 1 can be a structural path rather than
// a line number.
type Source struct {
	Text string
	Kind claim.SourceKind
}

// Span is a 1-based inclusive line range within a Source, plus the tier-4 hash
// the caller already computed for that span (evidence.content_sha256). Compute
// fills tiers 1–3 from the source and carries ByteHash through unchanged.
type Span struct {
	LineStart int
	LineEnd   int
	ByteHash  string
}

// VerdictKind is the outcome of resolving a stored anchor against a source.
type VerdictKind int

const (
	// Valid: tiers 1 and 2 agree. The claim survives. Tier 4 may have changed —
	// that is formatting churn and expires nothing.
	Valid VerdictKind = iota
	// SemanticChange: tier 1 or tier 2 disagrees. The anchored code or section
	// genuinely changed, or was removed. The claim expires.
	SemanticChange
	// Ambiguous: the symbol path resolves to more than one place in the current
	// source. The claim expires rather than guessing which one it meant.
	Ambiguous
	// Unanchored: the stored anchor lacks tiers 1–2 (a pre-existing tier-4-only
	// row, or an attestation). The ladder does not apply; the claim is left
	// untouched. Never an expiry — expiring on tier 4 alone is the L3 failure.
	Unanchored
)

func (k VerdictKind) String() string {
	switch k {
	case Valid:
		return "valid"
	case SemanticChange:
		return "semantic-change"
	case Ambiguous:
		return "ambiguous"
	case Unanchored:
		return "unanchored"
	default:
		return "unknown"
	}
}

// Expires reports whether this verdict should expire the claim. Only a
// semantic change or an ambiguous resolution does. This is the single place the
// "which verdicts invalidate" policy is stated.
func (k VerdictKind) Expires() bool {
	return k == SemanticChange || k == Ambiguous
}

// Verdict is a resolution outcome with a human-legible reason. L3's whole value
// is that the reason a claim survived or expired is a diff, not a score, so the
// reason is carried, not just the kind.
type Verdict struct {
	Kind   VerdictKind
	Reason string
}

// locateStatus is the result of finding a stored anchor's tier-1 path in a
// current source. It is the tier-1 half of the law; classify combines it with
// the tier-2 hash comparison.
type locateStatus int

const (
	found locateStatus = iota
	notFound
	ambiguous
)

// located is what an Anchorer's relocation produced: whether the tier-1 path was
// found in the current source and, if so, the tier-2 hash of what was found.
type located struct {
	status   locateStatus
	nodeHash string // tier 2 of the relocated node, valid when status == found
}

// classify is L3's law, stated once as a pure function of the stored anchor and
// the relocation result. Every Anchorer relocates by its own mechanics
// (headings for text, symbol paths for code) and then defers the *decision* to
// this function, so the law has exactly one implementation.
//
//   - not anchored          -> Unanchored     (ladder does not apply; do not expire)
//   - tier 1 gone           -> SemanticChange  (the section/symbol was removed or renamed)
//   - tier 1 ambiguous      -> Ambiguous       (two matches; never guess)
//   - tier 1 found, 2 same  -> Valid           (tiers 1 and 2 agree; tier 4 is irrelevant)
//   - tier 1 found, 2 diff  -> SemanticChange  (the anchored node's content changed)
func classify(stored Anchor, loc located) Verdict {
	if !stored.Anchored() {
		return Verdict{Unanchored, "evidence carries no tier-1/2 anchor (tier-4-only or attestation)"}
	}
	switch loc.status {
	case notFound:
		return Verdict{SemanticChange, "tier 1: symbol path " + quote(stored.SymbolPath) + " no longer present"}
	case ambiguous:
		return Verdict{Ambiguous, "tier 1: symbol path " + quote(stored.SymbolPath) + " resolves to more than one place"}
	default: // found
		if loc.nodeHash != stored.NodeHash {
			return Verdict{SemanticChange, "tier 2: enclosing node under " + quote(stored.SymbolPath) + " changed"}
		}
		return Verdict{Valid, "tiers 1 and 2 agree under " + quote(stored.SymbolPath)}
	}
}

// Anchorer computes an anchor for a span at ingest and resolves a stored anchor
// against a current source at re-anchor time. Text and code are different
// mechanics behind one interface, so the code anchorer (pure-Go tree-sitter,
// per the semantic-anchoring spike) is a drop-in that never touches this file.
type Anchorer interface {
	// Kind reports the source kind this anchorer handles.
	Kind() claim.SourceKind
	// Compute builds the ladder for span within src. It reads the whole file so
	// tier 1 is structural.
	Compute(src Source, span Span) Anchor
	// Resolve applies L3's law: locate the stored anchor's tier-1 path in the
	// current src, then decide. It is the deterministic invalidation check.
	Resolve(stored Anchor, src Source) Verdict
}

// For selects the anchorer for a source kind. It is the pluggable seam L3's
// ladder hangs on: the code anchorer (pure-Go tree-sitter, verified CGO-free by
// the semantic-anchoring spike) drops in at SourceCode without any caller
// changing. It does not ship in this slice — not because it needs CGO, which the
// spike disproved, but because no code evidence is produced yet and adopting a
// v0.x single-maintainer parser as the anchoring authority needs a grammar-
// fidelity verification first, the same discipline D-008 applied to the
// tokenizer. Until then code falls back to the text anchorer, which finds no
// heading path in Go and so records code evidence as tier-4-only rather than
// pretending to a symbol path it cannot produce.
//
// ok is false for attestations: a person's assertion is its own evidence and has
// no file span to re-anchor, so the ladder does not apply and the caller stores
// no anchor.
func For(kind claim.SourceKind) (Anchorer, bool) {
	switch kind {
	case claim.SourceDocument, claim.SourceCode:
		return TextAnchorer{}, true
	default: // attestation
		return nil, false
	}
}

// hashNormalized collapses all runs of whitespace to a single space, preserves
// case, and hashes the result. Whitespace-collapsed, case-preserved is what
// makes tier 2 survive reflow and reformatting while still catching a real edit.
func hashNormalized(s string) string {
	norm := strings.Join(strings.Fields(s), " ")
	if norm == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:])
}

func quote(s string) string { return "\"" + s + "\"" }

// splitLines splits without allocating a trailing empty element for a final
// newline, so line counts match an editor's.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines
}
