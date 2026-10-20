package report

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"testbox/internal/deployer"
	"time"
)

// One suite in a report.
type Suite struct {
	Name       string `json:"name"`
	Passed     bool   `json:"passed"`
	DurationMS int64  `json:"duration_ms"`
	Output     string `json:"output,omitempty"`
}

// One test run. Written as JSON.
type Run struct {
	Repo       string  `json:"repo"`
	App        string  `json:"app"`
	Version    string  `json:"version"`
	StartedAt  string  `json:"started_at"`
	DurationMS int64   `json:"duration_ms"`
	Passed     int     `json:"passed"`
	Failed     int     `json:"failed"`
	Suites     []Suite `json:"suites"`
}

// Builds a run from suite results.
func Build(repo string, app string, version string, started time.Time, results []deployer.SuiteResult) Run {
	run := Run{
		Repo:      repo,
		App:       app,
		Version:   version,
		StartedAt: started.UTC().Format(time.RFC3339),
		Suites:    []Suite{},
	}
	for _, r := range results {
		run.Suites = append(run.Suites, Suite{
			Name:       r.Name,
			Passed:     r.Passed,
			DurationMS: r.Duration.Milliseconds(),
			Output:     r.Output,
		})
		if r.Passed {
			run.Passed++
		} else {
			run.Failed++
		}
	}
	run.DurationMS = time.Since(started).Milliseconds()
	return run
}

// Writes the JSON report.
func WriteJSON(path string, run Run) error {
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("encode json report: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		return fmt.Errorf("write json report: %w", err)
	}
	return nil
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Body    string `xml:",chardata"`
}

type junitTestcase struct {
	Name      string        `xml:"name,attr"`
	Classname string        `xml:"classname,attr"`
	Time      string        `xml:"time,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitTestsuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Time     string          `xml:"time,attr"`
	Cases    []junitTestcase `xml:"testcase"`
}

type junitTestsuites struct {
	XMLName  xml.Name         `xml:"testsuites"`
	Name     string           `xml:"name,attr"`
	Tests    int              `xml:"tests,attr"`
	Failures int              `xml:"failures,attr"`
	Time     string           `xml:"time,attr"`
	Suites   []junitTestsuite `xml:"testsuite"`
}

// Seconds with 3 decimals.
func seconds(ms int64) string {
	return fmt.Sprintf("%.3f", float64(ms)/1000)
}

// Builds the junit document.
func BuildJUnit(run Run) junitTestsuites {
	doc := junitTestsuites{
		Name:     run.App,
		Tests:    len(run.Suites),
		Failures: run.Failed,
		Time:     seconds(run.DurationMS),
		Suites:   []junitTestsuite{},
	}
	for _, s := range run.Suites {
		c := junitTestcase{Name: s.Name, Classname: run.App, Time: seconds(s.DurationMS)}
		if !s.Passed {
			c.Failure = &junitFailure{Message: "suite failed", Body: s.Output}
		}
		doc.Suites = append(doc.Suites, junitTestsuite{
			Name:     s.Name,
			Tests:    1,
			Failures: boolToInt(!s.Passed),
			Time:     seconds(s.DurationMS),
			Cases:    []junitTestcase{c},
		})
	}
	return doc
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Writes the junit XML report.
func WriteJUnit(path string, run Run) error {
	data, err := xml.MarshalIndent(BuildJUnit(run), "", "  ")
	if err != nil {
		return fmt.Errorf("encode junit report: %w", err)
	}
	out := append([]byte(xml.Header), data...)
	if err := os.WriteFile(path, append(out, '\n'), 0644); err != nil {
		return fmt.Errorf("write junit report: %w", err)
	}
	return nil
}
