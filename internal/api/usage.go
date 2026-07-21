package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/canhta/cred/internal/limit"
	"github.com/canhta/cred/internal/store/pg"
)

// usage reports the calling principal's limit headroom, its denied-
// contribution count, and the per-scope cost/growth report. Every number
// comes from the same store counts and the same internal/limit decisions the
// enforcement path uses, so the console can never show a number that
// disagrees with what a write or a recall was actually allowed to do.
func (s *server) usage(c *gin.Context) {
	var q UsageQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid query parameters"})
		return
	}
	topN := q.Scopes
	if topN <= 0 {
		topN = 10
	}

	ctx := c.Request.Context()
	principal := principalFrom(c)
	now := time.Now().UTC()
	lc := s.cfg.Limits

	state, err := s.store.PrincipalWindowState(ctx, principal,
		limit.WindowStart(now, lc.ContributionWindow),
		limit.WindowStart(now, lc.CostWindow),
		limit.WindowStart(now, lc.RecallWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	denied, err := s.store.DeniedInWindow(ctx, principal, limit.WindowStart(now, lc.ContributionWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	costs, err := s.store.UsageByScope(ctx, limit.WindowStart(now, lc.CostWindow))
	if err != nil {
		s.fail(c, err)
		return
	}

	sizes, err := s.store.ScopeSizes(ctx, topN)
	if err != nil {
		s.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, usageResponse(string(principal), lc, state, denied, costs, sizes))
}

func usageResponse(principal string, lc limit.Config, state pg.ContributionState,
	denied int, costs []pg.ScopeCost, sizes []pg.ScopeSize,
) UsageResponse {
	contribution := limit.Contribution(state.Contributions, lc)
	cost := limit.Cost(state.InferenceCall, state.InputTokens, lc)
	recall := limit.RecallRate(state.Recalls, lc)

	costByScope := make([]ScopeCost, 0, len(costs))
	for _, c := range costs {
		costByScope = append(costByScope, ScopeCost{
			Scope:        Scope{Kind: string(c.Scope.Kind), Value: c.Scope.Value},
			Calls:        c.Calls,
			InputTokens:  c.InputTokens,
			OutputTokens: c.OutputTokens,
		})
	}

	scopeGrowth := make([]ScopeGrowth, 0, len(sizes))
	for _, sz := range sizes {
		scopeGrowth = append(scopeGrowth, ScopeGrowth{
			Scope:     Scope{Kind: string(sz.Scope.Kind), Value: sz.Scope.Value},
			Live:      sz.Live,
			Ceiling:   lc.ScopeClaimCeiling,
			NextPrune: limit.PruneTarget(sz.Live, lc),
		})
	}

	return UsageResponse{
		Principal: principal,
		Contribution: LimitStatus{
			Window: lc.ContributionWindow.String(), Used: state.Contributions,
			Ceiling: lc.ContributionQuota, Remaining: contribution.Remaining,
			Allowed: contribution.Allowed, Reason: string(contribution.Reason),
		},
		Cost: LimitStatus{
			Window: lc.CostWindow.String(), Used: state.InferenceCall,
			Ceiling: lc.MaxInferenceCalls, Remaining: cost.Remaining,
			Allowed: cost.Allowed, Reason: string(cost.Reason),
		},
		InputTokensUsed:    state.InputTokens,
		InputTokensCeiling: lc.MaxInputTokens,
		Recall: LimitStatus{
			Window: lc.RecallWindow.String(), Used: state.Recalls,
			Ceiling: lc.RecallRate, Remaining: recall.Remaining,
			Allowed: recall.Allowed, Reason: string(recall.Reason),
		},
		DeniedWindow: lc.ContributionWindow.String(),
		Denied:       denied,
		CostByScope:  costByScope,
		ScopeGrowth:  scopeGrowth,
	}
}
