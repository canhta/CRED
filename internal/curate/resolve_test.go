package curate

import "testing"

func TestLocateResolvesQuoteToLineRange(t *testing.T) {
	source := "line one\nline two\nline three\n"
	tests := []struct {
		name      string
		quote     string
		baseLine  int
		wantStart int
		wantEnd   int
		wantOK    bool
	}{
		{"first line", "line one", 1, 1, 1, true},
		{"middle line", "line two", 1, 2, 2, true},
		{"span across lines", "two\nline three", 1, 2, 3, true},
		{"base-line offset applies", "line one", 41, 41, 41, true},
		{"quote not in source is dropped", "line four", 1, 0, 0, false},
		{"empty quote is dropped", "", 1, 0, 0, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := locate(source, tc.quote, tc.baseLine)
			if ok != tc.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !ok {
				return
			}
			if got.lineStart != tc.wantStart || got.lineEnd != tc.wantEnd {
				t.Fatalf("lines = %d-%d, want %d-%d", got.lineStart, got.lineEnd, tc.wantStart, tc.wantEnd)
			}
			if got.text != tc.quote {
				t.Fatalf("text = %q, want the source span %q", got.text, tc.quote)
			}
		})
	}
}

func TestNormalizeStatementCollapsesCaseAndSpace(t *testing.T) {
	tests := []struct{ a, b string }{
		{"The sky is blue", "the   sky is blue"},
		{"  RRF at k=60 ", "rrf at k=60"},
		{"one\ntwo", "ONE TWO"},
	}
	for _, tc := range tests {
		if normalizeStatement(tc.a) != normalizeStatement(tc.b) {
			t.Fatalf("normalize(%q) != normalize(%q)", tc.a, tc.b)
		}
	}
	if normalizeStatement("sky is blue") == normalizeStatement("sky is red") {
		t.Fatal("distinct statements must not collide")
	}
}
