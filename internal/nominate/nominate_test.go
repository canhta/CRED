package nominate_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/nominate"
)

func TestValidRejectsOutOfSchema(t *testing.T) {
	base := nominate.Candidate{
		Kind: claim.KindDecision, Statement: "we use RRF at k=60",
		Quote: "we use RRF at k=60", Confidence: 0.8,
	}
	require.True(t, nominate.Valid(base), "a well-formed candidate should validate")

	tests := []struct {
		name string
		mut  func(c *nominate.Candidate)
	}{
		{"unknown kind", func(c *nominate.Candidate) { c.Kind = "Opinion" }},
		{"empty statement", func(c *nominate.Candidate) { c.Statement = "  " }},
		{"empty quote has no supporting span", func(c *nominate.Candidate) { c.Quote = "" }},
		{"confidence above 1", func(c *nominate.Candidate) { c.Confidence = 1.7 }},
		{"confidence below 0", func(c *nominate.Candidate) { c.Confidence = -0.1 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := base
			tc.mut(&c)
			require.False(t, nominate.Valid(c))
		})
	}
}

func TestValidRejectsOverlongStatement(t *testing.T) {
	long := make([]byte, nominate.MaxStatementRunes+1)
	for i := range long {
		long[i] = 'x'
	}
	c := nominate.Candidate{
		Kind: claim.KindReference, Statement: string(long),
		Quote: "x", Confidence: 0.5,
	}
	require.False(t, nominate.Valid(c), "a paragraph is not a claim")
}

// stubModel returns scripted responses so the gating and validation logic can be
// exercised without a network or a key — the boundary the write path's
// zero-config test property depends on.
type stubModel struct {
	responses []stubResponse
	calls     int
}

type stubResponse struct {
	raw  string
	stop string
	err  error
}

func (m *stubModel) Generate(_ context.Context, _ string, _ []byte) ([]byte, string, nominate.Usage, error) {
	r := m.responses[min(m.calls, len(m.responses)-1)]
	m.calls++
	return []byte(r.raw), r.stop, nominate.Usage{OutputTokens: 10}, r.err
}

func TestExtractorDiscardsTruncatedResponse(t *testing.T) {
	// A truncated response under constrained decoding is a valid JSON prefix
	// that parses cleanly and is silently wrong. It must never be parsed.
	valid := `{"candidates":[{"kind":"Decision","statement":"use RRF","quote":"use RRF","confidence":0.8}]}`
	m := &stubModel{responses: []stubResponse{
		{raw: valid, stop: nominate.StopLength}, // truncated: discard, retry
		{raw: valid, stop: "end_turn"},          // second attempt is clean
	}}
	ex := nominate.NewExtractor(m, nil, nil)

	cands, err := ex.Nominate(context.Background(), nominate.Input{Source: "use RRF"})
	require.NoError(t, err)
	require.Len(t, cands, 1)
	require.Equal(t, 2, m.calls, "the truncated response should have been discarded and retried")
}

func TestExtractorFailsWhenEveryAttemptTruncates(t *testing.T) {
	valid := `{"candidates":[{"kind":"Decision","statement":"x","quote":"x","confidence":0.5}]}`
	m := &stubModel{responses: []stubResponse{{raw: valid, stop: nominate.StopLength}}}
	ex := nominate.NewExtractor(m, nil, nil)

	_, err := ex.Nominate(context.Background(), nominate.Input{Source: "x"})
	require.ErrorIs(t, err, nominate.ErrNomination)
}

func TestExtractorDropsInvalidCandidatesKeepsValid(t *testing.T) {
	// One valid candidate, one with an out-of-schema kind, one with confidence
	// out of range. Code decides: the valid one survives, the others are gone,
	// and the batch is not abandoned because two members were bad.
	raw := `{"candidates":[
		{"kind":"Decision","statement":"keep me","quote":"keep me","confidence":0.9},
		{"kind":"Opinion","statement":"drop me","quote":"drop me","confidence":0.5},
		{"kind":"Reference","statement":"drop me too","quote":"drop me too","confidence":9}
	]}`
	m := &stubModel{responses: []stubResponse{{raw: raw, stop: "end_turn"}}}
	ex := nominate.NewExtractor(m, nil, nil)

	cands, err := ex.Nominate(context.Background(), nominate.Input{Source: "keep me"})
	require.NoError(t, err)
	require.Len(t, cands, 1)
	require.Equal(t, "keep me", cands[0].Statement)
}

func TestExtractorRetriesInvalidJSON(t *testing.T) {
	valid := `{"candidates":[{"kind":"Reference","statement":"x","quote":"x","confidence":0.5}]}`
	m := &stubModel{responses: []stubResponse{
		{raw: "not json at all", stop: "end_turn"},
		{raw: valid, stop: "end_turn"},
	}}
	ex := nominate.NewExtractor(m, nil, nil)

	cands, err := ex.Nominate(context.Background(), nominate.Input{Source: "x"})
	require.NoError(t, err)
	require.Len(t, cands, 1)
}

func TestFakeNeedsNoKeyAndExtractsDeterministically(t *testing.T) {
	f := &nominate.Fake{Kind: claim.KindConvention}
	in := nominate.Input{Source: "# heading\nfirst line\n\nsecond line\n"}

	a, err := f.Nominate(context.Background(), in)
	require.NoError(t, err)
	b, err := f.Nominate(context.Background(), in)
	require.NoError(t, err)

	require.Equal(t, a, b, "the fake must be deterministic")
	require.Len(t, a, 2, "one candidate per non-heading, non-blank line")
	for _, c := range a {
		require.Equal(t, claim.KindConvention, c.Kind)
		require.Equal(t, c.Statement, c.Quote, "the fake quotes the line it extracts from")
	}
}

func TestFakePropagatesError(t *testing.T) {
	sentinel := errors.New("boom")
	f := &nominate.Fake{Err: sentinel}
	_, err := f.Nominate(context.Background(), nominate.Input{Source: "x"})
	require.ErrorIs(t, err, sentinel)
}

func TestNewAnthropicModelRequiresKey(t *testing.T) {
	_, err := nominate.NewAnthropicModel("")
	require.ErrorIs(t, err, nominate.ErrNoKey)

	m, err := nominate.NewAnthropicModel("sk-test", nominate.WithModel("claude-haiku-4-5"))
	require.NoError(t, err)
	require.NotNil(t, m)
}
