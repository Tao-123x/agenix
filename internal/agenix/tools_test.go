package agenix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestShellExecReportsStartFailureWithoutPanic(t *testing.T) {
	policy, err := NewPolicy(Permissions{
		Network: true,
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
		Network: true,
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

func TestShellExecRejectsAllowlistedCommandWhenNetworkDisabled(t *testing.T) {
	policy, err := NewPolicy(Permissions{
		Network: true,
		Shell:   ShellPermissions{Allow: []ShellCommand{{Run: []string{"curl", "https://example.com"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	policy.permissions.Network = false
	trace := NewTrace("test", "fake-scripted", policy.permissions)
	tools := NewTools(policy, trace)

	called := 0
	original := execCommandRunner
	execCommandRunner = func(argv []string, cwd string, timeout time.Duration, env []string) (ShellResult, error) {
		called++
		return ShellResult{}, nil
	}
	t.Cleanup(func() {
		execCommandRunner = original
	})

	_, err = tools.ShellExec([]string{"curl", "https://example.com"}, "", time.Second)
	if err == nil {
		t.Fatal("expected PolicyViolation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	if called != 0 {
		t.Fatalf("expected runner not to be called, got %d", called)
	}
	if len(trace.Events) != 1 {
		t.Fatalf("expected one trace event, got %d", len(trace.Events))
	}
}

func TestShellExecMapsPythonNetworkAttemptToPolicyViolationWhenNetworkDisabled(t *testing.T) {
	repo := t.TempDir()
	scriptPath := filepath.Join(repo, "network_attempt.py")
	script := []byte("import socket\nsocket.create_connection(('127.0.0.1', 1), 0.1)\n")
	if err := os.WriteFile(scriptPath, script, 0o600); err != nil {
		t.Fatal(err)
	}

	policy, err := NewPolicy(Permissions{
		Network: false,
		Shell:   ShellPermissions{Allow: []ShellCommand{{Run: []string{"python3", "network_attempt.py"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	trace := NewTrace("test", "fake-scripted", policy.permissions)
	tools := NewTools(policy, trace)

	_, err = tools.ShellExec([]string{"python3", "network_attempt.py"}, repo, 2*time.Second)
	if err == nil {
		t.Fatal("expected PolicyViolation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
}
