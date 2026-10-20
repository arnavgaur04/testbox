package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealth(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body["ok"] != true {
		t.Errorf("expected ok true, got %v", body)
	}
}

func TestBuildMethodNotAllowed(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/build", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", rec.Code)
	}
}

func TestBuildInvalidJSON(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/build", strings.NewReader("{bad"))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestBuildMissingRepo(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/build", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestStatusMissingRepoParam(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/status", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestResetBadRepo(t *testing.T) {
	s := NewServer()
	body := `{"repo":"/no/such/dir"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/reset", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for bad repo, got %d", rec.Code)
	}
}

func TestDownVolumesFlag(t *testing.T) {
	s := NewServer()
	body := `{"repo":"/no/such/dir","volumes":true}`
	req := httptest.NewRequest(http.MethodPost, "/v1/down", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for bad repo, got %d", rec.Code)
	}
}
func TestStatusBadRepo(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodGet, "/v1/status?repo=/no/such/dir", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected 500 for bad repo, got %d", rec.Code)
	}
}

func TestLogsMissingRepo(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/logs", strings.NewReader(`{"service":"x"}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestTestBadWait(t *testing.T) {
	s := NewServer()
	body := `{"repo":"/tmp","wait":"soon"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/test", strings.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad wait, got %d", rec.Code)
	}
}

func TestTestMissingRepo(t *testing.T) {
	s := NewServer()
	req := httptest.NewRequest(http.MethodPost, "/v1/test", strings.NewReader(`{}`))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing repo, got %d", rec.Code)
	}
}
