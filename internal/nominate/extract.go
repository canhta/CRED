package nominate

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
)

// Usage is what one model call cost, for cost attribution. Recorded even on a
// call that is discarded (truncated or invalid), because a call that produced
// nothing usable still spent tokens.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

// StopLength is the stop reason that means the model ran out of output budget.
// Anthropic names it "max_tokens"; OpenAI names it "length". Either way, under
// constrained decoding a truncated response is a *valid JSON prefix* that
// parses cleanly and is silently wrong, so a response that stops for this
// reason is discarded rather than parsed (L2).
const StopLength = "max_tokens"

// Model is the raw LLM call: a prompt and a JSON schema in, raw bytes and a stop
// reason out. It performs no validation, because that is exactly what no
// provider does reliably server-side and what L2 requires be done locally.
//
// This is the whole surface the LLM boundary needs. Swapping Anthropic for
// another provider is a new Model, not a change to the extractor.
type Model interface {
	Generate(ctx context.Context, prompt string, schema []byte) (raw []byte, stopReason string, usage Usage, err error)
}

// UsageSink records model cost. The ledger the worker-ops spike specifies lives
// above this package; this is the hook it attaches to. A nil sink is fine.
type UsageSink interface {
	Record(ctx context.Context, u Usage)
}

// Extractor is the production Nominator: it prompts a Model, gates the response
// on its stop reason, validates every candidate locally, and drops the invalid
// ones. It holds a Model and nothing else — no store, no write path.
type Extractor struct {
	model    Model
	attempts int
	usage    UsageSink
	log      *slog.Logger
}

// NewExtractor builds an Extractor. attempts <= 0 defaults to 3.
func NewExtractor(model Model, usage UsageSink, log *slog.Logger) *Extractor {
	return &Extractor{model: model, attempts: 3, usage: usage, log: log}
}

// wire is the JSON shape the model is asked to emit. It is a flat object with
// one array, which is the intersection every target provider honors: Anthropic
// (the binding constraint) rejects recursion and numeric bounds, so the schema
// is flat and the bounds are re-checked in Valid.
type wire struct {
	Candidates []Candidate `json:"candidates"`
}

// Nominate implements Nominator.
func (e *Extractor) Nominate(ctx context.Context, in Input) ([]Candidate, error) {
	prompt := buildPrompt(in)
	schema := candidateSchema()

	attempts := e.attempts
	if attempts <= 0 {
		attempts = 3
	}

	for attempt := 0; attempt < attempts; attempt++ {
		raw, stop, usage, err := e.model.Generate(ctx, prompt, schema)
		if e.usage != nil {
			e.usage.Record(ctx, usage) // record even on failure — it still cost tokens
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil, err // caller cancelled or timed out; do not burn retries
			}
			e.logf("nominate model call failed", slog.Int("attempt", attempt), slog.String("error", err.Error()))
			continue
		}
		if stop == StopLength {
			// Constrained decoding makes truncation a valid JSON prefix that
			// parses cleanly and is silently wrong. Never parse it.
			e.logf("nominate response truncated, discarding", slog.Int("attempt", attempt))
			continue
		}

		var w wire
		if err := json.Unmarshal(raw, &w); err != nil {
			e.logf("nominate response was not valid json", slog.Int("attempt", attempt))
			continue
		}
		// Validate locally and drop the invalid. Code decides what is written,
		// not the model — an out-of-schema candidate is silently dropped, not an
		// error that abandons the valid ones alongside it.
		return keep(w.Candidates), nil
	}
	return nil, fmt.Errorf("%w after %d attempts", ErrNomination, attempts)
}

func (e *Extractor) logf(msg string, attrs ...any) {
	if e.log != nil {
		e.log.Warn(msg, attrs...)
	}
}

var _ Nominator = (*Extractor)(nil)
