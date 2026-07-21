package wordpiece

import (
	"bufio"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// The three suites are the ones that validated the tokenizer to byte-identity
// before it was adopted. They are reused rather than re-derived, and the fixtures
// are produced by internal/gen/make_fixtures.py against the pinned reference release.
//
// A tokenizer that is nearly right is worthless: one wrong token ID produces a
// different vector and recall degrades with no error. A textbook-correct
// implementation over Go's own Unicode tables scored 99.67% here, and every
// failure was invisible on English text.

type fixture struct {
	Text string  `json:"t"`
	IDs  []int32 `json:"i"`
}

func loadFixtures(t *testing.T, name string) []fixture {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", name))
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	zr, err := gzip.NewReader(f)
	require.NoError(t, err)
	defer func() { require.NoError(t, zr.Close()) }()

	var out []fixture
	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		var fx fixture
		require.NoError(t, json.Unmarshal(sc.Bytes(), &fx))
		out = append(out, fx)
	}
	require.NoError(t, sc.Err())
	require.NotEmpty(t, out, "fixture %s is empty", name)
	return out
}

func conform(t *testing.T, name string) {
	t.Helper()
	tok, err := New()
	require.NoError(t, err)

	fixtures := loadFixtures(t, name)
	mismatches := 0
	for _, fx := range fixtures {
		got := tok.Encode(fx.Text)
		if len(got) != len(fx.IDs) {
			mismatches++
		} else {
			for i := range got {
				if got[i] != fx.IDs[i] {
					mismatches++
					break
				}
			}
		}
		// Report the first few in full; a wall of diffs helps nobody.
		if mismatches > 0 && mismatches <= 5 {
			t.Errorf("mismatch on %q\n  want %v\n  got  %v", fx.Text, fx.IDs, got)
		}
	}
	require.Zero(t, mismatches, "%d of %d cases in %s did not match the reference",
		mismatches, len(fixtures), name)
}

func TestConformanceCurated(t *testing.T)    { conform(t, "curated.jsonl.gz") }
func TestConformanceFuzz(t *testing.T)       { conform(t, "fuzz.jsonl.gz") }
func TestConformanceCodepoints(t *testing.T) { conform(t, "codepoints.jsonl.gz") }

// TestSpecialTokensMatchBeforeNormalization pins the special-token defect:
// normalizing first destroys the match and it cannot be recovered.
func TestSpecialTokensMatchBeforeNormalization(t *testing.T) {
	tok, err := New()
	require.NoError(t, err)
	ids := tok.Encode("[CLS] x [SEP]")
	require.Equal(t, tok.clsID, ids[0])
	require.Equal(t, tok.clsID, ids[1], "literal [CLS] must survive lowercasing")
	require.Equal(t, tok.sepID, ids[len(ids)-1])
	require.Equal(t, tok.sepID, ids[len(ids)-2])
}

// TestReferenceTableQuirks pins the two places where the reference disagrees
// with both Go's tables and the BERT specification. Reproducing the
// disagreement is the point; "correct" is not the goal.
func TestReferenceTableQuirks(t *testing.T) {
	tests := []struct {
		name string
		r    rune
		want bool
		got  func(rune) bool
	}{
		// Unicode version skew: current tables call these an accent and a
		// punctuation mark. The frozen reference tables have them unassigned.
		{"U+1ABF is not a stripped mark", 0x1ABF, false, isMark},
		{"U+061D is not punctuation", 0x061D, false, isPunct},
		// The 256-codepoint hole at U+2B820..U+2B91F, CJK Extension E.
		// Original BERT specifies one contiguous range; the reference does not.
		{"U+2B81F is CJK", 0x2B81F, true, isCJK},
		{"U+2B820 is not CJK", 0x2B820, false, isCJK},
		{"U+2B91F is not CJK", 0x2B91F, false, isCJK},
		{"U+2B920 is CJK", 0x2B920, true, isCJK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, tc.got(tc.r))
		})
	}
}

func TestTruncationIncludesSpecialTokens(t *testing.T) {
	tok, err := New()
	require.NoError(t, err)
	long := ""
	for range 2000 {
		long += "word "
	}
	ids := tok.Encode(long)
	require.Len(t, ids, MaxSequenceLength)
	require.Equal(t, tok.clsID, ids[0])
	require.Equal(t, tok.sepID, ids[len(ids)-1])
}

func BenchmarkEncode(b *testing.B) {
	tok, err := New()
	require.NoError(b, err)
	const text = "Hybrid retrieval fuses dense vectors and PostgreSQL full-text " +
		"search with reciprocal rank fusion at k=60."
	b.ReportAllocs()
	for b.Loop() {
		_ = tok.Encode(text)
	}
}
