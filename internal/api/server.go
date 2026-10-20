package api

import (
	"encoding/json"
	"net/http"
	"testbox/internal/engine"
	"time"
)

// Holds the Engine for handlers.
// HTTP stays thin. Logic in Engine.
type Server struct {
	eng   *engine.Engine
	guard *Guard
}

func NewServer() *Server {
	g, _ := NewGuard("", nil)
	return &Server{eng: engine.NewEngine(), guard: g}
}

// Server with auth and repo allowlist.
func NewServerWithGuard(g *Guard) *Server {
	if g == nil {
		g, _ = NewGuard("", nil)
	}
	return &Server{eng: engine.NewEngine(), guard: g}
}

// Rejects repos outside the allowlist.
func (s *Server) repoOK(w http.ResponseWriter, repo string) bool {
	if err := s.guard.RepoAllowed(repo); err != nil {
		writeErr(w, http.StatusForbidden, err.Error())
		return false
	}
	return true
}

// Common body for most routes.
type repoRequest struct {
	Repo string `json:"repo"`
}

// Body for logs. Service optional.
type logsRequest struct {
	Repo    string `json:"repo"`
	Service string `json:"service"`
	Tail    string `json:"tail"`
}

// Reply shape. ok=false on error.
type response struct {
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

func writeOK(w http.ResponseWriter, output string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response{OK: true, Output: output})
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(response{OK: false, Error: msg})
}

func (s *Server) decodeRepo(w http.ResponseWriter, r *http.Request) (string, bool) {
	var req repoRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return "", false
	}
	if req.Repo == "" {
		writeErr(w, http.StatusBadRequest, "field repo is required")
		return "", false
	}
	if !s.repoOK(w, req.Repo) {
		return "", false
	}
	return req.Repo, true
}

// Runs Engine.Build.
func (s *Server) handleBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	repo, ok := s.decodeRepo(w, r)
	if !ok {
		return
	}
	if err := s.eng.Build(repo); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "build completed")
}

// Runs Engine.Up.
func (s *Server) handleUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	repo, ok := s.decodeRepo(w, r)
	if !ok {
		return
	}
	if err := s.eng.Up(repo); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "up completed")
}

// Runs Engine.Down.
// Body may set volumes true.
func (s *Server) handleDown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var req struct {
		Repo    string `json:"repo"`
		Volumes bool   `json:"volumes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Repo == "" {
		writeErr(w, http.StatusBadRequest, "field repo is required")
		return
	}
	if !s.repoOK(w, req.Repo) {
		return
	}
	if err := s.eng.Down(req.Repo, req.Volumes); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "down completed")
}

// Runs Engine.Reset. Fresh state.
func (s *Server) handleReset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	repo, ok := s.decodeRepo(w, r)
	if !ok {
		return
	}
	if err := s.eng.Reset(repo); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "reset completed")
}

// Runs Engine.Test. Returns results.
func (s *Server) handleTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	var req struct {
		Repo    string `json:"repo"`
		Cleanup bool   `json:"cleanup"`
		Volumes bool   `json:"volumes"`
		Wait    string `json:"wait"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.Repo == "" {
		writeErr(w, http.StatusBadRequest, "field repo is required")
		return
	}
	if !s.repoOK(w, req.Repo) {
		return
	}
	opts := engine.TestOptions{Cleanup: req.Cleanup, Volumes: req.Volumes}
	if req.Wait != "" {
		d, err := time.ParseDuration(req.Wait)
		if err != nil || d <= 0 {
			writeErr(w, http.StatusBadRequest, "field wait must be a positive duration")
			return
		}
		opts.Wait = d
	}
	results, err := s.eng.Test(req.Repo, opts)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	type item struct {
		Name   string `json:"name"`
		Passed bool   `json:"passed"`
		Output string `json:"output"`
	}
	items := []item{}
	passed := 0
	for _, res := range results {
		items = append(items, item{Name: res.Name, Passed: res.Passed, Output: res.Output})
		if res.Passed {
			passed++
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"results": items,
		"passed":  passed,
		"failed":  len(items) - passed,
	})
}

// Runs Engine.HelmUp.
func (s *Server) handleHelmUp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	repo, ok := s.decodeRepo(w, r)
	if !ok {
		return
	}
	if err := s.eng.HelmUp(repo); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "helm deploy completed")
}

// Runs Engine.HelmStatus.
func (s *Server) handleHelmStatus(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	if r.Method == http.MethodPost {
		var req repoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		repo = req.Repo
	}
	if repo == "" {
		writeErr(w, http.StatusBadRequest, "field repo is required (query ?repo= or JSON)")
		return
	}
	if !s.repoOK(w, repo) {
		return
	}
	out, err := s.eng.HelmStatus(repo)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, out)
}

// Runs Engine.HelmDown.
func (s *Server) handleHelmDown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	repo, ok := s.decodeRepo(w, r)
	if !ok {
		return
	}
	if err := s.eng.HelmDown(repo); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "helm release removed")
}

// Runs Build then Up.
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	repo, ok := s.decodeRepo(w, r)
	if !ok {
		return
	}
	if err := s.eng.Run(repo); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "run completed")
}

// Status via GET query or POST.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	if r.Method == http.MethodPost {
		var req repoRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		repo = req.Repo
	}
	if repo == "" {
		writeErr(w, http.StatusBadRequest, "field repo is required (query ?repo= or JSON)")
		return
	}
	if !s.repoOK(w, repo) {
		return
	}
	out, err := s.eng.Status(repo)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, out)
}

// Logs via GET query or POST.
func (s *Server) handleLogs(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get("repo")
	service := r.URL.Query().Get("service")
	tail := r.URL.Query().Get("tail")
	if r.Method == http.MethodPost {
		var req logsRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		repo = req.Repo
		service = req.Service
		tail = req.Tail
	}
	if repo == "" {
		writeErr(w, http.StatusBadRequest, "field repo is required")
		return
	}
	if !s.repoOK(w, repo) {
		return
	}
	out, err := s.eng.Logs(repo, service, tail)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, out)
}

// Liveness check for clients.
func handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, "ok")
}

// Wires all routes in one place.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handleHealth)
	mux.HandleFunc("/v1/build", s.handleBuild)
	mux.HandleFunc("/v1/up", s.handleUp)
	mux.HandleFunc("/v1/down", s.handleDown)
	mux.HandleFunc("/v1/reset", s.handleReset)
	mux.HandleFunc("/v1/test", s.handleTest)
	mux.HandleFunc("/v1/run", s.handleRun)
	mux.HandleFunc("/v1/helm/up", s.handleHelmUp)
	mux.HandleFunc("/v1/helm/status", s.handleHelmStatus)
	mux.HandleFunc("/v1/helm/down", s.handleHelmDown)
	mux.HandleFunc("/v1/status", s.handleStatus)
	mux.HandleFunc("/v1/logs", s.handleLogs)
	return s.guard.Middleware(mux)
}
