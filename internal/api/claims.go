package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/canhta/cred/internal/acl"
	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/store/pg"
)

const (
	defaultLimit = 50
	maxLimit     = 200
)

func (s *server) listClaims(c *gin.Context) {
	var q ClaimListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid query parameters"})
		return
	}

	status := pg.ClaimStatus(q.Status)
	switch status {
	case "":
		status = pg.ClaimLive
	case pg.ClaimLive, pg.ClaimExpired, pg.ClaimAny:
	default:
		c.JSON(http.StatusBadRequest,
			ErrorResponse{Error: "status must be live, expired, or all"})
		return
	}

	limit := q.Limit
	switch {
	case limit <= 0:
		limit = defaultLimit
	case limit > maxLimit:
		limit = maxLimit
	}
	offset := q.Offset
	if offset < 0 {
		offset = 0
	}

	now := time.Now().UTC()
	ctx := c.Request.Context()
	claims, err := s.store.ListClaims(ctx, pg.ClaimQuery{
		Status:     status,
		ScopeKind:  q.ScopeKind,
		ScopeValue: q.ScopeValue,
		Now:        now,
	})
	if err != nil {
		s.fail(c, err)
		return
	}
	if err := s.hydrate(ctx, claims); err != nil {
		s.fail(c, err)
		return
	}

	// Access control is decided here, in Go, before paginating — never as a SQL
	// predicate and never after the page is cut, either of which would leak the
	// existence of claims the caller may not read.
	visible := acl.Filter(claims, principalFrom(c), now)

	page := visible
	if offset >= len(page) {
		page = nil
	} else {
		page = page[offset:]
	}
	if len(page) > limit {
		page = page[:limit]
	}

	items := make([]ClaimListItem, 0, len(page))
	for i := range page {
		items = append(items, listItem(page[i], now))
	}
	c.JSON(http.StatusOK, ClaimListResponse{
		Claims: items,
		Limit:  limit,
		Offset: offset,
		Count:  len(items),
	})
}

func (s *server) getClaim(c *gin.Context) {
	id := c.Param("id")
	now := time.Now().UTC()
	ctx := c.Request.Context()

	claims, err := s.store.ListClaims(ctx, pg.ClaimQuery{ID: id, Now: now})
	if err != nil {
		s.fail(c, err)
		return
	}
	if err := s.hydrate(ctx, claims); err != nil {
		s.fail(c, err)
		return
	}

	// A claim the caller may not read is reported as absent: an authorization
	// error is an existence oracle, so unauthorized and nonexistent must be the
	// same 404.
	if len(claims) == 0 || !acl.CanRead(claims[0], principalFrom(c), now) {
		c.JSON(http.StatusNotFound, ErrorResponse{Error: "claim not found"})
		return
	}
	c.JSON(http.StatusOK, claimDetail(claims[0], now))
}

// hydrate attaches each claim's evidence and the grants on the claim and its
// evidence, which is what internal/acl needs to decide visibility.
func (s *server) hydrate(ctx context.Context, claims []claim.Claim) error {
	if len(claims) == 0 {
		return nil
	}
	ids := make([]string, len(claims))
	for i, c := range claims {
		ids[i] = c.ID
	}

	evByClaim, err := s.store.LoadEvidence(ctx, ids)
	if err != nil {
		return err
	}
	claimACLs, err := s.store.LoadClaimACLs(ctx, ids)
	if err != nil {
		return err
	}

	var evidenceIDs []string
	for _, evs := range evByClaim {
		for _, e := range evs {
			evidenceIDs = append(evidenceIDs, e.ID)
		}
	}
	evidenceACLs, err := s.store.LoadEvidenceACLs(ctx, evidenceIDs)
	if err != nil {
		return err
	}

	for i := range claims {
		claims[i].ACL = claimACLs[claims[i].ID]
		evs := evByClaim[claims[i].ID]
		for j := range evs {
			evs[j].ACL = evidenceACLs[evs[j].ID]
		}
		claims[i].Evidence = evs
	}
	return nil
}

func (s *server) fail(c *gin.Context, err error) {
	s.log.Error("api", "path", c.Request.URL.Path, "err", err.Error())
	c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "internal error"})
}

func listItem(c claim.Claim, now time.Time) ClaimListItem {
	item := ClaimListItem{
		ID:            c.ID,
		Statement:     c.Statement,
		Kind:          string(c.Kind),
		Scope:         Scope{Kind: string(c.Scope.Kind), Value: c.Scope.Value},
		Status:        statusOf(c, now),
		ContributedBy: string(c.ContributedBy),
		RecordedAt:    rfc3339(c.Recorded.From),
		ValidFrom:     rfc3339(c.Valid.From),
		ValidUntil:    rfc3339(c.Valid.Until),
		SupersededAt:  rfc3339(c.Recorded.Until),
	}
	if len(c.Evidence) > 0 {
		e := c.Evidence[0]
		item.Source = &Source{
			Kind:       string(e.Kind),
			Repo:       e.Repo,
			Path:       e.Path,
			LineStart:  e.LineStart,
			LineEnd:    e.LineEnd,
			SymbolPath: e.AnchorSymbolPath,
		}
	}
	return item
}

func claimDetail(c claim.Claim, now time.Time) ClaimDetail {
	status := statusOf(c, now)
	d := ClaimDetail{
		ID:            c.ID,
		Statement:     c.Statement,
		Kind:          string(c.Kind),
		Scope:         Scope{Kind: string(c.Scope.Kind), Value: c.Scope.Value},
		Status:        status,
		Confidence:    c.Confidence,
		ContributedBy: string(c.ContributedBy),
		SourceRepo:    c.SourceRepo,
		RecordedAt:    rfc3339(c.Recorded.From),
		ValidFrom:     rfc3339(c.Valid.From),
		ValidUntil:    rfc3339(c.Valid.Until),
		SupersededAt:  rfc3339(c.Recorded.Until),
		Evidence:      make([]EvidenceItem, 0, len(c.Evidence)),
	}
	if status == string(pg.ClaimExpired) {
		d.ExpiredReason = c.SupersedeReason
	}
	for _, e := range c.Evidence {
		anchor := "unanchored"
		if e.AnchorSymbolPath != "" {
			anchor = "anchored"
		}
		d.Evidence = append(d.Evidence, EvidenceItem{
			ID:         e.ID,
			Kind:       string(e.Kind),
			Repo:       e.Repo,
			Path:       e.Path,
			LineStart:  e.LineStart,
			LineEnd:    e.LineEnd,
			SymbolPath: e.AnchorSymbolPath,
			Anchor:     anchor,
		})
	}
	return d
}

// statusOf reports whether a claim reads as live or expired: superseded in
// transaction time, or past the end of its valid interval, is expired.
func statusOf(c claim.Claim, now time.Time) string {
	if !c.Recorded.Until.IsZero() {
		return string(pg.ClaimExpired)
	}
	if !c.Valid.Until.IsZero() && !now.Before(c.Valid.Until) {
		return string(pg.ClaimExpired)
	}
	return string(pg.ClaimLive)
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}
