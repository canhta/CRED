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
	r.Use(requestLogger(log), gin.Recovery(), authenticate(cfg))

	s := &server{store: st, embedder: emb, count: count, cfg: cfg, log: log}
	api := r.Group("/api")
	{
		api.GET("/health", s.health)
		api.GET("/claims", s.listClaims)
		api.GET("/claims/:id", s.getClaim)
		api.GET("/recall", s.recall)
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

// principalFrom returns the principal the auth middleware resolved. Handlers
// read the caller's identity through here, never from the request header.
func principalFrom(c *gin.Context) claim.PrincipalID {
	if p, ok := c.Request.Context().Value(principalKey{}).(claim.PrincipalID); ok {
		return p
	}
	return ""
}

// authenticate resolves the principal for every request and, when a token is
// configured, gates access on it. This is the whole authentication seam:
// replacing it with OIDC/SSO later touches no handler, because handlers read
// the principal the middleware put on the context, not the header.
func authenticate(cfg config.Config) gin.HandlerFunc {
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

		// The header names who the request acts as; absent, the single-user
		// self-host identity from config stands in.
		principal := claim.PrincipalID(cfg.Principal)
		if h := c.GetHeader("X-CRED-Principal"); h != "" {
			principal = claim.PrincipalID(h)
		}

		ctx := context.WithValue(c.Request.Context(), principalKey{}, principal)
		c.Request = c.Request.WithContext(ctx)
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
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "ok",
		Version:   mcpsrv.Version,
		Principal: string(principalFrom(c)),
	})
}
