// Package nominate is the LLM boundary. It is the first place in CRED that
// crosses to a model, and it is built so the model can only ever *propose*.
//
// L2 — the model nominates, code decides — is enforced here structurally, not
// by convention. A Nominator returns candidate claims and holds nothing that
// can write, supersede, or expire: depguard forbids this package from importing
// internal/store, so an extractor with a database handle would not compile. The
// deterministic write executor lives in internal/curate, on the other side of
// this boundary, and it is the only thing that turns a Candidate into a row.
//
// L1 — no claim without evidence — is enforced by the shape of a Candidate: it
// carries a Quote, and the executor drops any candidate whose Quote is not a
// span of the trusted Input.Source. The evidence stored is that resolved span,
// taken from the source, never the free text the model returned. That is also
// the L8 defense: even a model that fabricates a statement cannot fabricate the
// evidence, because code reads the evidence out of the source itself.
package nominate

import (
	"context"
	"errors"
	"strings"

	"github.com/canhta/cred/internal/claim"
)

// PromptVersion is recorded on every claim written from a nomination
// (provenance, required by L8). Bump it whenever the prompt or schema changes,
// so a later quality regression is traceable to the prompt that produced it.
const PromptVersion = "nominate/v1"

// MaxStatementRunes bounds a candidate statement. The PRD keeps a claim's
// statement short; a model that returns a paragraph is proposing something
// other than a claim, and code declines it rather than storing it.
const MaxStatementRunes = 300

// Errors a Nominator may return. A nomination that produces no valid candidate
// is not an error — it is an empty slice, and the write path simply writes
// nothing. ErrNomination is reserved for a boundary that failed to produce any
// validated response at all (every attempt errored, timed out, or truncated).
var (
	ErrNomination = errors.New("nominate: no validated response from the model")
	ErrNoKey      = errors.New("nominate: no API key configured for the write path")
)

// Input is the trusted material to extract from, plus the context code needs to
// resolve and scope what the model proposes.
//
// Source is authoritative. Evidence is materialized from Source by the
// executor, never from the model's output, so Source must be exactly the text
// the model is shown. BaseLine lets the executor turn a byte offset in Source
// into a file line range.
type Input struct {
	Source     string           // the turn, tool result, or file span shown to the model
	SourceKind claim.SourceKind // document, code, or attestation
	Repo       string
	Path       string
	BaseLine   int // 1-based line number of Source's first line
	Scope      claim.Scope
	Principals []claim.PrincipalID
	Trigger    string // remember | turn | tool_result | session_end — provenance only
}

// Candidate is a proposed claim. It is the constrained schema the model emits
// into. It has no identifier, no ACL, and no evidence row — those are code's to
// assign. It points at the span that produced it with Quote.
type Candidate struct {
	Kind       claim.Kind `json:"kind"`
	Statement  string     `json:"statement"`
	Quote      string     `json:"quote"`
	Confidence float64    `json:"confidence"`
}

// Nominator proposes candidates from input. This is the entire authority the
// LLM boundary exposes: propose, and nothing else. There is no Write, no
// Supersede, no store in any signature.
type Nominator interface {
	Nominate(ctx context.Context, in Input) ([]Candidate, error)
}

// validKinds is the closed set from the PRD. A candidate proposing any other
// kind is dropped, because kind determines validity semantics and an unknown
// kind has none.
var validKinds = map[claim.Kind]struct{}{
	claim.KindConvention:       {},
	claim.KindDecision:         {},
	claim.KindConstraint:       {},
	claim.KindRejectedApproach: {},
	claim.KindFailure:          {},
	claim.KindReference:        {},
}

// Valid reports whether a candidate is well-formed enough for code to consider
// writing it. This is the L2 local validation: no provider validates structured
// output server-side, several silently drop schema constraints, and constrained
// decoding guarantees a valid prefix rather than valid JSON — so every field is
// checked here, in Go, where L2 says the decision belongs.
//
// It deliberately does not check that Quote resolves against a source: that
// needs the source, which lives with the executor. Valid is the schema gate;
// evidence resolution is the L1 gate, and both must pass.
func Valid(c Candidate) bool {
	if _, ok := validKinds[c.Kind]; !ok {
		return false
	}
	if strings.TrimSpace(c.Statement) == "" {
		return false
	}
	if len([]rune(c.Statement)) > MaxStatementRunes {
		return false
	}
	if strings.TrimSpace(c.Quote) == "" {
		return false
	}
	// Confidence is an explainable score in [0,1]. A model that returns 1.7 is
	// not more confident; it is out of schema. Ranges are enforced here because
	// the target providers drop numeric bounds from the schema silently.
	if c.Confidence < 0 || c.Confidence > 1 {
		return false
	}
	return true
}

// keep returns the candidates that pass Valid, preserving order. The dropped
// ones are gone, not stored — code decided.
func keep(cands []Candidate) []Candidate {
	out := cands[:0:0]
	for _, c := range cands {
		if Valid(c) {
			out = append(out, c)
		}
	}
	return out
}
