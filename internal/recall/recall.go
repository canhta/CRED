package recall

import (
	"context"
	"fmt"
	"time"

	"github.com/canhta/cred/internal/acl"
	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/store/pg"
)

// Store is the subset of the row store retrieval needs.
//
// Note what is absent: no method takes a principal, and none returns a
// decision. The store hands back rows; internal/acl decides who may see them.
type Store interface {
	PresentModel(ctx context.Context) (id int, name string, dims int, err error)
	DenseSearch(ctx context.Context, modelID int, vec []float32, k int) ([]pg.Hit, error)
	LexicalSearch(ctx context.Context, query string, k int) ([]pg.Hit, error)
	LoadClaims(ctx context.Context, ids []string) ([]claim.Claim, error)
	LoadEvidence(ctx context.Context, claimIDs []string) (map[string][]claim.Evidence, error)
	LoadClaimACLs(ctx context.Context, claimIDs []string) (map[string]claim.ACL, error)
	LoadEvidenceACLs(ctx context.Context, evidenceIDs []string) (map[string]claim.ACL, error)
}

// Embedder is the query-side of internal/embed.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	ModelName() string
}

// TokenCounter measures a string in model tokens. The CLI and the MCP server
// pass the real WordPiece tokenizer, so the budget is exact rather than a
// characters-over-four guess.
type TokenCounter func(string) int

// DefaultCandidateDepth is how deep each arm fetches before fusion. Both arms
// use the same depth; unequal depth systematically penalizes anything one arm
// alone found.
const DefaultCandidateDepth = 50

// DefaultTokenBudget is the assembled package's ceiling. Adherence collapses
// above roughly 6,000 words, and exceeding the ceiling drops whole claims
// rather than truncating text — a half-quoted claim is a claim whose evidence
// no longer supports it.
const DefaultTokenBudget = 4000

// Request is one recall.
type Request struct {
	Query     string
	Principal claim.PrincipalID
	Limit     int
	Budget    int
	Depth     int
	// Now is injected rather than read, so ACL TTL evaluation is testable and
	// replay is deterministic.
	Now time.Time
}

// Scored is a claim with the reason it ranked where it did.
type Scored struct {
	Claim         claim.Claim
	Score         float64
	Contributions []Contribution
	Tokens        int
}

// Timings records where the latency went, per arm.
type Timings struct {
	Embed   time.Duration
	Dense   time.Duration
	Lexical time.Duration
	Load    time.Duration
	Total   time.Duration
}

// Result is the assembled package.
type Result struct {
	Claims []Scored

	// AsOf and StalenessSeconds are on every response. A response with no
	// freshness marker is a response whose age the caller must guess.
	AsOf             time.Time
	StalenessSeconds float64

	// Candidates is what fusion saw; Authorized is what survived L5;
	// OmittedForBudget is what the ceiling dropped. Truncation is reported
	// explicitly, because a silent truncation is a lie.
	Candidates       int
	Authorized       int
	OmittedForBudget int

	TokensUsed  int
	TokenBudget int

	DominantArm   Arm
	DominantShare float64

	Timings Timings
}

// Service runs the retrieval pipeline.
type Service struct {
	store  Store
	embed  Embedder
	tokens TokenCounter
}

// New builds a Service.
func New(store Store, embedder Embedder, tokens TokenCounter) *Service {
	return &Service{store: store, embed: embedder, tokens: tokens}
}

// Recall retrieves the claims relevant to req.Query that req.Principal may
// read.
//
// The order is load-bearing: fuse, then decide access, then budget. Access
// control applied after budgeting leaks the existence of what was dropped,
// and a count computed before filtering is a side channel of its own.
func (s *Service) Recall(ctx context.Context, req Request) (*Result, error) {
	started := time.Now()
	if req.Now.IsZero() {
		req.Now = started
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Depth <= 0 {
		req.Depth = DefaultCandidateDepth
	}
	if req.Budget <= 0 {
		req.Budget = DefaultTokenBudget
	}

	modelID, modelName, _, err := s.store.PresentModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve embedding model: %w", err)
	}
	// Reads filter on the model. A per-row model column that reads ignore is
	// worthless, and a mismatch between the loaded model and the indexed one
	// returns nonsense rather than an error.
	if modelName != s.embed.ModelName() {
		return nil, fmt.Errorf(
			"embedding model mismatch: database holds %q, this binary loaded %q",
			modelName, s.embed.ModelName())
	}

	t := time.Now()
	vecs, err := s.embed.Embed(ctx, []string{req.Query})
	if err != nil {
		return nil, fmt.Errorf("embed query: %w", err)
	}
	if len(vecs) != 1 {
		return nil, fmt.Errorf("embed query: got %d vectors, want 1", len(vecs))
	}
	timings := Timings{Embed: time.Since(t)}

	t = time.Now()
	dense, err := s.store.DenseSearch(ctx, modelID, vecs[0], req.Depth)
	if err != nil {
		return nil, fmt.Errorf("dense search: %w", err)
	}
	timings.Dense = time.Since(t)

	t = time.Now()
	lexical, err := s.store.LexicalSearch(ctx, req.Query, req.Depth)
	if err != nil {
		return nil, fmt.Errorf("lexical search: %w", err)
	}
	timings.Lexical = time.Since(t)

	arms := map[Arm][]Ranked{
		ArmDense:   toRanked(dense),
		ArmLexical: toRanked(lexical),
	}
	if !EqualDepth(arms, req.Depth) {
		return nil, fmt.Errorf("retrieval arms exceeded depth %d", req.Depth)
	}
	fused := FuseRRF(arms)

	t = time.Now()
	claims, err := s.hydrate(ctx, fused)
	if err != nil {
		return nil, err
	}
	timings.Load = time.Since(t)

	// L5, in Go, on loaded rows. Never a SQL predicate.
	authorized := acl.Filter(claims, req.Principal, req.Now)

	byID := make(map[string]claim.Claim, len(authorized))
	for _, c := range authorized {
		byID[c.ID] = c
	}

	res := &Result{
		AsOf:        req.Now,
		Candidates:  len(fused),
		Authorized:  len(authorized),
		TokenBudget: req.Budget,
		Timings:     timings,
	}

	for _, f := range fused {
		c, ok := byID[f.ID]
		if !ok {
			continue // denied, or superseded between the two queries
		}
		if len(res.Claims) >= req.Limit {
			res.OmittedForBudget++
			continue
		}
		n := s.count(c)
		if res.TokensUsed+n > req.Budget {
			// Drop the whole claim. Truncating its text would leave a claim
			// whose evidence no longer supports it.
			res.OmittedForBudget++
			continue
		}
		res.TokensUsed += n
		res.Claims = append(res.Claims, Scored{
			Claim:         c,
			Score:         f.Score,
			Contributions: f.Contributions,
			Tokens:        n,
		})
	}

	res.DominantArm, res.DominantShare = SingleArmShare(fused, req.Limit)
	res.StalenessSeconds = stalest(res.Claims, req.Now)
	res.Timings.Total = time.Since(started)
	return res, nil
}

func (s *Service) count(c claim.Claim) int {
	if s.tokens == nil {
		return 0
	}
	n := s.tokens(c.Statement)
	for _, e := range c.Evidence {
		n += s.tokens(e.ExtractedText)
	}
	return n
}

// hydrate loads the claims behind a fused candidate list, along with their
// evidence and both sets of grants. Evidence is attached before any access
// decision, because the decision needs the evidence ACLs to intersect.
func (s *Service) hydrate(ctx context.Context, fused []Fused) ([]claim.Claim, error) {
	if len(fused) == 0 {
		return nil, nil
	}
	ids := make([]string, len(fused))
	for i, f := range fused {
		ids[i] = f.ID
	}

	claims, err := s.store.LoadClaims(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load claims: %w", err)
	}
	evidence, err := s.store.LoadEvidence(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load evidence: %w", err)
	}
	claimACLs, err := s.store.LoadClaimACLs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("load claim grants: %w", err)
	}

	var evidenceIDs []string
	for _, es := range evidence {
		for _, e := range es {
			evidenceIDs = append(evidenceIDs, e.ID)
		}
	}
	evidenceACLs, err := s.store.LoadEvidenceACLs(ctx, evidenceIDs)
	if err != nil {
		return nil, fmt.Errorf("load evidence grants: %w", err)
	}

	for i := range claims {
		claims[i].ACL = claimACLs[claims[i].ID]
		es := evidence[claims[i].ID]
		for j := range es {
			es[j].ACL = evidenceACLs[es[j].ID]
		}
		claims[i].Evidence = es
	}
	return claims, nil
}

func toRanked(hits []pg.Hit) []Ranked {
	out := make([]Ranked, len(hits))
	for i, h := range hits {
		out[i] = Ranked{ID: h.ClaimID, Rank: h.Rank, Raw: h.Raw}
	}
	return out
}

func stalest(claims []Scored, now time.Time) float64 {
	var oldest time.Time
	for _, s := range claims {
		if oldest.IsZero() || s.Claim.Recorded.From.Before(oldest) {
			oldest = s.Claim.Recorded.From
		}
	}
	if oldest.IsZero() {
		return 0
	}
	return now.Sub(oldest).Seconds()
}
