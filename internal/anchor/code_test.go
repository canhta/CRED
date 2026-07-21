package anchor_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/anchor"
	"github.com/canhta/cred/internal/claim"
)

// These tests take no database and no connection — internal/anchor is pure. They
// are the code ladder's law across languages: the symbol path is the tier-1
// anchor, reformatting that preserves structure holds tiers 1 and 2 (expire
// nothing), an edit to the anchored definition's body changes tier 2 (expire),
// and renaming or removing the definition loses tier 1 (expire). A language the
// registry does not know produces no tier-1 anchor and is never expired.

// lineOf returns the 1-based line number of the first line containing marker.
func lineOf(t *testing.T, text, marker string) int {
	t.Helper()
	for i, l := range strings.Split(text, "\n") {
		if strings.Contains(l, marker) {
			return i + 1
		}
	}
	require.Failf(t, "marker not found", "%q", marker)
	return 0
}

// codeAnchor computes the anchor for the line containing marker in src, as the
// ingest path would for a file at path.
func codeAnchor(t *testing.T, path, src, marker string) anchor.Anchor {
	t.Helper()
	line := lineOf(t, src, marker)
	span := anchor.Span{LineStart: line, LineEnd: line, ByteHash: rawHash(marker)}
	return anchor.CodeAnchorer{}.Compute(
		anchor.Source{Text: src, Kind: claim.SourceCode, Path: path}, span)
}

func resolveCode(path, src string, a anchor.Anchor) anchor.Verdict {
	return anchor.CodeAnchorer{}.Resolve(a, anchor.Source{Text: src, Kind: claim.SourceCode, Path: path})
}

type codeCase struct {
	name     string
	path     string
	src      string
	marker   string // line inside the anchored definition
	want     string // expected tier-1 symbol path
	reformat func(string) string
	bodyEdit func(string) string
	rename   func(string) string
}

func replacer(pairs ...string) func(string) string {
	return func(s string) string { return strings.NewReplacer(pairs...).Replace(s) }
}

func replaceOnce(old, repl string) func(string) string {
	return func(s string) string { return strings.Replace(s, old, repl, 1) }
}

const goSrc = `package main

type Executor struct {
	limit int
}

func (e *Executor) writeOne(x int) int {
	total := x + e.limit
	return total
}

func Helper() string {
	return "ok"
}
`

const tsSrc = `export class Service {
  async handle(req: Request): Response {
    const y = req.id + 1;
    return y;
  }

  helper(): void {
    doThing();
  }
}
`

const pySrc = `class Repo:
    def fetch(self, id):
        row = self.db.get(id)
        return row

    def store(self, x):
        self.db.put(x)
`

const rustSrc = `pub struct Config {
    limit: u32,
}

impl Config {
    pub fn scaled(&self, factor: u32) -> u32 {
        let out = self.limit * factor;
        out
    }
}
`

const cSrc = `#include <stdio.h>

int add(int a, int b) {
    int sum = a + b;
    return sum;
}

int mul(int a, int b) {
    return a * b;
}
`

const cssSrc = `.card {
    color: black;
    padding: 4px;
}

.card .title {
    font-weight: bold;
}

@media (max-width: 600px) {
    .card {
        padding: 2px;
    }
}
`

func codeCases() []codeCase {
	return []codeCase{
		{
			name: "go method", path: "exec.go", src: goSrc,
			marker: "total := x + e.limit", want: "Executor.writeOne",
			reformat: replacer("\ttotal := x + e.limit\n", "\n\t\t\ttotal :=   x + e.limit\n\n"),
			bodyEdit: replaceOnce("x + e.limit", "x - e.limit"),
			rename:   replaceOnce("writeOne", "writeTwo"),
		},
		{
			name: "ts nested class method", path: "service.ts", src: tsSrc,
			marker: "const y = req.id + 1", want: "Service > handle",
			reformat: replacer("    const y = req.id + 1;\n", "\n        const y = req.id + 1;\n\n"),
			bodyEdit: replaceOnce("req.id + 1", "req.id + 2"),
			rename:   replaceOnce("async handle", "async process"),
		},
		{
			name: "python nested method", path: "repo.py", src: pySrc,
			marker: "row = self.db.get(id)", want: "Repo > fetch",
			reformat: replacer("        row = self.db.get(id)\n", "\n        row =   self.db.get(id)\n\n"),
			bodyEdit: replaceOnce("self.db.get(id)", "self.db.fetch(id)"),
			rename:   replaceOnce("def fetch", "def load"),
		},
		{
			name: "rust impl method", path: "config.rs", src: rustSrc,
			marker: "let out = self.limit * factor", want: "Config > scaled",
			reformat: replacer("        let out = self.limit * factor;\n", "\n        let out =  self.limit * factor;\n\n"),
			bodyEdit: replaceOnce("self.limit * factor", "self.limit + factor"),
			rename:   replaceOnce("fn scaled", "fn scale"),
		},
		{
			name: "c function", path: "math.c", src: cSrc,
			marker: "int sum = a + b", want: "add",
			reformat: replacer("    int sum = a + b;\n", "\n        int sum =  a + b;\n\n"),
			bodyEdit: replaceOnce("a + b", "a - b"),
			rename:   replaceOnce("int add(", "int plus("),
		},
		{
			name: "css selector", path: "card.css", src: cssSrc,
			marker: "color: black", want: ".card",
			reformat: replacer("    color: black;\n", "\n        color:   black;\n\n"),
			bodyEdit: replaceOnce("color: black", "color: white"),
			rename:   replaceOnce(".card {", ".panel {"),
		},
	}
}

func TestCodeComputeAndResolve(t *testing.T) {
	for _, tc := range codeCases() {
		t.Run(tc.name, func(t *testing.T) {
			a := codeAnchor(t, tc.path, tc.src, tc.marker)
			require.Equal(t, tc.want, a.SymbolPath, "tier 1 is the nested symbol path")
			require.NotEmpty(t, a.NodeHash, "tier 2 hashes the enclosing definition")
			require.True(t, a.Anchored())

			// A structure-preserving reformat (reindent, blank lines, whitespace)
			// holds tiers 1 and 2. A pure-formatting commit expires nothing.
			reformatted := tc.reformat(tc.src)
			require.NotEqual(t, tc.src, reformatted, "the fixture must actually reformat")
			v := resolveCode(tc.path, reformatted, a)
			require.Equal(t, anchor.Valid, v.Kind, "reformat must hold: %s", v.Reason)
			require.False(t, v.Kind.Expires())

			// Editing the anchored definition's body changes tier 2 but not tier 1.
			edited := tc.bodyEdit(tc.src)
			require.NotEqual(t, tc.src, edited, "the fixture must actually edit the body")
			v = resolveCode(tc.path, edited, a)
			require.Equal(t, anchor.SemanticChange, v.Kind, "body edit must expire: %s", v.Reason)
			require.True(t, v.Kind.Expires())

			// Renaming or removing the definition loses tier 1.
			renamed := tc.rename(tc.src)
			require.NotEqual(t, tc.src, renamed, "the fixture must actually rename")
			v = resolveCode(tc.path, renamed, a)
			require.Equal(t, anchor.SemanticChange, v.Kind, "rename must expire: %s", v.Reason)
			require.True(t, v.Kind.Expires())
		})
	}
}

func TestUnknownExtensionIsTierFourOnly(t *testing.T) {
	// An unregistered extension yields no tier-1 anchor; the span carries only its
	// raw hash. Such evidence resolves Unanchored and is never expired — expiring
	// on tier 4 alone is the failure the ladder exists to prevent.
	src := "some content\nfunction foo() { return 1 }\nmore content\n"
	a := codeAnchor(t, "notes.xyz", src, "function foo")
	require.Empty(t, a.SymbolPath, "an unknown language produces no symbol path")
	require.Empty(t, a.NodeHash)
	require.False(t, a.Anchored())

	v := resolveCode("notes.xyz", src, a)
	require.Equal(t, anchor.Unanchored, v.Kind, v.Reason)
	require.False(t, v.Kind.Expires())
}

func TestCodeEditElsewhereDoesNotExpire(t *testing.T) {
	// An edit to a sibling definition must not touch a claim anchored to another.
	a := codeAnchor(t, "exec.go", goSrc, "total := x + e.limit")
	elsewhere := strings.Replace(goSrc, `return "ok"`, `return "changed"`, 1)
	require.NotEqual(t, goSrc, elsewhere)
	v := resolveCode("exec.go", elsewhere, a)
	require.Equal(t, anchor.Valid, v.Kind, v.Reason)
}
