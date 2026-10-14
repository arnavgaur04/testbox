package api

import (
	"encoding/json"
	"net/http"
	"testbox/internal/engine"
)

// Holds the Engine for handlers.
// HTTP stays thin. Logic in Engine.
type Server struct {
	eng *engine.Engine
}

func NewServer() *Server {
	return &Server{eng: engine.NewEngine()}
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
func (s *Server) handleDown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeErr(w, http.StatusMethodNotAllowed, "use POST")
		return
	}
	repo, ok := s.decodeRepo(w, r)
	if !ok {
		return
	}
	if err := s.eng.Down(repo); err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeOK(w, "down completed")
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
	mux.HandleFunc("/v1/run", s.handleRun)
	mux.HandleFunc("/v1/status", s.handleStatus)
	mux.HandleFunc("/v1/logs", s.handleLogs)
	return mux
}
