package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTempYaml(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "make.yaml")
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatalf("write temp yaml: %v", err)
	}
	return p
}

const validYaml = `app:
  name: "my-test-app"
  version: "0.1.0"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "my-test-app:latest"
local_deploy:
  auto_generate_compose: true
  services:
    - service_name: "my-test-app"
      port: 8080
      container_port: 80
`

func TestParseValid(t *testing.T) {
	p := writeTempYaml(t, validYaml)
	cfg, err := ParseMakeYaml(p)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.App.Name != "my-test-app" {
		t.Errorf("wrong app name: %s", cfg.App.Name)
	}
	if cfg.LocalDeploy.Services[0].ContainerPort != 80 {
		t.Errorf("wrong container_port: %d", cfg.LocalDeploy.Services[0].ContainerPort)
	}
}

func TestContainerPortDefaultsToPort(t *testing.T) {
	y := `app:
  name: "a"
  version: "1"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "a:latest"
local_deploy:
  auto_generate_compose: true
  services:
    - service_name: "a"
      port: 8080
`
	p := writeTempYaml(t, y)
	cfg, err := ParseMakeYaml(p)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.LocalDeploy.Services[0].ContainerPort != 8080 {
		t.Errorf("expected default container_port 8080, got %d", cfg.LocalDeploy.Services[0].ContainerPort)
	}
}

func TestMissingAppName(t *testing.T) {
	y := `app:
  name: ""
  version: "1"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "a:latest"
local_deploy:
  auto_generate_compose: true
  services:
    - service_name: "a"
      port: 8080
`
	p := writeTempYaml(t, y)
	if _, err := ParseMakeYaml(p); err == nil {
		t.Error("expected error for empty app.name")
	}
}

func TestComposeFileRequiredWhenNotAuto(t *testing.T) {
	y := `app:
  name: "a"
  version: "1"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "a:latest"
local_deploy:
  auto_generate_compose: false
  services:
    - service_name: "a"
      port: 8080
`
	p := writeTempYaml(t, y)
	if _, err := ParseMakeYaml(p); err == nil {
		t.Error("expected error for missing compose_file")
	}
}

func TestInvalidPort(t *testing.T) {
	for _, port := range []int{0, -1, 99999} {
		y := `app:
  name: "a"
  version: "1"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "a:latest"
local_deploy:
  auto_generate_compose: true
  services:
    - service_name: "a"
      port: ` + itoa(port) + `
`
		p := writeTempYaml(t, y)
		if _, err := ParseMakeYaml(p); err == nil {
			t.Errorf("expected error for port %d", port)
		}
	}
}

func TestMissingFile(t *testing.T) {
	if _, err := ParseMakeYaml("/no/such/dir/make.yaml"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestMcpEnabledRequiresSource(t *testing.T) {
	y := `app:
  name: "a"
  version: "1"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "a:latest"
local_deploy:
  auto_generate_compose: true
  services:
    - service_name: "a"
      port: 8080
mcp:
  enabled: true
`
	p := writeTempYaml(t, y)
	if _, err := ParseMakeYaml(p); err == nil {
		t.Error("expected error when mcp enabled with no openapi or tools_file")
	}
}

func TestMcpValid(t *testing.T) {
	y := `app:
  name: "a"
  version: "1"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "a:latest"
local_deploy:
  auto_generate_compose: true
  services:
    - service_name: "a"
      port: 8080
mcp:
  enabled: true
  openapi: ./openapi.yaml
  tools_file: ./mcp-tools.yaml
`
	p := writeTempYaml(t, y)
	cfg, err := ParseMakeYaml(p)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Mcp == nil || !cfg.Mcp.Enabled || cfg.Mcp.OpenAPI != "./openapi.yaml" {
		t.Errorf("wrong mcp config: %+v", cfg.Mcp)
	}
}

func TestMcpAbsentIsNil(t *testing.T) {
	p := writeTempYaml(t, validYaml)
	cfg, err := ParseMakeYaml(p)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Mcp != nil {
		t.Errorf("expected nil mcp when section absent, got %+v", cfg.Mcp)
	}
}

func testsYaml(extra string) string {
	return `app:
  name: "a"
  version: "1"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "a:latest"
local_deploy:
  auto_generate_compose: true
  services:
    - service_name: "a"
      port: 8080
` + extra
}

func TestTestsValid(t *testing.T) {
	y := testsYaml(`tests:
  - name: unit
    service: a
    command: pytest -q
    copy:
      - ./tests:/app/tests
`)
	p := writeTempYaml(t, y)
	cfg, err := ParseMakeYaml(p)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(cfg.Tests) != 1 || cfg.Tests[0].Name != "unit" {
		t.Errorf("wrong tests: %+v", cfg.Tests)
	}
}

func TestTestsMissingCommand(t *testing.T) {
	y := testsYaml(`tests:
  - name: unit
    service: a
`)
	if _, err := ParseMakeYaml(writeTempYaml(t, y)); err == nil {
		t.Error("expected error for missing command")
	}
}

func TestTestsUnknownService(t *testing.T) {
	y := testsYaml(`tests:
  - name: unit
    service: ghost
    command: pytest -q
`)
	if _, err := ParseMakeYaml(writeTempYaml(t, y)); err == nil {
		t.Error("expected error for unknown service")
	}
}

func TestTestsBadCopy(t *testing.T) {
	for _, cp := range []string{"justpath", ":/dst", "src:"} {
		y := testsYaml("tests:\n  - name: unit\n    service: a\n    command: pytest -q\n    copy:\n      - " + cp + "\n")
		if _, err := ParseMakeYaml(writeTempYaml(t, y)); err == nil {
			t.Errorf("expected error for copy %q", cp)
		}
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := []byte{}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}
