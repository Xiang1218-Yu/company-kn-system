package web

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// SPA serves a compiled single-page app from a directory. Any path that maps to
// a real file is served as-is; anything else returns index.html so client-side
// routing (e.g. /kb/:id/documents) works on refresh and direct links.
//
// It is intentionally simple: it does not do directory listing and treats a
// missing root as a no-op (so the API still runs when the frontend isn't built
// in, e.g. during pure-backend development).
type SPA struct {
	root    string
	index   string
	enabled bool
}

// New constructs an SPA handler rooted at dir. If dir does not exist, the
// handler is disabled and Register is a no-op — the binary can run with or
// without a bundled frontend.
func New(dir string) *SPA {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return &SPA{enabled: false}
	}
	if _, err := os.Stat(filepath.Join(abs, "index.html")); err != nil {
		return &SPA{enabled: false}
	}
	return &SPA{root: abs, index: filepath.Join(abs, "index.html"), enabled: true}
}

// Enabled reports whether a frontend bundle was found. Used by callers that
// want to log whether they are serving the SPA.
func (s *SPA) Enabled() bool { return s.enabled }

// Register attaches the SPA as a NoRoute handler on the engine. It is a no-op
// when disabled. API routes are registered separately and take precedence
// because Gin matches registered routes before NoRoute.
func (s *SPA) Register(r *gin.Engine) {
	if !s.enabled {
		return
	}
	r.NoRoute(func(c *gin.Context) {
		rel := strings.TrimPrefix(c.Request.URL.Path, "/")
		if rel == "" {
			rel = "index.html"
		}
		// Clean prevents traversal: filepath.Clean on a joined root+rel can
		// still escape only if rel starts with .., which we reject here.
		if strings.Contains(rel, "..") {
			c.String(http.StatusForbidden, "forbidden")
			return
		}
		full := filepath.Join(s.root, filepath.Clean(rel))
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			c.Header("Cache-Control", "no-cache")
			http.ServeFile(c.Writer, c.Request, full)
			return
		}
		// Fall back to the SPA shell for client-side routing.
		c.Header("Cache-Control", "no-cache")
		http.ServeFile(c.Writer, c.Request, s.index)
	})
}
