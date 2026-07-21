// Package recall orchestrates retrieval: two arms, reciprocal rank fusion,
// then the access-control decision.
//
// Fusion is a pure function over ranks and is unit-tested as one. The
// orchestration around it takes a store, but the ranking arithmetic never
// does.
package recall

import (
	"sort"
)

// RRFConstant is the k in 1/(k + rank). 60 is the published default and the
// value CRED commits to; changing it silently reorders everything.
const RRFConstant = 60.0

// Arm names a retrieval signal. Keeping the name on the contribution is what
// makes the score inspectable in `cred recall` output — the instrument whose
// absence left users of three competing systems unable to tell why a populated
// store returned nothing.
type Arm string

const (
	ArmDense   Arm = "dense"
	ArmLexical Arm = "lexical"
)

// Ranked is one arm's ordered result. Rank is 1-based.
type Ranked struct {
	ID   string
	Rank int
	Raw  float64
}

// Contribution records what one arm added to a fused score.
type Contribution struct {
	Arm   Arm
	Rank  int
	Raw   float64
	Score float64
}

// Fused is one candidate after fusion.
type Fused struct {
	ID            string
	Score         float64
	Contributions []Contribution
}

// FuseRRF combines ranked lists by reciprocal rank fusion.
//
// Ranks, never scores. BM25 scores are unbounded and corpus-dependent while
// cosine is bounded, so a weighted sum of normalized scores makes the weights
// query-dependent and untunable. Three failure modes this avoids, all silent:
//
//   - Direction. pgvector's <#> is a distance and ts_rank is a score. Fusing
//     them without normalizing direction ranks the worst results first, and
//     still returns topically related results, so it reads as noise rather
//     than as inversion. Converting to rank at the SQL boundary — each arm
//     ORDER BYs in its own correct direction — removes the question.
//   - 0-based rank. It shifts every score by roughly 1.6%: enough to reorder
//     near-ties, never enough to notice. Rank 0 is rejected below.
//   - Unequal arm depth, which systematically penalizes anything found by one
//     arm only. Assert equal depth with EqualDepth before calling this.
//
// Ties are broken by identifier so the output is a deterministic total order.
// A ranking that reorders between identical runs cannot be regression-tested.
func FuseRRF(arms map[Arm][]Ranked) []Fused {
	byID := make(map[string]*Fused)
	names := make([]Arm, 0, len(arms))
	for name := range arms {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool { return names[i] < names[j] })

	for _, name := range names {
		for _, r := range arms[name] {
			if r.Rank < 1 {
				// A 0-based rank is a bug that produces plausible output.
				// Refusing it is cheaper than finding it later.
				continue
			}
			score := 1.0 / (RRFConstant + float64(r.Rank))
			f, ok := byID[r.ID]
			if !ok {
				f = &Fused{ID: r.ID}
				byID[r.ID] = f
			}
			f.Score += score
			f.Contributions = append(f.Contributions, Contribution{
				Arm: name, Rank: r.Rank, Raw: r.Raw, Score: score,
			})
		}
	}

	out := make([]Fused, 0, len(byID))
	for _, f := range byID {
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// EqualDepth reports whether every arm returned the same number of candidates,
// or fewer only because the corpus ran out.
//
// Unequal depth systematically penalizes anything one arm alone found. An arm
// that returned fewer than depth results is fine when the corpus is smaller
// than depth; an arm truncated below depth by its own limit is not.
func EqualDepth(arms map[Arm][]Ranked, depth int) bool {
	for _, rs := range arms {
		if len(rs) > depth {
			return false
		}
	}
	return true
}

// SingleArmShare reports the fraction of the top n results contributed by the
// arm that dominates them.
//
// Above 0.95 the system is doing single-arm search with the latency of two,
// and that is worth an alarm rather than a shrug.
func SingleArmShare(fused []Fused, n int) (Arm, float64) {
	if n > len(fused) {
		n = len(fused)
	}
	if n == 0 {
		return "", 0
	}
	counts := make(map[Arm]int)
	for _, f := range fused[:n] {
		// A result found by both arms counts for neither: the concern is
		// results only one arm could see.
		if len(f.Contributions) == 1 {
			counts[f.Contributions[0].Arm]++
		}
	}
	var best Arm
	var most int
	for arm, c := range counts {
		if c > most || (c == most && arm < best) {
			best, most = arm, c
		}
	}
	return best, float64(most) / float64(n)
}
