package seed_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/seed"
)

func write(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(p), 0o750))
	require.NoError(t, os.WriteFile(p, []byte(body), 0o600))
}

func TestDiscoverFindsInstructionFilesAndDocs(t *testing.T) {
	root := t.TempDir()
	write(t, root, "AGENTS.md", "# A\n")
	write(t, root, "CLAUDE.md", "@AGENTS.md\n")
	write(t, root, ".cursorrules", "rules\n")
	write(t, root, "docs/one.md", "# One\n")
	write(t, root, "docs/nested/two.md", "# Two\n")
	write(t, root, "docs/notes.txt", "ignored\n")
	write(t, root, "main.go", "package main\n")

	got, err := seed.Discover(root)
	require.NoError(t, err)

	var rel []string
	for _, p := range got {
		r, err := filepath.Rel(root, p)
		require.NoError(t, err)
		rel = append(rel, filepath.ToSlash(r))
	}
	require.ElementsMatch(t, []string{
		"AGENTS.md", "CLAUDE.md", ".cursorrules",
		"docs/one.md", "docs/nested/two.md",
	}, rel)
	require.NotContains(t, rel, "docs/notes.txt")
	require.NotContains(t, rel, "main.go")
}

// TestDiscoverIsStable — chunk ordinals are part of a chunk's identity, so an
// unstable walk order would supersede the entire corpus on every run.
func TestDiscoverIsStable(t *testing.T) {
	root := t.TempDir()
	for _, n := range []string{"b", "a", "c", "z", "m"} {
		write(t, root, "docs/"+n+".md", "# "+n+"\n\nbody\n")
	}
	first, err := seed.Discover(root)
	require.NoError(t, err)
	for range 10 {
		again, err := seed.Discover(root)
		require.NoError(t, err)
		require.Equal(t, first, again)
	}
}

// TestSeedsFromDocumentationNotHistory pins the documentation-not-history rule
// structurally. The seeder has no notion of a commit, and this asserts the
// target list stays that way: a claim whose evidence is immutable can never
// expire, which is the inverse of what CRED is for.
func TestSeedsFromDocumentationNotHistory(t *testing.T) {
	for _, f := range seed.TargetFiles {
		require.NotContains(t, strings.ToLower(f), "git")
		require.NotContains(t, strings.ToLower(f), "commit")
	}
	root := t.TempDir()
	write(t, root, ".git/COMMIT_EDITMSG", "a commit message\n")
	write(t, root, "AGENTS.md", "# A\n\nbody\n")

	got, err := seed.Discover(root)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Contains(t, got[0], "AGENTS.md")
}

func TestChunkFileProducesUsableLineRanges(t *testing.T) {
	root := t.TempDir()
	body := "# Title\n\nFirst paragraph.\n\n## Section\n\nSecond paragraph.\n"
	write(t, root, "docs/a.md", body)

	chunks, err := seed.ChunkFile(filepath.Join(root, "docs", "a.md"), root)
	require.NoError(t, err)
	require.NotEmpty(t, chunks)

	lines := strings.Split(body, "\n")
	for i, c := range chunks {
		require.Equal(t, i, c.Ordinal, "ordinals must be contiguous")
		require.Equal(t, "docs/a.md", c.Path)
		require.GreaterOrEqual(t, c.LineStart, 1, "line ranges are 1-based")
		require.GreaterOrEqual(t, c.LineEnd, c.LineStart)
		require.LessOrEqual(t, c.LineEnd, len(lines))
		require.NotEmpty(t, c.SHA256)
		require.Len(t, c.SHA256, 64)
		require.NotEmpty(t, c.Statement())
	}
}

// TestChunkingIsDeterministic — the content hash is the change detector, so
// identical input must produce identical chunks and identical hashes. If
// chunking drifted, every re-seed would supersede everything.
func TestChunkingIsDeterministic(t *testing.T) {
	root := t.TempDir()
	write(t, root, "docs/a.md",
		strings.Repeat("# H\n\nSome prose that fills a section.\n\n", 30))

	first, err := seed.ChunkFile(filepath.Join(root, "docs", "a.md"), root)
	require.NoError(t, err)
	require.NotEmpty(t, first)
	for range 5 {
		again, err := seed.ChunkFile(filepath.Join(root, "docs", "a.md"), root)
		require.NoError(t, err)
		require.Equal(t, first, again)
	}
}

// TestHeadingsInsideFencesAreNotStructure — a '#' inside a code block is a
// shell comment, and splitting on it would cut a chunk mid-example.
func TestHeadingsInsideFencesAreNotStructure(t *testing.T) {
	root := t.TempDir()
	write(t, root, "docs/a.md",
		"# Real heading\n\n```sh\n# not a heading\ncred migrate\n```\n\ntail\n")

	chunks, err := seed.ChunkFile(filepath.Join(root, "docs", "a.md"), root)
	require.NoError(t, err)
	require.Len(t, chunks, 1, "the fenced comment split the chunk")
	require.Equal(t, "Real heading", chunks[0].Heading)
	require.Contains(t, chunks[0].Text, "cred migrate")
}

func TestChangedContentChangesTheHash(t *testing.T) {
	root := t.TempDir()
	write(t, root, "docs/a.md", "# H\n\noriginal\n")
	before, err := seed.ChunkFile(filepath.Join(root, "docs", "a.md"), root)
	require.NoError(t, err)

	write(t, root, "docs/a.md", "# H\n\nchanged\n")
	after, err := seed.ChunkFile(filepath.Join(root, "docs", "a.md"), root)
	require.NoError(t, err)

	require.Len(t, before, 1)
	require.Len(t, after, 1)
	require.NotEqual(t, before[0].SHA256, after[0].SHA256)
}

func TestEmptyFileProducesNoChunks(t *testing.T) {
	root := t.TempDir()
	write(t, root, "docs/empty.md", "\n\n   \n")
	chunks, err := seed.ChunkFile(filepath.Join(root, "docs", "empty.md"), root)
	require.NoError(t, err)
	require.Empty(t, chunks, "a whitespace-only file must not produce an orphan claim")
}
