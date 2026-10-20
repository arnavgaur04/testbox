package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testbox/internal/deployer"
	"testing"
	"time"
)

func sampleResults() []deployer.SuiteResult {
	return []deployer.SuiteResult{
		{Name: "unit", Passed: true, Output: "ok", Duration: 1500 * time.Millisecond},
		{Name: "api", Passed: false, Output: "boom <&>", Duration: 250 * time.Millisecond},
	}
}

func TestBuildCounts(t *testing.T) {
	run := Build("/repo", "app", "1.2.3", time.Now(), sampleResults())
	if run.Passed != 1 || run.Failed != 1 {
		t.Errorf("wrong counts: %+v", run)
	}
	if run.App != "app" || run.Version != "1.2.3" || run.Repo != "/repo" {
		t.Errorf("wrong metadata: %+v", run)
	}
	if len(run.Suites) != 2 || run.Suites[1].DurationMS != 250 {
		t.Errorf("wrong suites: %+v", run.Suites)
	}
}

func TestBuildEmptyResults(t *testing.T) {
	run := Build("/repo", "app", "1", time.Now(), nil)
	if run.Passed != 0 || run.Failed != 0 || len(run.Suites) != 0 {
		t.Errorf("expected empty run, got %+v", run)
	}
}

func TestWriteJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.json")
	run := Build("/repo", "app", "1", time.Now(), sampleResults())
	if err := WriteJSON(path, run); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var got Run
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if got.Passed != 1 || got.Failed != 1 || len(got.Suites) != 2 {
		t.Errorf("roundtrip mismatch: %+v", got)
	}
}

func TestWriteJSONBadPath(t *testing.T) {
	if err := WriteJSON("/no/such/dir/r.json", Run{}); err == nil {
		t.Error("expected error for bad path")
	}
}

func TestBuildJUnit(t *testing.T) {
	run := Build("/repo", "app", "1", time.Now(), sampleResults())
	doc := BuildJUnit(run)
	if doc.Tests != 2 || doc.Failures != 1 {
		t.Errorf("wrong counts: %+v", doc)
	}
	if doc.Suites[0].Cases[0].Failure != nil {
		t.Error("passing suite must have no failure")
	}
	if doc.Suites[1].Cases[0].Failure == nil {
		t.Error("failing suite needs failure node")
	}
	if doc.Suites[0].Time != "1.500" {
		t.Errorf("wrong time format: %q", doc.Suites[0].Time)
	}
}

func TestWriteJUnit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "report.xml")
	run := Build("/repo", "app", "1", time.Now(), sampleResults())
	if err := WriteJUnit(path, run); err != nil {
		t.Fatalf("write: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	content := string(data)
	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		`<testsuites name="app" tests="2" failures="1"`,
		`<testcase name="api" classname="app"`,
		`<failure message="suite failed">boom &lt;&amp;&gt;</failure>`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("missing %q in:\n%s", want, content)
		}
	}
}

func TestWriteJUnitBadPath(t *testing.T) {
	if err := WriteJUnit("/no/such/dir/r.xml", Run{}); err == nil {
		t.Error("expected error for bad path")
	}
}
