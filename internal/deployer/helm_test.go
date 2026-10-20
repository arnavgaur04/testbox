package deployer

import (
	"os"
	"path/filepath"
	"strings"
	"testbox/pkg/models"
	"testing"
)

func TestHelmUpArgsFull(t *testing.T) {
	cfg := models.HelmConfig{
		ReleaseName: "api",
		Namespace:   "prod",
		KubeContext: "colima",
	}
	got := strings.Join(HelmUpArgs(cfg, "/repo/chart", "/repo/values.yaml"), " ")
	for _, want := range []string{
		"upgrade --install api /repo/chart",
		"--wait --timeout 10m",
		"--namespace prod --create-namespace",
		"--kube-context colima",
		"--values /repo/values.yaml",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in %q", want, got)
		}
	}
}

func TestHelmUpArgsMinimal(t *testing.T) {
	got := strings.Join(HelmUpArgs(models.HelmConfig{ReleaseName: "api"}, "/c", ""), " ")
	if strings.Contains(got, "--values") || strings.Contains(got, "--namespace") {
		t.Errorf("unexpected optional flags: %q", got)
	}
}

func TestHelmStatusAndDownArgs(t *testing.T) {
	cfg := models.HelmConfig{ReleaseName: "api", Namespace: "prod"}
	status := strings.Join(HelmStatusArgs(cfg), " ")
	if status != "status api --namespace prod" {
		t.Errorf("wrong status args: %q", status)
	}
	down := strings.Join(HelmDownArgs(cfg), " ")
	if down != "uninstall api --namespace prod" {
		t.Errorf("wrong down args: %q", down)
	}
}

func TestResolveHelmPaths(t *testing.T) {
	dir := t.TempDir()
	chart := filepath.Join(dir, "chart")
	if err := os.MkdirAll(chart, 0755); err != nil {
		t.Fatal(err)
	}
	values := filepath.Join(dir, "values.yaml")
	if err := os.WriteFile(values, []byte("a: b\n"), 0644); err != nil {
		t.Fatal(err)
	}
	gotChart, gotValues, err := resolveHelmPaths(dir, models.HelmConfig{ChartPath: "chart", ValuesFile: "values.yaml"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotChart != chart || gotValues != values {
		t.Errorf("wrong paths: %q %q", gotChart, gotValues)
	}
}

func TestResolveHelmPathsMissingChart(t *testing.T) {
	_, _, err := resolveHelmPaths(t.TempDir(), models.HelmConfig{ChartPath: "nope"})
	if err == nil || !strings.Contains(err.Error(), "chart not found") {
		t.Errorf("expected missing chart error, got %v", err)
	}
}

func TestHelmUpMissingChartFails(t *testing.T) {
	err := HelmUp(t.TempDir(), models.HelmConfig{ChartPath: "nope", ReleaseName: "api"})
	if err == nil {
		t.Error("expected error before running helm")
	}
}
