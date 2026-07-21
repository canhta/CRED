package curate

import "strings"

// span is a resolved evidence location: a line range in the source file and the
// exact bytes that back it.
type span struct {
	lineStart int
	lineEnd   int
	text      string
}

// locate resolves a candidate's quote against the trusted source (L1). It is a
// pure function so the evidence-resolution rule can be unit-tested without a
// database or a model.
//
// The quote must be a verbatim substring of the source. If it is not, the
// candidate has no evidence and locate reports !ok — the executor then drops it.
// The returned text is the slice of the *source*, not the caller's quote, so the
// evidence stored is provably the source's bytes even if the two were expected
// to be identical.
//
// baseLine is the 1-based line number of the source's first line, so a span
// found inside a file chunk carries the file's real line numbers.
func locate(source, quote string, baseLine int) (span, bool) {
	if quote == "" {
		return span{}, false
	}
	idx := strings.Index(source, quote)
	if idx < 0 {
		return span{}, false
	}
	if baseLine < 1 {
		baseLine = 1
	}
	start := baseLine + strings.Count(source[:idx], "\n")
	end := start + strings.Count(quote, "\n")
	return span{
		lineStart: start,
		lineEnd:   end,
		text:      source[idx : idx+len(quote)],
	}, true
}

// normalizeStatement produces the key for exact-hash deduplication (D-010).
// Two statements that differ only in case or whitespace hash to the same value
// and are treated as duplicates; anything else is a distinct claim. Exact
// hashing has a zero false-merge rate by construction, which is why v1 uses it
// and defers fuzzy matching — a tunable false-merge rate on a destructive
// operation is the bias that shipped as a library default for years.
func normalizeStatement(s string) string {
	return strings.Join(strings.Fields(strings.ToLower(s)), " ")
}
