package mcp

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleSpec = `openapi: 3.0.0
info:
  title: Sample
  version: 1.0.0
paths:
  /users/{id}:
    get:
      operationId: getUser
      summary: Get one user
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: integer
        - name: verbose
          in: query
          required: false
          schema:
            type: boolean
  /users:
    get:
      summary: List users
      parameters:
        - name: limit
          in: query
          required: false
          schema:
            type: integer
    post:
      summary: Create user
`

func writeTemp(t *testing.T, name string, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write temp: %v", err)
	}
	return p
}

func TestLoadOpenAPITools(t *testing.T) {
	p := writeTemp(t, "openapi.yaml", sampleSpec)
	tools, err := LoadOpenAPITools(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("expected 2 GET tools (POST skipped), got %d", len(tools))
	}
	byName := map[string]ToolDef{}
	for _, tl := range tools {
		byName[tl.Name] = tl
	}
	getUser, ok := byName["getuser"]
	if !ok {
		t.Fatalf("expected getuser tool, got %v", tools)
	}
	if len(getUser.Params) != 2 {
		t.Errorf("expected 2 params, got %d", len(getUser.Params))
	}
	// Path params stay required.
	for _, pr := range getUser.Params {
		if pr.Name == "id" && !pr.Required {
			t.Error("path param id must be required")
		}
	}
}

func TestLoadOpenAPIFromURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, sampleSpec)
	}))
	defer srv.Close()
	tools, err := LoadOpenAPITools(srv.URL + "/openapi.yaml")
	if err != nil {
		t.Fatalf("load from URL: %v", err)
	}
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

func TestLoadCustomTools(t *testing.T) {
	y := `tools:
  - name: ping_app
    description: Ping the app.
    method: GET
    path: /ping
`
	p := writeTemp(t, "tools.yaml", y)
	tools, err := LoadCustomTools(p)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(tools) != 1 || tools[0].Name != "ping_app" {
		t.Errorf("wrong tools: %v", tools)
	}
}

func TestLoadCustomToolsRejectsPost(t *testing.T) {
	y := `tools:
  - name: create_thing
    description: Create.
    method: POST
    path: /things
`
	p := writeTemp(t, "tools.yaml", y)
	if _, err := LoadCustomTools(p); err == nil {
		t.Error("expected error for POST tool in GET-only phase")
	}
}

func TestMergeToolsOverrideAndAdd(t *testing.T) {
	auto := []ToolDef{
		{Name: "getuser", Description: "auto", Method: "GET", Path: "/users/{id}"},
		{Name: "listusers", Description: "auto", Method: "GET", Path: "/users"},
	}
	custom := []ToolDef{
		{Name: "getuser", Description: "custom edit", Method: "GET", Path: "/users/{id}"},
		{Name: "ping_app", Description: "new", Method: "GET", Path: "/ping"},
	}
	merged := MergeTools(auto, custom)
	if len(merged) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(merged))
	}
	for _, m := range merged {
		if m.Name == "getuser" && m.Description != "custom edit" {
			t.Error("custom tool must replace auto tool with same name")
		}
	}
}

func TestExecGET(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	def := ToolDef{
		Name: "getuser", Method: "GET", Path: "/users/{id}",
		Params: []ParamDef{
			{Name: "id", In: "path", Required: true},
			{Name: "verbose", In: "query", Required: false},
		},
	}
	out, err := ExecGET(context.Background(), srv.URL, def, map[string]any{"id": "42", "verbose": "true"}, nil)
	if err != nil {
		t.Fatalf("exec: %v", err)
	}
	if gotPath != "/users/42" {
		t.Errorf("wrong path: %s", gotPath)
	}
	if !strings.Contains(gotQuery, "verbose=true") {
		t.Errorf("wrong query: %s", gotQuery)
	}
	if !strings.Contains(out, "HTTP 200") {
		t.Errorf("expected HTTP 200 in output, got %s", out)
	}
}

func TestExecGETMissingPathParam(t *testing.T) {
	def := ToolDef{
		Name: "getuser", Method: "GET", Path: "/users/{id}",
		Params: []ParamDef{{Name: "id", In: "path", Required: true}},
	}
	if _, err := ExecGET(context.Background(), "http://x", def, map[string]any{}, nil); err == nil {
		t.Error("expected error for missing path param")
	}
}

func TestExecGETSendsHeaders(t *testing.T) {
	var gotAuth, gotKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotKey = r.Header.Get("X-Api-Key")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	def := ToolDef{Name: "ping", Method: "GET", Path: "/ping"}
	headers := map[string]string{
		"Authorization": "Bearer tok",
		"X-Api-Key":     "k1",
	}
	if _, err := ExecGET(context.Background(), srv.URL, def, nil, headers); err != nil {
		t.Fatalf("exec: %v", err)
	}
	if gotAuth != "Bearer tok" {
		t.Errorf("wrong Authorization: %q", gotAuth)
	}
	if gotKey != "k1" {
		t.Errorf("wrong X-Api-Key: %q", gotKey)
	}
}

func TestExecGETNoHeaders(t *testing.T) {
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		fmt.Fprint(w, "ok")
	}))
	defer srv.Close()
	def := ToolDef{Name: "ping", Method: "GET", Path: "/ping"}
	if _, err := ExecGET(context.Background(), srv.URL, def, nil, nil); err != nil {
		t.Fatalf("exec: %v", err)
	}
	if auth != "" {
		t.Errorf("expected no auth header, got %q", auth)
	}
}
