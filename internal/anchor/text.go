package anchor

import (
	"strings"

	"github.com/canhta/cred/internal/claim"
)

// windowRadius is how many lines on each side of a span tier 3 hashes. Small,
// because tier 3's job is to survive a local edit while still noticing that the
// neighbourhood was rewritten — a wide window would survive nothing.
const windowRadius = 3

// TextAnchorer is the ladder for Markdown and plain-text evidence, which is the
// entire corpus today because seeding ingests documentation, not code. Tier 1 is
// the Markdown heading path — e.g. "D-010 > Reasoning" — which survives line
// moves and edits elsewhere in the file. Tier 2 is the normalized hash of the
// enclosing section, which survives reflow and reformatting.
//
// It handles both SourceDocument and code-as-text: a code file with no
// tree-sitter anchorer available falls back here and gets tiers 3–4 only (no
// heading path), which is honest — the ladder degrades to what it can compute
// rather than pretending to a symbol path it cannot produce.
type TextAnchorer struct{}

// Kind reports that this anchorer handles document evidence.
func (TextAnchorer) Kind() claim.SourceKind { return claim.SourceDocument }

// heading is one Markdown heading with the full path that leads to it and the
// extent of its *immediate* section.
type heading struct {
	level int
	path  string // "D-010 > Reasoning": the full ancestor chain, joined — tier 1
	start int    // 1-based line of the heading itself
	end   int    // 1-based inclusive last line before the next heading of ANY level
}

// parseHeadings returns every heading in text with its resolved path (the full
// ancestor chain) and the extent of the section it *immediately* owns — from its
// own line down to the line before the next heading of any level, child headings
// excluded. Headings inside fenced code blocks are sample text, not structure,
// and are skipped — the same rule internal/seed uses.
//
// Tier 1 (path) is the full chain, so it survives insertions and edits
// elsewhere. Tier 2 hashes the immediate section only, so a change under a child
// heading does not expire a claim anchored to the parent's prose. Using the full
// nested extent for tier 2 would make the H1 section cover the whole file, and
// any edit anywhere would expire the top-level claim — the coarse-grained
// over-expiry the ladder exists to avoid.
func parseHeadings(text string) []heading {
	lines := splitLines(text)
	var heads []heading
	var stack []heading // enclosing headings, by increasing level, for path building
	inFence := false

	for i, line := range lines {
		n := i + 1
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence || !strings.HasPrefix(line, "#") {
			continue
		}
		title := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if title == "" {
			continue // a bare "#" run with no title is not a heading
		}
		level := len(line) - len(strings.TrimLeft(line, "#"))

		// The previous heading's immediate section ends at the line above this one.
		if len(heads) > 0 {
			heads[len(heads)-1].end = n - 1
		}

		// Pop the ancestor stack to this heading's parent, then build the path.
		for len(stack) > 0 && stack[len(stack)-1].level >= level {
			stack = stack[:len(stack)-1]
		}
		path := title
		if len(stack) > 0 {
			path = stack[len(stack)-1].path + " > " + title
		}
		h := heading{level: level, path: path, start: n, end: len(lines)}
		heads = append(heads, h)
		stack = append(stack, h)
	}
	return heads
}

// enclosing returns the deepest heading whose section contains line, or false
// if line falls before any heading. "Deepest" is the one with the highest level
// among those covering the line, which is the section a claim is actually about.
func enclosing(heads []heading, line int) (heading, bool) {
	var best heading
	ok := false
	for _, h := range heads {
		if h.start <= line && line <= h.end {
			if !ok || h.level > best.level {
				best = h
				ok = true
			}
		}
	}
	return best, ok
}

// sectionText returns the raw lines [start,end] of text as one string.
func sectionText(text string, start, end int) string {
	lines := splitLines(text)
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	if start > end {
		return ""
	}
	return strings.Join(lines[start-1:end], "\n")
}

// Compute builds the ladder for span within src. Tier 1 is the heading path at
// the span's first line; tier 2 hashes the enclosing section; tier 3 hashes a
// small window; tier 4 is the raw span hash the caller already has.
func (TextAnchorer) Compute(src Source, span Span) Anchor {
	a := Anchor{ByteHash: span.ByteHash}

	heads := parseHeadings(src.Text)
	if h, ok := enclosing(heads, span.LineStart); ok {
		a.SymbolPath = h.path
		a.NodeHash = hashNormalized(sectionText(src.Text, h.start, h.end))
	}

	a.WindowHash = hashNormalized(sectionText(src.Text, span.LineStart-windowRadius, span.LineEnd+windowRadius))
	return a
}

// Resolve applies the ladder's law. It relocates the stored tier-1 heading path
// in the current source, then defers the decision to classify — the one place
// the law lives. Tier 4 is never consulted: a section whose heading path and
// normalized text both match is valid no matter how its bytes were reformatted.
func (TextAnchorer) Resolve(stored Anchor, src Source) Verdict {
	if !stored.Anchored() {
		return classify(stored, located{})
	}

	var matches []heading
	for _, h := range parseHeadings(src.Text) {
		if h.path == stored.SymbolPath {
			matches = append(matches, h)
		}
	}

	switch len(matches) {
	case 0:
		return classify(stored, located{status: notFound})
	case 1:
		h := matches[0]
		return classify(stored, located{
			status:   found,
			nodeHash: hashNormalized(sectionText(src.Text, h.start, h.end)),
		})
	default:
		return classify(stored, located{status: ambiguous})
	}
}
