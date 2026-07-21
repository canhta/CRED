package nominate

import (
	"context"
	"strings"

	"github.com/canhta/cred/internal/claim"
)

// Fake is a Nominator that needs no network and no API key. It carries roughly
// 95% of the tests (per the worker-ops spike): the adversarial inputs that
// matter for the write path — an unresolvable quote, an out-of-schema kind, a
// truncated response — are constructible here and not recordable from a real
// provider.
//
// Its extraction is deterministic: one candidate per non-empty line of the
// source, with the line as both the quote and the statement. That is enough to
// exercise the whole pipeline (validate, resolve evidence against the source,
// write, dedup) without asserting anything about model quality, which is not
// this package's job to test.
//
// Crucially, Fake holds no store and no write authority — the same as the real
// extractor. The write path's zero-config property is preserved: tests never
// need a key, even though the *write* path in production does.
type Fake struct {
	// Kind is assigned to every candidate. Defaults to Reference.
	Kind claim.Kind
	// Return, if set, is returned verbatim, ignoring the source. It lets a test
	// inject a candidate whose quote does not resolve (L1) or whose kind is out
	// of schema (L2) without depending on the line-splitting heuristic.
	Return []Candidate
	// Err, if set, is returned as the nomination error.
	Err error
}

// Nominate implements Nominator.
func (f *Fake) Nominate(_ context.Context, in Input) ([]Candidate, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Return != nil {
		return keep(f.Return), nil
	}

	kind := f.Kind
	if kind == "" {
		kind = claim.KindReference
	}

	var out []Candidate
	for _, line := range strings.Split(in.Source, "\n") {
		s := strings.TrimSpace(line)
		if s == "" || strings.HasPrefix(s, "#") || strings.HasPrefix(s, "```") {
			continue
		}
		if len([]rune(s)) > MaxStatementRunes {
			s = string([]rune(s)[:MaxStatementRunes])
		}
		out = append(out, Candidate{
			Kind:       kind,
			Statement:  s,
			Quote:      s,
			Confidence: 0.6,
		})
	}
	return keep(out), nil
}

var _ Nominator = (*Fake)(nil)
