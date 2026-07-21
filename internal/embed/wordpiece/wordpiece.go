// Package wordpiece implements the BERT WordPiece tokenizer for
// bge-small-en-v1.5, byte-identically to HuggingFace `tokenizers`.
//
// This package must not call unicode.Is, unicode.IsPunct, strings.ToLower, or
// any other function backed by Go's Unicode tables. Go's tables are current and
// correct; the reference tokenizer's are frozen and, in one place, wrong. Both
// properties disqualify them here — the goal is byte-identity with the
// tokenizer that trained the model, not Unicode correctness. The two disagree
// on hundreds of codepoints and every disagreement is invisible on English
// text, which is the worst failure shape available.
//
// The character-class tables live in tables_gen.go and are probed out of the
// pinned reference release by internal/gen/probe_tables.py. Regenerate them;
// never edit them.
package wordpiece

import (
	"bufio"
	_ "embed"
	"fmt"
	"strings"
)

// The generator emits valid but unaligned Go; gofmt owns the alignment, so the
// generated file is byte-stable across regenerations.
//go:generate python3 internal/gen/probe_tables.py --out tables_gen.go
//go:generate gofmt -w tables_gen.go

// vocabTxt is the 30,522-entry vocabulary of BAAI/bge-small-en-v1.5 (MIT).
// It is committed rather than fetched so that tokenization — and therefore the
// conformance suite — needs no network and no model cache.
//
//go:embed vocab.txt
var vocabTxt string

// Special tokens, in the order HuggingFace's AddedVocabulary scans for them.
// All five carry normalized:false, which is why they are matched against the
// raw input before the normalizer runs.
const (
	clsToken  = "[CLS]"
	sepToken  = "[SEP]"
	unkToken  = "[UNK]"
	padToken  = "[PAD]"
	maskToken = "[MASK]"
)

var addedTokens = []string{padToken, unkToken, clsToken, sepToken, maskToken}

// maxInputCharsPerWord comes from tokenizer.json's WordPiece config. A word
// longer than this becomes a single [UNK] without being decomposed.
const maxInputCharsPerWord = 100

// MaxSequenceLength is the model's context, including [CLS] and [SEP].
const MaxSequenceLength = 512

const continuingPrefix = "##"

// Tokenizer is safe for concurrent use once constructed.
type Tokenizer struct {
	vocab map[string]int32
	unkID int32
	clsID int32
	sepID int32
}

// New builds a tokenizer over the embedded vocabulary.
func New() (*Tokenizer, error) {
	vocab := make(map[string]int32, 30600)
	sc := bufio.NewScanner(strings.NewReader(vocabTxt))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	var i int32
	for sc.Scan() {
		vocab[sc.Text()] = i
		i++
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read vocab: %w", err)
	}
	t := &Tokenizer{vocab: vocab}
	for _, spec := range []struct {
		tok string
		dst *int32
	}{{unkToken, &t.unkID}, {clsToken, &t.clsID}, {sepToken, &t.sepID}} {
		id, ok := vocab[spec.tok]
		if !ok {
			return nil, fmt.Errorf("vocabulary is missing %s", spec.tok)
		}
		*spec.dst = id
	}
	return t, nil
}

// Encode returns the token IDs for text, wrapped in [CLS] … [SEP] and
// truncated to MaxSequenceLength. It is the equivalent of
// tok(text, truncation=True, max_length=512)["input_ids"].
func (t *Tokenizer) Encode(text string) []int32 {
	ids := make([]int32, 0, 32)
	ids = append(ids, t.clsID)

	budget := MaxSequenceLength - 2 // [CLS] and [SEP]
	for _, seg := range splitOnAddedTokens(text) {
		if len(ids)-1 >= budget {
			break
		}
		if seg.special {
			ids = append(ids, t.vocab[seg.text])
			continue
		}
		for _, word := range preTokenize(normalize(seg.text)) {
			if len(ids)-1 >= budget {
				break
			}
			ids = t.appendWordPieces(ids, word)
		}
	}

	if len(ids) > budget+1 {
		ids = ids[:budget+1]
	}
	return append(ids, t.sepID)
}

// appendWordPieces runs greedy longest-match-first WordPiece over one
// pre-tokenized word.
func (t *Tokenizer) appendWordPieces(ids []int32, word string) []int32 {
	runes := []rune(word)
	if len(runes) > maxInputCharsPerWord {
		return append(ids, t.unkID)
	}

	// Collect into a scratch slice: WordPiece is all-or-nothing. If any
	// substring fails to match, the whole word is a single [UNK] and the
	// pieces found so far are discarded.
	start := 0
	pieces := make([]int32, 0, 8)
	for start < len(runes) {
		end := len(runes)
		matched := int32(-1)
		for end > start {
			sub := string(runes[start:end])
			if start > 0 {
				sub = continuingPrefix + sub
			}
			if id, ok := t.vocab[sub]; ok {
				matched = id
				break
			}
			end--
		}
		if matched < 0 {
			return append(ids, t.unkID)
		}
		pieces = append(pieces, matched)
		start = end
	}
	return append(ids, pieces...)
}

type segment struct {
	text    string
	special bool
}

// splitOnAddedTokens scans the raw, un-normalized input for the five special
// tokens and splits around them.
//
// This ordering is not incidental. HuggingFace's AddedVocabulary runs before
// the normalizer, and all five tokens carry normalized:false, so lowercasing
// never reaches them. An implementation that normalizes first can never
// recover the match, because "[CLS]" has already become "[cls]". CRED's corpus
// is documentation and source code, where bracketed uppercase tokens occur
// naturally in logs, changelogs and template strings.
func splitOnAddedTokens(text string) []segment {
	segs := []segment{{text: text}}
	for _, tok := range addedTokens {
		var next []segment
		for _, seg := range segs {
			if seg.special {
				next = append(next, seg)
				continue
			}
			rest := seg.text
			for {
				i := strings.Index(rest, tok)
				if i < 0 {
					break
				}
				if i > 0 {
					next = append(next, segment{text: rest[:i]})
				}
				next = append(next, segment{text: tok, special: true})
				rest = rest[i+len(tok):]
			}
			if rest != "" {
				next = append(next, segment{text: rest})
			}
		}
		segs = next
	}
	return segs
}
