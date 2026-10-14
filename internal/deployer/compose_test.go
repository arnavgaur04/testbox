package deployer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testbox/pkg/models"
)

func TestGenerateComposeFile(t *testing.T) {
	dir := t.TempDir()
	cfg := &models.MakeYaml{}
	cfg.Docker.ImageName = "my-test-app:latest"
	cfg.LocalDeploy.Services = []models.LocalServiceConfig{
		{ServiceName: "my-test-app", Port: 8080, ContainerPort: 80},
	}

	if err := GenerateComposeFile(dir, cfg); err != nil {
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
