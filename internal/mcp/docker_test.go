package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteBuildContext(t *testing.T) {
	dir := t.TempDir()
	// Local spec files inside the fake repo.
	if err := os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("openapi: 3.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), []byte("tools: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := WriteBuildContext(dir, "./openapi.yaml", "./tools.yaml", "http://my-app:80", 18082, "", []string{"my-app"})
	if err != nil {
		t.Fatalf("write context: %v", err)
	}
	if info.Target != "http://my-app:80" {
		t.Errorf("wrong target: %s", info.Target)
	}
	if info.HostPort != 18082 || info.ContainerPort != 8082 {
		t.Errorf("wrong ports: %+v", info)
	}
	if info.BuildContext == "" {
		t.Error("expected build context for local build")
	}
	dockerfile, err := os.ReadFile(filepath.Join(dir, ".testbox", "mcp", "Dockerfile"))
	if err != nil {
		t.Fatalf("read Dockerfile: %v", err)
	}
	content := string(dockerfile)
	if !strings.Contains(content, "cmd/mcp-server") {
		t.Error("Dockerfile must build cmd/mcp-server")
	}
	if !strings.Contains(content, "COPY openapi.yaml /mcp/openapi.yaml") {
		t.Error("Dockerfile must COPY staged openapi")
	}
	if !strings.Contains(content, "COPY tools.yaml /mcp/tools.yaml") {
		t.Error("Dockerfile must COPY staged tools")
	}
	cmd := strings.Join(info.Command(), " ")
	for _, want := range []string{"--transport http", "--target http://my-app:80", "--openapi /mcp/openapi.yaml", "--tools /mcp/tools.yaml"} {
		if !strings.Contains(cmd, want) {
			t.Errorf("command missing %q: %s", want, cmd)
		}
	}
}

func TestWriteBuildContextURLPassthrough(t *testing.T) {
	dir := t.TempDir()
	info, err := WriteBuildContext(dir, "https://x/openapi.yaml", "", "http://a:80", 8082, "", []string{"a"})
	if err != nil {
		t.Fatalf("write context: %v", err)
	}
	if info.OpenAPIRef != "https://x/openapi.yaml" {
		t.Errorf("URL must pass through: %s", info.OpenAPIRef)
	}
	if info.HasOpenAPI {
		t.Error("URL must not set HasOpenAPI")
	}
	dockerfile, _ := os.ReadFile(filepath.Join(dir, ".testbox", "mcp", "Dockerfile"))
	if strings.Contains(string(dockerfile), "COPY openapi.yaml") {
		t.Error("Dockerfile must not COPY missing openapi")
	}
}

func TestWriteBuildContextPrebuiltImage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "tools.yaml"), []byte("tools: []\n"), 0644); err != nil {
		t.Fatal(err)
	}
	info, err := WriteBuildContext(dir, "", "./tools.yaml", "http://a:80", 8082, "my-mcp:1", []string{"a"})
	if err != nil {
		t.Fatalf("write context: %v", err)
	}
	if info.BuildContext != "" {
		t.Error("prebuilt image must not set build context")
	}
	if info.Image != "my-mcp:1" {
		t.Errorf("wrong image: %s", info.Image)
	}
	if _, err := os.Stat(filepath.Join(dir, ".testbox", "mcp", "Dockerfile")); !os.IsNotExist(err) {
		t.Error("prebuilt image must not write Dockerfile")
	}
}

// Mirrors CI with -repo . in repo.
// Context must stay absolute.
func TestWriteBuildContextRelativeRepo(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("openapi: 3.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	info, err := WriteBuildContext(".", "./openapi.yaml", "", "http://a:80", 8082, "", []string{"a"})
	if err != nil {
		t.Fatalf("write context: %v", err)
	}
	if !filepath.IsAbs(info.BuildContext) {
		t.Errorf("build context must be absolute, got %s", info.BuildContext)
	}
	if _, err := os.Stat(filepath.Join(info.BuildContext, "Dockerfile")); err != nil {
		t.Errorf("Dockerfile must exist at absolute context: %v", err)
	}
}
