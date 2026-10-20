package deployer

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestParsePsOutputArray(t *testing.T) {
	raw := `[{"Name":"a-1","Service":"a","State":"running","Health":"healthy"}]`
	states, err := ParsePsOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 1 || states[0].Service != "a" || states[0].Health != "healthy" {
		t.Errorf("wrong parse: %+v", states)
	}
}

func TestParsePsOutputLines(t *testing.T) {
	raw := "{\"Service\":\"a\",\"State\":\"running\",\"Health\":\"\"}\n{\"Service\":\"b\",\"State\":\"running\",\"Health\":\"healthy\"}\n"
	states, err := ParsePsOutput(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 2 {
		t.Fatalf("expected 2 states, got %d", len(states))
	}
	if states[1].Service != "b" {
		t.Errorf("wrong second service: %+v", states[1])
	}
}

func TestParsePsOutputEmpty(t *testing.T) {
	states, err := ParsePsOutput("  \n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(states) != 0 {
		t.Errorf("expected empty, got %+v", states)
	}
}

func TestParsePsOutputBad(t *testing.T) {
	if _, err := ParsePsOutput("{not json}"); err == nil {
		t.Error("expected error for bad line")
	}
	if _, err := ParsePsOutput("[not json]"); err == nil {
		t.Error("expected error for bad array")
	}
}

func TestEvaluateReadyHealthy(t *testing.T) {
	ok, pending := EvaluateReady([]ServiceState{
		{Service: "a", State: "running", Health: "healthy"},
		{Service: "b", State: "running"},
	})
	if !ok {
		t.Errorf("expected ready, pending %v", pending)
	}
}

func TestEvaluateReadyStarting(t *testing.T) {
	ok, pending := EvaluateReady([]ServiceState{
		{Service: "db", State: "running", Health: "starting"},
	})
	if ok {
		t.Error("expected not ready for starting")
	}
	if len(pending) != 1 || !strings.Contains(pending[0], "starting") {
		t.Errorf("wrong pending: %v", pending)
	}
}

func TestEvaluateReadyNotRunning(t *testing.T) {
	ok, pending := EvaluateReady([]ServiceState{
		{Service: "a", State: "exited", Health: ""},
	})
	if ok {
		t.Error("expected not ready for exited")
	}
	if !strings.Contains(strings.Join(pending, " "), "exited") {
		t.Errorf("wrong pending: %v", pending)
	}
}

func TestEvaluateReadyNoContainers(t *testing.T) {
	ok, pending := EvaluateReady(nil)
	if ok {
		t.Error("expected not ready with no containers")
	}
	if len(pending) != 1 {
		t.Errorf("expected one pending note, got %v", pending)
	}
}

func TestWaitReadySucceedsAfterPolls(t *testing.T) {
	calls := 0
	fetch := func() ([]ServiceState, error) {
		calls++
		if calls < 3 {
			return []ServiceState{{Service: "a", State: "running", Health: "starting"}}, nil
		}
		return []ServiceState{{Service: "a", State: "running", Health: "healthy"}}, nil
	}
	if err := WaitReady(fetch, 2*time.Second, time.Millisecond); err != nil {
		t.Fatalf("expected ready, got %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 polls, got %d", calls)
	}
}

func TestWaitReadyTimeout(t *testing.T) {
	fetch := func() ([]ServiceState, error) {
		return []ServiceState{{Service: "db", State: "running", Health: "starting"}}, nil
	}
	err := WaitReady(fetch, 10*time.Millisecond, time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
	if !strings.Contains(err.Error(), "db") {
		t.Errorf("error should name service: %v", err)
	}
}

func TestWaitReadyFetchError(t *testing.T) {
	fetch := func() ([]ServiceState, error) {
		return nil, errors.New("daemon down")
	}
	if err := WaitReady(fetch, time.Second, time.Millisecond); err == nil {
		t.Error("expected fetch error")
	}
}
