package api

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// Guards the API.
// Empty token means no auth.
type Guard struct {
	Token        string
	AllowedRepos []string
}

// Absolute roots, symlinks resolved.
func NewGuard(token string, allowedRepos []string) (*Guard, error) {
	g := &Guard{Token: token}
	for _, root := range allowedRepos {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, fmt.Errorf("bad allowed repo %s: %w", root, err)
		}
		real, err := filepath.EvalSymlinks(abs)
		if err != nil {
			return nil, fmt.Errorf("bad allowed repo %s: %w", root, err)
		}
		g.AllowedRepos = append(g.AllowedRepos, real)
	}
	return g, nil
}

// Checks the bearer token.
// Empty token skips the check.
func (g *Guard) Authorized(r *http.Request) bool {
	if g.Token == "" {
		return true
	}
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return false
	}
	got := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return subtle.ConstantTimeCompare([]byte(got), []byte(g.Token)) == 1
}

// Checks repo is under an allowed root.
// Empty list allows every repo.
func (g *Guard) RepoAllowed(repo string) error {
	if len(g.AllowedRepos) == 0 {
		return nil
	}
	abs, err := filepath.Abs(repo)
	if err != nil {
		return fmt.Errorf("bad repo path: %w", err)
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return fmt.Errorf("repo not allowed: %s", repo)
	}
	for _, root := range g.AllowedRepos {
		if real == root || strings.HasPrefix(real, root+string(os.PathSeparator)) {
			return nil
		}
	}
	return fmt.Errorf("repo not allowed: %s", repo)
}

// Wraps a handler with auth.
// /health stays open.
func (g *Guard) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		if !g.Authorized(r) {
			w.Header().Set("WWW-Authenticate", `Bearer realm="testbox"`)
			writeErr(w, http.StatusUnauthorized, "missing or bad bearer token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
