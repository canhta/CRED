package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/canhta/cred/internal/recall"
)

// RecallQuery is the query string of GET /api/recall.
type RecallQuery struct {
	Q      string `json:"q" form:"q"`
	Limit  int    `json:"limit" form:"limit"`
	Depth  int    `json:"depth" form:"depth"`
	Budget int    `json:"budget" form:"budget"`
}

// Contribution is what one retrieval arm added to a claim's fused rank: the
// arm's name, the rank it gave the claim, its raw score, and the reciprocal-rank
// points that rank contributed. It is the per-arm reason a result placed where
// it did.
type Contribution struct {
	Arm   string  `json:"arm"`
	Rank  int     `json:"rank"`
	Raw   float64 `json:"raw"`
	Score float64 `json:"score"`
}

// RecalledClaim is one recall result carrying why it ranked.
type RecalledClaim struct {
	ID            string         `json:"id"`
	Statement     string         `json:"statement"`
	Kind          string         `json:"kind"`
	Scope         Scope          `json:"scope"`
	Status        string         `json:"status"`
	Source        *Source        `json:"source"`
	Score         float64        `json:"score"`
	Tokens        int            `json:"tokens"`
	Contributions []Contribution `json:"contributions"`
}

// RecallTimings is where a recall's latency went, per arm, in milliseconds.
type RecallTimings struct {
	EmbedMs   float64 `json:"embed_ms"`
	DenseMs   float64 `json:"dense_ms"`
	LexicalMs float64 `json:"lexical_ms"`
	LoadMs    float64 `json:"load_ms"`
	TotalMs   float64 `json:"total_ms"`
}

// RecallResponse is the body of GET /api/recall: the ranked claims plus the
// retrieval accounting that explains them. Candidates is what fusion saw,
// Authorized is what survived access control, and OmittedForBudget is what the
// token ceiling dropped — reported so a short result is never a silent one.
type RecallResponse struct {
	Query            string          `json:"query"`
	Claims           []RecalledClaim `json:"claims"`
	Candidates       int             `json:"candidates"`
	Authorized       int             `json:"authorized"`
	OmittedForBudget int             `json:"omitted_for_budget"`
	TokensUsed       int             `json:"tokens_used"`
	TokenBudget      int             `json:"token_budget"`
	DominantArm      string          `json:"dominant_arm"`
	DominantShare    float64         `json:"dominant_share"`
	AsOf             string          `json:"as_of"`
	StalenessSeconds float64         `json:"staleness_seconds"`
	Timings          RecallTimings   `json:"timings"`
}

// recall runs a query through the hybrid retrieval pipeline and returns each
// result with the per-arm contributions that placed it. Access control is the
// pipeline's own: the store returns candidates and internal/acl filters them to
// what this principal may read, so an unauthorized claim never reaches the wire.
func (s *server) recall(c *gin.Context) {
	if s.embedder == nil {
		c.JSON(http.StatusServiceUnavailable,
			ErrorResponse{Error: "recall is unavailable: the embedding model is not loaded"})
		return
	}

	var q RecallQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid query parameters"})
		return
	}
	if strings.TrimSpace(q.Q) == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "q is required"})
		return
	}

	limit := q.Limit
	switch {
	case limit <= 0:
		limit = 5
	case limit > maxLimit:
		limit = maxLimit
	}
	depth := q.Depth
	if depth <= 0 {
		depth = recall.DefaultCandidateDepth
	}
	budget := q.Budget
	if budget <= 0 {
		budget = recall.DefaultTokenBudget
	}

	now := time.Now().UTC()
	svc := recall.New(s.store, s.embedder, s.count).WithLimits(s.cfg.Limits, s.store)
	res, err := svc.Recall(c.Request.Context(), recall.Request{
		Query:     q.Q,
		Principal: principalFrom(c),
		Limit:     limit,
		Depth:     depth,
		Budget:    budget,
		Now:       now,
	})
	if err != nil {
		if _, ok := recall.AsBudgetError(err); ok {
			c.JSON(http.StatusTooManyRequests,
				ErrorResponse{Error: "recall rate limit exceeded for this principal"})
			return
		}
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, recallResponse(q.Q, res, now))
}

func recallResponse(query string, res *recall.Result, now time.Time) RecallResponse {
	claims := make([]RecalledClaim, 0, len(res.Claims))
	for i := range res.Claims {
		sc := res.Claims[i]
		rc := RecalledClaim{
			ID:            sc.Claim.ID,
			Statement:     sc.Claim.Statement,
			Kind:          string(sc.Claim.Kind),
			Scope:         Scope{Kind: string(sc.Claim.Scope.Kind), Value: sc.Claim.Scope.Value},
			Status:        statusOf(sc.Claim, now),
			Score:         sc.Score,
			Tokens:        sc.Tokens,
			Contributions: make([]Contribution, 0, len(sc.Contributions)),
		}
		if len(sc.Claim.Evidence) > 0 {
			e := sc.Claim.Evidence[0]
			rc.Source = &Source{
				Kind:       string(e.Kind),
				Repo:       e.Repo,
				Path:       e.Path,
				LineStart:  e.LineStart,
				LineEnd:    e.LineEnd,
				SymbolPath: e.AnchorSymbolPath,
			}
		}
		for _, con := range sc.Contributions {
			rc.Contributions = append(rc.Contributions, Contribution{
				Arm:   string(con.Arm),
				Rank:  con.Rank,
				Raw:   con.Raw,
				Score: con.Score,
			})
		}
		claims = append(claims, rc)
	}

	return RecallResponse{
		Query:            query,
		Claims:           claims,
		Candidates:       res.Candidates,
		Authorized:       res.Authorized,
		OmittedForBudget: res.OmittedForBudget,
		TokensUsed:       res.TokensUsed,
		TokenBudget:      res.TokenBudget,
		DominantArm:      string(res.DominantArm),
		DominantShare:    res.DominantShare,
		AsOf:             rfc3339(res.AsOf),
		StalenessSeconds: res.StalenessSeconds,
		Timings: RecallTimings{
			EmbedMs:   ms(res.Timings.Embed),
			DenseMs:   ms(res.Timings.Dense),
			LexicalMs: ms(res.Timings.Lexical),
			LoadMs:    ms(res.Timings.Load),
			TotalMs:   ms(res.Timings.Total),
		},
	}
}

func ms(d time.Duration) float64 {
	return float64(d.Microseconds()) / 1000.0
}
