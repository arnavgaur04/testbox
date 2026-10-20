package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardNoTokenAllows(t *testing.T) {
	g, err := NewGuard("", nil)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	if !g.Authorized(req) {
		t.Error("expected open access without token")
	}
}

func TestGuardTokenRequired(t *testing.T) {
	g, _ := NewGuard("s3cret", nil)
	s := NewServerWithGuard(g)

	req := httptest.NewRequest(http.MethodGet, "/v1/status?repo=/tmp", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", rec.Code)
	}
	if rec.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}

	req = httptest.NewRequest(http.MethodGet, "/v1/status?repo=/tmp", nil)
	req.Header.Set("Authorization", "Bearer s3cret")
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code == http.StatusUnauthorized {
		t.Error("expected authorized request to pass")
	}
}

func TestGuardBadToken(t *testing.T) {
	g, _ := NewGuard("s3cret", nil)
	s := NewServerWithGuard(g)
	req := httptest.NewRequest(http.MethodGet, "/v1/status?repo=/tmp", nil)
	req.Header.Set("Authorization", "Bearer wrong")
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for wrong token, got %d", rec.Code)
	}
}

func TestGuardHealthStaysOpen(t *testing.T) {
	g, _ := NewGuard("s3cret", nil)
	s := NewServerWithGuard(g)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("expected health 200 without token, got %d", rec.Code)
	}
}

func TestGuardRepoAllowed(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "repo")
	if err := os.Mkdir(inside, 0755); err != nil {
		t.Fatal(err)
	}
	g, err := NewGuard("", []string{root})
	if err != nil {
		t.Fatal(err)
	}
	if err := g.RepoAllowed(inside); err != nil {
		t.Errorf("expected inside repo allowed: %v", err)
	}
	if err := g.RepoAllowed(t.TempDir()); err == nil {
		t.Error("expected outside repo rejected")
	}
	if err := g.RepoAllowed(root); err != nil {
		t.Errorf("expected root itself allowed: %v", err)
	}
}

func TestGuardRepoPrefixTrap(t *testing.T) {
	root := t.TempDir()
	sibling := root + "-evil"
	if err := os.Mkdir(sibling, 0755); err != nil {
		t.Fatal(err)
	}
	g, _ := NewGuard("", []string{root})
	if err := g.RepoAllowed(sibling); err == nil {
		t.Error("prefix sibling must be rejected")
	}
}

func TestGuardRepoSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	link := filepath.Join(root, "escape")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink not supported: %v", err)
	}
	g, _ := NewGuard("", []string{root})
	if err := g.RepoAllowed(link); err == nil {
		t.Error("symlink escape must be rejected")
	}
}

func TestGuardRepoForbiddenRoute(t *testing.T) {
	root := t.TempDir()
	g, _ := NewGuard("", []string{root})
	s := NewServerWithGuard(g)
	body := `{"repo":"` + strings.ReplaceAll(t.TempDir(), `\`, `\\`) + `"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/up", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Errorf("expected 403 for outside repo, got %d", rec.Code)
	}
}

func TestGuardNewGuardBadRoot(t *testing.T) {
	if _, err := NewGuard("", []string{"/no/such/root"}); err == nil {
		t.Error("expected error for missing root")
	}
}
