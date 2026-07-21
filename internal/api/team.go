package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const inviteLifetime = 7 * 24 * time.Hour

// createInvite generates a single-use, email-targeted invite token. The raw
// token is visible exactly once, in this response -- only its hash is ever
// stored, the same convention session tokens already follow.
func (s *server) createInvite(c *gin.Context) {
	var req CreateInviteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	raw, hash, err := newOpaqueToken()
	if err != nil {
		s.fail(c, err)
		return
	}

	expiresAt := time.Now().UTC().Add(inviteLifetime)
	_, err = s.store.CreateInvite(c.Request.Context(), req.Email, req.Role, hash, principalFrom(c), expiresAt)
	if err != nil {
		s.fail(c, err)
		return
	}

	c.JSON(http.StatusOK, CreateInviteResponse{
		Email:     req.Email,
		Role:      req.Role,
		ExpiresAt: rfc3339(expiresAt),
		Token:     raw,
	})
}

// listInvites reports every invite still redeemable -- used, expired, and
// revoked invites aren't surfaced this pass.
func (s *server) listInvites(c *gin.Context) {
	invites, err := s.store.PendingInvites(c.Request.Context(), time.Now().UTC())
	if err != nil {
		s.fail(c, err)
		return
	}

	out := make([]Invite, 0, len(invites))
	for _, inv := range invites {
		out = append(out, Invite{
			ID:        inv.ID,
			Email:     inv.Email,
			Role:      inv.Role,
			CreatedAt: rfc3339(inv.CreatedAt),
			ExpiresAt: rfc3339(inv.ExpiresAt),
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *server) revokeInvite(c *gin.Context) {
	if err := s.store.RevokeInvite(c.Request.Context(), c.Param("id"), time.Now().UTC()); err != nil {
		s.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *server) teamMembers(c *gin.Context) {
	members, err := s.store.TeamMembers(c.Request.Context())
	if err != nil {
		s.fail(c, err)
		return
	}

	out := make([]TeamMember, 0, len(members))
	for _, m := range members {
		out = append(out, TeamMember{
			PrincipalID: m.PrincipalID,
			Email:       m.Email,
			Role:        m.Role,
			CreatedAt:   rfc3339(m.CreatedAt),
		})
	}
	c.JSON(http.StatusOK, out)
}
