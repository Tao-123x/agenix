package agenix

import (
	"encoding/json"
	"testing"
	"time"
)

func TestShellExecReportsStartFailureWithoutPanic(t *testing.T) {
	policy, err := NewPolicy(Permissions{
		Shell: ShellPermissions{Allow: []ShellCommand{{Run: []string{"agenix-missing-command"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	tools := NewTools(policy, NewTrace("test", "fake-scripted", Permissions{}))

	_, err = tools.ShellExec([]string{"agenix-missing-command"}, "", time.Second)
	if err == nil {
		t.Fatal("expected command start failure")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
}

func TestShellExecTraceRecordsRequestedAndResolvedCommands(t *testing.T) {
	policy, err := NewPolicy(Permissions{
		Shell: ShellPermissions{Allow: []ShellCommand{{Run: []string{"agenix-missing-command"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	trace := NewTrace("test", "fake-scripted", Permissions{})
	tools := NewTools(policy, trace)

	_, _ = tools.ShellExec([]string{"agenix-missing-command"}, "", time.Second)
	if len(trace.Events) != 1 {
		t.Fatalf("expected one trace event, got %d", len(trace.Events))
	}

	raw, err := json.Marshal(trace.Events[0].Request)
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	if request["cmd"] == nil || request["resolved_cmd"] == nil {
		t.Fatalf("trace request should include cmd and resolved_cmd: %#v", request)
	}
}
