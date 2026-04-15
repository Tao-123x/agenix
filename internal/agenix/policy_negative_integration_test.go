package agenix

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

type shellMismatchAdapter struct{}

func (shellMismatchAdapter) Execute(_ Manifest, tools *Tools) (map[string]any, error) {
	_, err := tools.ShellExec([]string{"python3", "-m", "pip", "install", "pytest"}, "", 5*time.Second)
	return map[string]any{
		"patch_summary": "attempted disallowed shell command",
		"changed_files": []string{},
	}, err
}

func materializePolicyScenario(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("testdata", "policy_negative", name)
	dst := filepath.Join(t.TempDir(), name)
	copyDir(t, src, dst)
	return filepath.Join(dst, "manifest.yaml")
}

func readPolicyNegativeTrace(t *testing.T, path string) *Trace {
	t.Helper()
	trace, err := ReadTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	return trace
}

func traceEventNamed(t *testing.T, trace *Trace, eventType, name string) TraceEvent {
	t.Helper()
	for _, event := range trace.Events {
		if event.Type == eventType && event.Name == name {
			return event
		}
	}
	t.Fatalf("missing %s event %q in trace: %#v", eventType, name, trace.Events)
	return TraceEvent{}
}

func requestMap(t *testing.T, value interface{}) map[string]any {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	return request
}

func TestPolicyNegativeWriteScopeEscape(t *testing.T) {
	manifestPath := materializePolicyScenario(t, "write_scope_escape")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")
	outsidePath := filepath.Join(t.TempDir(), "outside.txt")

	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter:      EscapeAdapter{Path: outsidePath},
	})
	if err == nil {
		t.Fatal("expected policy violation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	if result.TracePath == "" {
		t.Fatal("expected trace path for policy failure")
	}

	trace := readPolicyNegativeTrace(t, result.TracePath)
	event := traceEventNamed(t, trace, "tool_call", "fs.write")
	if eventErrorClass(event.Error) != ErrPolicyViolation {
		t.Fatalf("expected fs.write policy violation, got %#v", event)
	}
}

func TestPolicyNegativeShellAllowlistMismatch(t *testing.T) {
	manifestPath := materializePolicyScenario(t, "shell_allowlist_mismatch")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")

	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter:      shellMismatchAdapter{},
	})
	if err == nil {
		t.Fatal("expected policy violation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	if result.TracePath == "" {
		t.Fatal("expected trace path for policy failure")
	}

	trace := readPolicyNegativeTrace(t, result.TracePath)
	event := traceEventNamed(t, trace, "tool_call", "shell.exec")
	if eventErrorClass(event.Error) != ErrPolicyViolation {
		t.Fatalf("expected shell.exec policy violation, got %#v", event)
	}
	request := requestMap(t, event.Request)
	if request["cmd"] == nil {
		t.Fatalf("missing cmd in shell request: %#v", request)
	}
}

func TestPolicyNegativeVerifierPolicyReject(t *testing.T) {
	manifestPath := materializePolicyScenario(t, "verifier_policy_reject")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")

	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
	})
	if err == nil {
		t.Fatal("expected invalid manifest")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
	if result.TracePath != "" {
		t.Fatalf("expected no trace path when manifest load fails, got %q", result.TracePath)
	}
}
