package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/canhta/cred/internal/api"
	"github.com/canhta/cred/internal/config"
)

// WebAssets is the built SPA, set from package main before Run. It is non-nil
// only in a binary built with `-tags embed`; nil makes `cred web` fall back to
// serving web/dist from disk, or a stub page when that is absent too.
var WebAssets fs.FS

func runWeb(ctx context.Context, args []string, cfg config.Config,
	log *slog.Logger, stderr io.Writer,
) error {
	fs := flag.NewFlagSet("web", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	addr := fs.String("addr", cfg.WebAddr, "listen address")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %w", ErrUsage, err)
	}

	st, err := openStore(ctx, cfg)
	if err != nil {
		return err
	}
	defer st.Close()

	// One engine serves both surfaces: /api routes are registered by api.New,
	// and the SPA (with its history fallback) is the NoRoute handler on the same
	// engine, so a single server answers the API and the console alike.
	handler := api.New(st, cfg, log)
	engine, ok := handler.(*gin.Engine)
	if !ok {
		return fmt.Errorf("web: api handler is not a *gin.Engine")
	}
	engine.NoRoute(spaHandler(resolveSite(stderr)))

	claims, evidence, err := st.Counts(ctx)
	if err != nil {
		return err
	}
	gate := "open"
	if cfg.WebToken != "" {
		gate = "token-gated"
	}
	fmt.Fprintf(stderr, "cred web  %s  %d claims, %d evidence  console %s\n",
		*addr, claims, evidence, gate)

	srv := &http.Server{
		Addr:              *addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
			return
		}
		errc <- nil
	}()

	// ctx is cancelled by the signal handler in main. On the signal, drain with
	// a fresh context: the signalled one is already done, and Shutdown needs a
	// live one to let in-flight requests finish.
	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
		if err := srv.Shutdown(context.WithoutCancel(ctx)); err != nil {
			return fmt.Errorf("web: shutdown: %w", err)
		}
		return nil
	}
}

// site is the resolved source of SPA files: the embedded bundle, web/dist on
// disk, or nothing (which serves a stub explaining how to build it).
type site struct {
	fsys fs.FS
}

// resolveSite picks the SPA source, preferring the embedded bundle, then
// web/dist on disk, and otherwise nothing.
func resolveSite(stderr io.Writer) site {
	if WebAssets != nil {
		return site{fsys: WebAssets}
	}
	if info, err := os.Stat("web/dist/index.html"); err == nil && !info.IsDir() {
		return site{fsys: os.DirFS("web/dist")}
	}
	fmt.Fprintf(stderr, "cred web  SPA not built; serving a stub. Run `task build`, or `task dev` for development.\n")
	return site{}
}

// spaHandler serves static assets and falls back to index.html for any
// unmatched path, so client-side routing works on a hard refresh. Unknown /api
// paths return a JSON 404 rather than the SPA shell.
func spaHandler(s site) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		if p == "/api" || strings.HasPrefix(p, "/api/") {
			c.JSON(http.StatusNotFound, api.ErrorResponse{Error: "not found"})
			return
		}
		if s.fsys == nil {
			serveStub(c)
			return
		}

		name := strings.TrimPrefix(path.Clean("/"+p), "/")
		if name == "" {
			name = "index.html"
		}
		if serveFile(c, s.fsys, name) {
			return
		}
		// History fallback: an unmatched path is a client-side route.
		if !serveFile(c, s.fsys, "index.html") {
			serveStub(c)
		}
	}
}

// serveFile writes the named file from fsys with the right cache header,
// returning false when it is absent or a directory so the caller can fall back.
func serveFile(c *gin.Context, fsys fs.FS, name string) bool {
	f, err := fsys.Open(name)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()
	info, err := f.Stat()
	if err != nil || info.IsDir() {
		return false
	}

	switch {
	case strings.HasPrefix(name, "assets/"):
		// Content-hashed: the name changes when the bytes do, so it is safe to
		// cache forever. This is what prevents a stale bundle after a deploy.
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	default:
		// index.html and any unhashed file must be revalidated so a deploy is
		// seen immediately.
		c.Header("Cache-Control", "no-cache")
	}

	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(c.Writer, c.Request, name, info.ModTime(), rs)
		return true
	}
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, f)
	return true
}

func serveStub(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(stubHTML))
}

const stubHTML = `<!doctype html>
<html lang="en">
<head><meta charset="utf-8"><title>CRED console</title></head>
<body style="font-family: system-ui, sans-serif; max-width: 40rem; margin: 4rem auto; padding: 0 1rem;">
<h1>CRED console</h1>
<p>The web console is not built yet. The API is live under <code>/api</code>.</p>
<p>Build the SPA with <code>task build</code>, or run <code>task dev</code> for a live-reloading dev server.</p>
</body>
</html>
`
