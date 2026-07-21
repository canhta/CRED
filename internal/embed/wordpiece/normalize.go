package wordpiece

import (
	"sort"
	"strings"

	"golang.org/x/text/unicode/norm"
)

// rrange is an inclusive codepoint range in a generated table.
type rrange struct{ lo, hi rune }

func inRanges(r rune, tbl []rrange) bool {
	i := sort.Search(len(tbl), func(i int) bool { return tbl[i].hi >= r })
	return i < len(tbl) && r >= tbl[i].lo
}

func isControl(r rune) bool    { return inRanges(r, controlRanges) }
func isWhitespace(r rune) bool { return inRanges(r, whitespaceRanges) }
func isMark(r rune) bool       { return inRanges(r, nonspacingMarkRanges) }
func isPunct(r rune) bool      { return inRanges(r, punctRanges) }
func isCJK(r rune) bool        { return inRanges(r, cjkRanges) }

// normalize runs BertNormalizer's four stages in the order the reference
// applies them: clean_text, handle_chinese_chars, strip_accents, lowercase.
//
// The order of the last two matters and is easy to invert. tokenizer.json sets
// strip_accents:null, which means "inherit lowercase" — so accents are
// stripped, and stripped *before* lowercasing. Reversing them silently
// corrupts every accented input.
func normalize(s string) string {
	return lowercase(stripAccents(padCJK(cleanText(s))))
}

// cleanText drops control characters and collapses every whitespace character
// to a plain space.
func cleanText(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch {
		case isControl(r):
			// Dropped. U+0000 and U+FFFD are in the generated table.
		case isWhitespace(r):
			b.WriteByte(' ')
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// padCJK surrounds each CJK character with spaces so the pre-tokenizer emits
// one word per character.
//
// The generated cjkRanges reproduce a defect: the reference has a
// 256-codepoint hole at U+2B820..U+2B91F that original BERT does not have.
// That is a bug in HuggingFace and it is reproduced deliberately, because the
// target is the tokenizer that trained the model, not the specification.
func padCJK(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if isCJK(r) {
			b.WriteByte(' ')
			b.WriteRune(r)
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// stripAccents decomposes to NFD and drops the nonspacing marks the reference
// tables recognise. Probing only covers NFD-stable codepoints because NFD runs
// first, so a decomposing codepoint never reaches the table.
func stripAccents(s string) string {
	d := norm.NFD.String(s)
	var b strings.Builder
	b.Grow(len(d))
	for _, r := range d {
		if isMark(r) {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// lowercase applies the reference's full case mapping. One rune can produce
// several — Turkish dotted I is the case that matters — so the table maps to a
// string, and strings.ToLower is never called.
func lowercase(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if low, ok := lowerMap[r]; ok {
			b.WriteString(low)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// preTokenize is BertPreTokenizer: split on whitespace, discarding it, then
// isolate each punctuation character as its own word.
func preTokenize(s string) []string {
	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	for _, r := range s {
		switch {
		case isWhitespace(r):
			flush()
		case isPunct(r):
			flush()
			words = append(words, string(r))
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return words
}
