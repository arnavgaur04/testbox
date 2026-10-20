package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validMake = `app:
  name: "demo"
  version: "1"
docker:
  context: "."
  dockerfile: "Dockerfile"
  image_name: "demo:latest"
local_deploy:
  auto_generate_compose: true
  services:
    - service_name: "demo"
      port: 18080
`

func writeRepo(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "make.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestLoadConfigMissingFile(t *testing.T) {
	e := NewEngine()
	if _, err := e.LoadConfig(t.TempDir()); err == nil {
		t.Error("expected error for missing make.yaml")
	}
}

func TestLoadConfigBadYaml(t *testing.T) {
	e := NewEngine()
	dir := writeRepo(t, "app: [")
	if _, err := e.LoadConfig(dir); err == nil {
		t.Error("expected error for bad yaml")
	}
}

func TestEnsureTestboxDir(t *testing.T) {
	e := NewEngine()
	dir := t.TempDir()
	got, err := e.ensureTestboxDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != filepath.Join(dir, ".testbox") {
		t.Errorf("wrong dir: %s", got)
	}
	if _, err := os.Stat(got); err != nil {
		t.Errorf("dir not created: %v", err)
	}
}

func TestTestNoSuitesSkipsDocker(t *testing.T) {
	e := NewEngine()
	dir := writeRepo(t, validMake)
	results, err := e.Test(dir, TestOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results != nil {
		t.Errorf("expected nil results, got %+v", results)
	}
}

func TestHelmConfigMissing(t *testing.T) {
	e := NewEngine()
	dir := writeRepo(t, validMake)
	if _, err := e.helmConfig(dir); err == nil || !strings.Contains(err.Error(), "helm") {
		t.Errorf("expected helm error, got %v", err)
	}
}

func TestHelmConfigPresent(t *testing.T) {
	e := NewEngine()
	dir := writeRepo(t, validMake+`helm:
  chart_path: ./chart
  release_name: app
`)
	cfg, err := e.helmConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ReleaseName != "app" {
		t.Errorf("wrong release: %s", cfg.ReleaseName)
	}
}

func TestResolveComposeFilesAuto(t *testing.T) {
	e := NewEngine()
	dir := writeRepo(t, validMake)
	config, err := e.LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	files, err := e.resolveComposeFiles(dir, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected one file, got %v", files)
	}
	if _, err := os.Stat(files[0]); err != nil {
		t.Errorf("compose file not written: %v", err)
	}
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "demo:latest") {
		t.Errorf("compose missing image:\n%s", data)
	}
}

func TestResolveComposeFilesCustom(t *testing.T) {
	e := NewEngine()
	y := strings.Replace(validMake, "  auto_generate_compose: true", "  compose_file: ./compose.yaml", 1)
	dir := writeRepo(t, y)
	config, err := e.LoadConfig(dir)
	if err != nil {
		t.Fatal(err)
	}
	files, err := e.resolveComposeFiles(dir, config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != filepath.Join(dir, "compose.yaml") {
		t.Errorf("wrong files: %v", files)
	}
}
