package deployer

import (
	"os"
	"path/filepath"
	"strings"
	"testbox/pkg/models"
	"testing"
)

func TestGenerateComposeFile(t *testing.T) {
	dir := t.TempDir()
	cfg := &models.MakeYaml{}
	cfg.Docker.ImageName = "my-test-app:latest"
	cfg.LocalDeploy.Services = []models.LocalServiceConfig{
		{ServiceName: "my-test-app", Port: 8080, ContainerPort: 80},
	}

	if err := GenerateComposeFile(dir, cfg, "master"); err != nil {
		t.Fatalf("generate failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, ".testbox", "docker-compose.local.yaml"))
	if err != nil {
		t.Fatalf("read generated file: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "version:") {
		t.Error("generated file must not contain obsolete version key")
	}
	if !strings.Contains(content, `"8080:80"`) {
		t.Errorf("expected HOST:CONTAINER mapping 8080:80, got:\n%s", content)
	}
	if !strings.Contains(content, "my-test-app:latest") {
		t.Errorf("expected image name, got:\n%s", content)
	}
}

func TestStatusMissingFile(t *testing.T) {
	if _, err := Status(t.TempDir(), "/no/such/compose.yaml"); err == nil {
		t.Error("expected error for missing compose file")
	}
}

func TestLogsMissingFile(t *testing.T) {
	if _, err := Logs(t.TempDir(), "/no/such/compose.yaml", "", "10"); err == nil {
		t.Error("expected error for missing compose file")
	}
}

func TestUpMissingFile(t *testing.T) {
	if err := Up(t.TempDir(), "/no/such/compose.yaml"); err == nil {
		t.Error("expected error for missing compose file")
	}
}

func TestDownMissingFile(t *testing.T) {
	if err := Down(t.TempDir(), "/no/such/compose.yaml"); err == nil {
		t.Error("expected error for missing compose file")
	}
}

func mcpTestConfig() *models.MakeYaml {
	cfg := &models.MakeYaml{}
	cfg.Docker.ImageName = "my-test-app:latest"
	cfg.LocalDeploy.Services = []models.LocalServiceConfig{
		{ServiceName: "my-test-app", Port: 8080, ContainerPort: 80},
	}
	cfg.Mcp = &models.McpConfig{Enabled: true, OpenAPI: "./openapi.yaml", Port: 18082}
	return cfg
}

func TestGenerateComposeWithMcp(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("openapi: 3.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := GenerateComposeFile(dir, mcpTestConfig(), "master"); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".testbox", "docker-compose.local.yaml"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	for _, want := range []string{"mcp:", "http://my-test-app:80", `"18082:8082"`, "TESTBOX_REF: master", "- my-test-app"} {
		if !strings.Contains(content, want) {
			t.Errorf("expected %q in compose:\n%s", want, content)
		}
	}
}

func TestGenerateComposeWithoutMcp(t *testing.T) {
	dir := t.TempDir()
	cfg := mcpTestConfig()
	cfg.Mcp = nil
	if err := GenerateComposeFile(dir, cfg, "master"); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".testbox", "docker-compose.local.yaml"))
	if strings.Contains(string(data), "mcp:") {
		t.Errorf("mcp service must be absent when disabled:\n%s", data)
	}
}

func TestGenerateComposeMcpImageOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "openapi.yaml"), []byte("openapi: 3.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	cfg := mcpTestConfig()
	cfg.Mcp.Image = "my-mcp:9"
	if err := GenerateComposeFile(dir, cfg, "master"); err != nil {
		t.Fatalf("generate failed: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".testbox", "docker-compose.local.yaml"))
	content := string(data)
	if !strings.Contains(content, "image: my-mcp:9") {
		t.Errorf("expected image override:\n%s", content)
	}
	if strings.Contains(content, "build:") {
		t.Errorf("image override must not set build:\n%s", content)
	}
}
