package api

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/limit"
	"github.com/canhta/cred/internal/store/pg"
)

const (
	sessionCookieName = "cred_session"
	sessionLifetime   = 30 * 24 * time.Hour
)

// dummyHash is compared against when an email does not exist, so a login
// attempt takes the same time whether or not the email is registered --
// closing the timing side-channel the identical 401 response already closes
// at the response-body level.
var dummyHash []byte

func init() {
	h, err := bcrypt.GenerateFromPassword([]byte("cred-dummy-password-for-timing"), bcrypt.DefaultCost)
	if err != nil {
		panic(fmt.Sprintf("api: dummy bcrypt hash: %v", err))
	}
	dummyHash = h
}

// hashToken returns the SHA-256 hex digest of a raw session token. Sessions
// store only this -- the raw token is a bearer credential and is never
// written to the database.
func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

// newOpaqueToken generates a fresh 32-byte random token and its hash --
// shared by sessions and invites, since a bearer token's shape (opaque,
// hashed at rest, raw value never persisted) doesn't differ between them.
func newOpaqueToken() (raw string, hash string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	raw = base64.RawURLEncoding.EncodeToString(buf)
	return raw, hashToken(raw), nil
}

// setSessionCookie writes the session cookie. Secure is always true:
// browsers treat localhost/127.0.0.1 as a secure context even over plain
// HTTP, and cred web always serves the API and the SPA from the same origin,
// so this never breaks local dev.
func setSessionCookie(c *gin.Context, token string, expiresAt time.Time) {
	c.SetCookieData(&http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(c *gin.Context) {
	c.SetCookieData(&http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// startSession creates a session for principal, sets the cookie, and writes
// the response. Shared by register and login so the two paths cannot drift.
func (s *server) startSession(c *gin.Context, principal claim.PrincipalID, role string) {
	token, hash, err := newOpaqueToken()
	if err != nil {
		s.fail(c, err)
		return
	}
	expiresAt := time.Now().UTC().Add(sessionLifetime)
	if err := s.store.CreateSession(c.Request.Context(), principal, hash, expiresAt); err != nil {
		s.fail(c, err)
		return
	}
	setSessionCookie(c, token, expiresAt)
	c.JSON(http.StatusOK, AuthResponse{Principal: string(principal), Role: role})
}

// register redeems an invite when one is present; otherwise it creates the
// first account as admin and closes registration for every account after
// it. Once an invite exists, that's the only way a second account can
// exist -- until this pass, closed registration was the only enforcement.
func (s *server) register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	ctx := c.Request.Context()

	if req.Invite != "" {
		s.registerWithInvite(c, req)
		return
	}

	count, err := s.store.UserCount(ctx)
	if err != nil {
		s.fail(c, err)
		return
	}
	if count > 0 {
		c.JSON(http.StatusForbidden, ErrorResponse{Error: "registration is closed"})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.fail(c, err)
		return
	}

	// The first account is always admin, regardless of what the request
	// carried -- a client-supplied role on the only unauthenticated write
	// path this feature adds is exactly the privilege-escalation seam this
	// feature exists to close.
	principal, err := s.store.CreateUser(ctx, req.Email, string(hash), "admin")
	if err != nil {
		if errors.Is(err, pg.ErrEmailTaken) {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "email already registered"})
			return
		}
		if errors.Is(err, pg.ErrBootstrapExists) {
			// Another registration won the race to be first: from this
			// client's perspective that is indistinguishable from
			// registration having already been closed.
			c.JSON(http.StatusForbidden, ErrorResponse{Error: "registration is closed"})
			return
		}
		s.fail(c, err)
		return
	}

	s.startSession(c, principal, "admin")
}

// registerWithInvite hashes the password before touching the invite, the
// same ordering login uses for its own slow bcrypt step -- so the one
// database round trip that can fail (ClaimInvite) happens after the
// expensive, deterministic work, not before it.
func (s *server) registerWithInvite(c *gin.Context, req RegisterRequest) {
	ctx := c.Request.Context()

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.fail(c, err)
		return
	}

	role, invitedBy, err := s.store.ClaimInvite(ctx, hashToken(req.Invite), req.Email, time.Now().UTC())
	if err != nil {
		if errors.Is(err, pg.ErrNotFound) {
			// Not found, expired, used, revoked, or a mismatched email are
			// all the same response -- distinguishing them would let a
			// caller learn which one it was, the same existence-oracle
			// failure class this handler's own bootstrap path already
			// avoids for "registration is closed".
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid or expired invite"})
			return
		}
		s.fail(c, err)
		return
	}

	principal, err := s.store.CreateInvitedUser(ctx, req.Email, string(hash), role, invitedBy)
	if err != nil {
		if errors.Is(err, pg.ErrEmailTaken) {
			c.JSON(http.StatusConflict, ErrorResponse{Error: "email already registered"})
			return
		}
		s.fail(c, err)
		return
	}

	s.startSession(c, principal, role)
}

// login rate-limits by email before touching bcrypt: a bcrypt compare is
// deliberately slow, and running it before the rate check would let an
// attacker burn server CPU right up to the limit on every window.
func (s *server) login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "invalid request"})
		return
	}

	ctx := c.Request.Context()
	now := time.Now().UTC()
	lc := s.cfg.Limits

	failed, err := s.store.FailedLoginsInWindow(ctx, req.Email, limit.WindowStart(now, lc.LoginWindow))
	if err != nil {
		s.fail(c, err)
		return
	}
	if !limit.LoginAttempts(failed, lc).Allowed {
		c.JSON(http.StatusTooManyRequests,
			ErrorResponse{Error: "too many login attempts, try again later"})
		return
	}

	principal, hash, role, err := s.store.CredentialsByEmail(ctx, req.Email)
	if err != nil && !errors.Is(err, pg.ErrNotFound) {
		s.fail(c, err)
		return
	}

	hashToCompare := dummyHash
	if err == nil {
		hashToCompare = []byte(hash)
	}
	match := bcrypt.CompareHashAndPassword(hashToCompare, []byte(req.Password)) == nil
	valid := err == nil && match

	if recErr := s.store.RecordLoginAttempt(ctx, req.Email, valid, now); recErr != nil {
		s.fail(c, recErr)
		return
	}

	// A missing email and a wrong password fail identically -- an email-
	// existence oracle is the same failure class getClaim's 404 already
	// avoids for authorization.
	if !valid {
		c.JSON(http.StatusUnauthorized, ErrorResponse{Error: "invalid email or password"})
		return
	}

	s.startSession(c, principal, role)
}

func (s *server) logout(c *gin.Context) {
	if cookie, err := c.Cookie(sessionCookieName); err == nil {
		_ = s.store.DeleteSession(c.Request.Context(), hashToken(cookie))
	}
	clearSessionCookie(c)
	c.Status(http.StatusNoContent)
}
