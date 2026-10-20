package deployer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// One row from compose ps.
// Mirrors the v2 JSON fields.
type ServiceState struct {
	Name    string `json:"Name"`
	Service string `json:"Service"`
	State   string `json:"State"`
	Health  string `json:"Health"`
}

// Parses compose ps JSON output.
// Handles array and per-line forms.
func ParsePsOutput(raw string) ([]ServiceState, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	if strings.HasPrefix(trimmed, "[") {
		var arr []ServiceState
		if err := json.Unmarshal([]byte(trimmed), &arr); err != nil {
			return nil, fmt.Errorf("bad ps json: %w", err)
		}
		return arr, nil
	}
	states := []ServiceState{}
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		// Skip compose warnings and notes.
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var st ServiceState
		if err := json.Unmarshal([]byte(line), &st); err != nil {
			return nil, fmt.Errorf("bad ps line: %w", err)
		}
		states = append(states, st)
	}
	return states, nil
}

// Ready when running and healthy.
// Blank health means no check.
func EvaluateReady(states []ServiceState) (bool, []string) {
	if len(states) == 0 {
		return false, []string{"no containers"}
	}
	pending := []string{}
	for _, st := range states {
		label := st.Service
		if label == "" {
			label = st.Name
		}
		if !strings.EqualFold(st.State, "running") {
			pending = append(pending, fmt.Sprintf("%s: %s", label, st.State))
			continue
		}
		health := strings.ToLower(st.Health)
		if health != "" && health != "healthy" {
			pending = append(pending, fmt.Sprintf("%s: %s", label, health))
		}
	}
	return len(pending) == 0, pending
}

// Polls until ready or timeout.
// Fetch is injected for tests.
func WaitReady(fetch func() ([]ServiceState, error), timeout time.Duration, poll time.Duration) error {
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	if poll <= 0 {
		poll = time.Second
	}
	deadline := time.Now().Add(timeout)
	pending := []string{}
	for {
		states, err := fetch()
		if err != nil {
			return err
		}
		ok, waiting := EvaluateReady(states)
		if ok {
			return nil
		}
		pending = waiting
		if time.Now().After(deadline) {
			return fmt.Errorf("services not ready after %s: %s", timeout, strings.Join(pending, ", "))
		}
		time.Sleep(poll)
	}
}

// Waits for compose services to be ready.
func WaitComposeReady(repoPath string, files []string, timeout time.Duration) error {
	fetch := func() ([]ServiceState, error) {
		out, err := composePSOutput(repoPath, files)
		if err != nil {
			return nil, err
		}
		return ParsePsOutput(out)
	}
	return WaitReady(fetch, timeout, time.Second)
}
