package nominate

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"
)

// Usage is what one model call cost, for cost attribution. Recorded even on a
// call that is discarded (truncated or invalid), because a call that produced
// nothing usable still spent tokens.
//
// Wall is the wall-clock of the single Generate call, measured by the Extractor
// around the boundary — the provider does not report it. It is the third cost
// dimension tracked, alongside inference calls and tokens.
type Usage struct {
	InputTokens  int
	OutputTokens int
	Wall         time.Duration
}

// StopLength is the stop reason that means the model ran out of output budget.
// Anthropic names it "max_tokens"; OpenAI names it "length". Either way, under
// constrained decoding a truncated response is a *valid JSON prefix* that
// parses cleanly and is silently wrong, so a response that stops for this
// reason is discarded rather than parsed.
const StopLength = "max_tokens"

// Model is the raw LLM call: a prompt and a JSON schema in, raw bytes and a stop
// reason out. It performs no validation, because that is exactly what no
// provider does reliably server-side, and validation must be done locally in code.
//
// This is the whole surface the LLM boundary needs. Swapping Anthropic for
// another provider is a new Model, not a change to the extractor.
type Model interface {
	Generate(ctx context.Context, prompt string, schema []byte) (raw []byte, stopReason string, usage Usage, err error)
}

// UsageSink records model cost. The ledger lives above this package (in the
// store, keyed per principal and per scope); this is the hook it attaches to. A
// nil sink is fine.
//
// Record is handed the Input as well as the Usage so the sink can attribute cost
// to the principal and scope that occasioned the call — per-principal,
// per-scope attribution. nominate still may not reach the store — depguard
// forbids it — so the sink is an interface implemented on the other side of the
// boundary and passed in.
type UsageSink interface {
	Record(ctx context.Context, in Input, u Usage)
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
		start := time.Now()
		raw, stop, usage, err := e.model.Generate(ctx, prompt, schema)
		usage.Wall = time.Since(start) // the provider does not report wall-clock
		if e.usage != nil {
			// Attribute to the principal and scope in Input, and record even on
			// failure — a truncated or errored call still spent tokens.
			e.usage.Record(ctx, in, usage)
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
