// Package seed performs cold-start ingestion from a repository's own
// documentation.
//
// Documentation, never git history. That is D-016, and the reason is CRED's
// own thesis rather than convenience: a claim anchored to AGENTS.md:42 expires
// when that file changes, while a claim extracted from commit abc123 has
// immutable evidence and can therefore never expire. Seeding from history
// would fill the store with permanently unfalsifiable claims — the exact
// inverse of "a claim lives only while its evidence does". The tagline is a
// constraint on what may be ingested, not decoration.
//
// Nothing here calls a model. Chunking is deterministic and the claim
// statement is derived mechanically from the heading path, so L2 is not merely
// respected, it is unexercised: there is no nomination step to constrain.
package seed

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TargetFiles are the agent instruction files seeded from a repository root.
//
// The list matches what the one shipped competitor cold-start importer reads,
// deliberately: this is the validated mechanism, and diverging from it here
// would be inventing rather than building.
var TargetFiles = []string{
	"AGENTS.md",
	"CLAUDE.md",
	"README.md",
	".cursorrules",
	".windsurfrules",
}

// DocsGlob is the second source: every markdown file under docs/.
const DocsGlob = "docs"

// maxFileBytes skips files large enough to be generated output rather than
// written documentation.
const maxFileBytes = 512 * 1024

// Chunk is one unit of evidence: a contiguous span of one file.
type Chunk struct {
	Path      string
	Ordinal   int
	LineStart int
	LineEnd   int
	Text      string
	Heading   string
	SHA256    string
}

// Statement is the claim a chunk carries.
//
// Derived mechanically from the heading path and the chunk's first sentence.
// No model is involved, so the result is byte-identical across runs — which is
// what makes re-seeding idempotent for reasons beyond the content hash.
func (c Chunk) Statement() string {
	head := c.Heading
	if head == "" {
		head = strings.TrimSuffix(filepath.Base(c.Path), filepath.Ext(c.Path))
	}
	first := firstSentence(c.Text)
	if first == "" {
		return truncate(head, 300)
	}
	return truncate(head+" — "+first, 300)
}

// Discover returns the documentation files to seed from root, in a stable
// order. A stable order matters: chunk ordinals are part of a chunk's identity,
// and an unstable walk would supersede everything on every run.
func Discover(root string) ([]string, error) {
	var out []string
	seen := map[string]bool{}

	for _, name := range TargetFiles {
		p := filepath.Join(root, name)
		if fi, err := os.Stat(p); err == nil && fi.Mode().IsRegular() {
			if !seen[p] {
				out = append(out, p)
				seen[p] = true
			}
		}
	}

	docs := filepath.Join(root, DocsGlob)
	if fi, err := os.Stat(docs); err == nil && fi.IsDir() {
		err := filepath.WalkDir(docs, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if strings.HasPrefix(d.Name(), ".") {
					return fs.SkipDir
				}
				return nil
			}
			if strings.ToLower(filepath.Ext(p)) != ".md" {
				return nil
			}
			if !seen[p] {
				out = append(out, p)
				seen[p] = true
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walk %s: %w", docs, err)
		}
	}

	sort.Strings(out)
	return out, nil
}

// targetLines is the chunk size in lines. Documentation chunks are kept small
// enough that the embedding stays inside a short sequence bucket, which is
// what keeps seeding a laptop-scale operation rather than an overnight one.
const targetLines = 40

// Chunk splits a file into evidence-sized spans on markdown heading
// boundaries, falling back to a line count inside long sections.
//
// Line ranges are 1-based and inclusive, because that is what an editor and a
// `path:line` reference mean. Every chunk therefore points at something a
// human can open.
func ChunkFile(path string, relTo string) ([]Chunk, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Size() > maxFileBytes {
		return nil, nil
	}

	rel, err := filepath.Rel(relTo, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		lines = append(lines, sc.Text())
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var chunks []Chunk
	var buf []string
	heading := ""
	pendingHeading := ""
	start := 1
	inFence := false

	flush := func(end int) {
		text := strings.TrimSpace(strings.Join(buf, "\n"))
		buf = buf[:0]
		if text == "" {
			return
		}
		sum := sha256.Sum256([]byte(text))
		chunks = append(chunks, Chunk{
			Path:      rel,
			Ordinal:   len(chunks),
			LineStart: start,
			LineEnd:   end,
			Text:      text,
			Heading:   heading,
			SHA256:    hex.EncodeToString(sum[:]),
		})
	}

	for i, line := range lines {
		n := i + 1
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
		}
		// A heading inside a fenced block is sample text, not structure.
		if !inFence && strings.HasPrefix(line, "#") {
			if len(buf) > 0 {
				flush(n - 1)
				start = n
			}
			heading = pendingHeading
			if h := strings.TrimSpace(strings.TrimLeft(line, "#")); h != "" {
				heading = h
				pendingHeading = h
			}
		}
		buf = append(buf, line)
		if len(buf) >= targetLines && !inFence {
			flush(n)
			start = n + 1
		}
	}
	flush(len(lines))

	// Re-number after empty chunks were skipped, so ordinals are contiguous.
	for i := range chunks {
		chunks[i].Ordinal = i
	}
	return chunks, nil
}

func firstSentence(s string) string {
	s = strings.TrimSpace(s)
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "```") {
			continue
		}
		if i := strings.Index(line, ". "); i > 0 {
			return line[:i+1]
		}
		return line
	}
	return ""
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
