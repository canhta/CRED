// Package api is the CRED web console's HTTP surface: plain Gin handlers over
// typed request and response structs. It is a projection over the store, not a
// second model — the store returns rows and internal/acl decides what a
// principal may see, exactly as the CLI does. An endpoint that returned a claim
// its caller may not read would be the access-control failure, now over HTTP.
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/canhta/cred/internal/claim"
	"github.com/canhta/cred/internal/config"
	"github.com/canhta/cred/internal/mcpsrv"
	"github.com/canhta/cred/internal/recall"
	"github.com/canhta/cred/internal/store/pg"
)

// New builds the console's API handler over an existing store. The embedder and
// token counter power the recall inspector; both may be nil, in which case that
// one endpoint reports itself unavailable and the rest of the console is
// unaffected. The caller wraps the handler so /api routes reach this engine and
// everything else serves the SPA.
func New(st *pg.Store, emb recall.Embedder, count recall.TokenCounter,
	cfg config.Config, log *slog.Logger,
) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(requestLogger(log), gin.Recovery(), authenticate(cfg, st))

	s := &server{store: st, embedder: emb, count: count, cfg: cfg, log: log}
	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		api.GET("/claims", s.listClaims)
		api.GET("/claims/:id", s.getClaim)
		api.GET("/recall", s.recall)
		api.GET("/usage", s.usage)
		api.POST("/auth/register", s.register)
		api.POST("/auth/login", s.login)
		api.POST("/auth/logout", s.logout)

		admin := api.Group("")
		admin.Use(requireAdmin())
		admin.GET("/usage/org", s.usageOrg)
	}
	return r
}

// server holds the dependencies each handler needs.
type server struct {
	store    *pg.Store
	embedder recall.Embedder
	count    recall.TokenCounter
	cfg      config.Config
	log      *slog.Logger
}

// principalKey is the context key the auth middleware stores the resolved
// principal under. A private type keeps it from colliding with any other
// package's context values.
type principalKey struct{}

// roleKey is the context key for the resolved principal's role, attached by
// authenticate() on every path -- session and header/default alike -- and
// read by roleFrom.
type roleKey struct{}

// principalFrom returns the principal the auth middleware resolved. Handlers
// read the caller's identity through here, never from the request header.
func principalFrom(c *gin.Context) claim.PrincipalID {
	if p, ok := c.Request.Context().Value(principalKey{}).(claim.PrincipalID); ok {
		return p
	}
	return ""
}

// roleFrom returns the resolved principal's role ("admin" or "member"), or
// "" when none is known -- a header/default-principal caller with no
// console account resolves the same way as an unauthenticated one. Empty
// never passes requireAdmin, so "role unknown" and "role member" fail
// identically.
func roleFrom(c *gin.Context) string {
	if r, ok := c.Request.Context().Value(roleKey{}).(string); ok {
		return r
	}
	return ""
}

// authenticate resolves the principal for every request and, when a token is
// configured, gates access on it. A session cookie is checked first and, when
// valid, is authoritative -- its principal came from a verified login, not a
// client-supplied header, so it is never overridden by one. Replacing this
// with OIDC/SSO later touches no handler, because handlers read the
// principal the middleware put on the context, not the header or the cookie.
func authenticate(cfg config.Config, store *pg.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		if cfg.WebToken != "" {
			const prefix = "Bearer "
			auth := c.GetHeader("Authorization")
			if len(auth) <= len(prefix) || auth[:len(prefix)] != prefix ||
				auth[len(prefix):] != cfg.WebToken {
				c.AbortWithStatusJSON(http.StatusUnauthorized,
					ErrorResponse{Error: "unauthorized"})
				return
			}
		}

		// An invalid or expired cookie falls through to the header/config-
		// default path below rather than failing the request outright -- a
		// stale cookie from a rotated session must not lock out a caller
		// that also has a valid bearer token configured.
		if cookie, err := c.Cookie(sessionCookieName); err == nil {
			principal, role, serr := store.SessionPrincipal(c.Request.Context(), hashToken(cookie), time.Now().UTC())
			if serr == nil {
				ctx := context.WithValue(c.Request.Context(), principalKey{}, principal)
				ctx = context.WithValue(ctx, roleKey{}, role)
				c.Request = c.Request.WithContext(ctx)
				c.Next()
				return
			}
		}

		// The header names who the request acts as; absent, the single-user
		// self-host identity from config stands in.
		principal := claim.PrincipalID(cfg.Principal)
		if h := c.GetHeader("X-CRED-Principal"); h != "" {
			principal = claim.PrincipalID(h)
		}

		// A header/default-authenticated caller still gets its real console
		// role, not an assumed one -- an admin using the CLI or a bearer
		// token is still an admin. A lookup error is treated the same as no
		// role found, matching the tolerant fall-through the session branch
		// above already uses for its own error case.
		role, _ := store.RoleForPrincipal(c.Request.Context(), principal)

		ctx := context.WithValue(c.Request.Context(), principalKey{}, principal)
		ctx = context.WithValue(ctx, roleKey{}, role)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// requireAdmin aborts with 403 unless the resolved principal's role is
// "admin". Applied to a route group, never inlined in a handler -- a
// centralized check every admin route inherits, not a conditional a future
// edit could silently drop.
func requireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if roleFrom(c) != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: "admin role required"})
			return
		}
		c.Next()
	}
}

// requestLogger records each request to the slog logger. It logs identifiers,
// the status, and the duration — never claim or evidence content.
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration", time.Since(start).String())
	}
}

func (s *server) health(c *gin.Context) {
	count, err := s.store.UserCount(c.Request.Context())
	if err != nil {
		s.fail(c, err)
		return
	}
	c.JSON(http.StatusOK, HealthResponse{
		Status:           "ok",
		Version:          mcpsrv.Version,
		Principal:        string(principalFrom(c)),
		Role:             roleFrom(c),
		RegistrationOpen: count == 0,
	})
}
