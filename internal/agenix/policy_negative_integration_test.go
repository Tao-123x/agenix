package agenix

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

type negativeWriteAdapter struct {
	path string
}

func (a negativeWriteAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "negative-write",
		ModelProfile:    "negative-write",
		SupportedSkills: []string{"policy_negative.write_scope_escape"},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 32000,
			ReasoningLevel:   "medium",
		},
	}
}

func (a negativeWriteAdapter) Execute(_ Manifest, tools *Tools) (map[string]any, error) {
	err := tools.FSWrite(a.path, "escape", true)
	return map[string]any{
		"patch_summary": "attempted out-of-scope write",
		"changed_files": []string{a.path},
	}, err
}

type shellMismatchAdapter struct{}

func (shellMismatchAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "shell-mismatch",
		ModelProfile:    "shell-mismatch",
		SupportedSkills: []string{"policy_negative.shell_allowlist_mismatch"},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 32000,
			ReasoningLevel:   "medium",
		},
	}
}

func (shellMismatchAdapter) Execute(_ Manifest, tools *Tools) (map[string]any, error) {
	_, err := tools.ShellExec([]string{"python3", "-m", "pip", "install", "pytest"}, "", 5*time.Second)
	return map[string]any{
		"patch_summary": "attempted disallowed shell command",
		"changed_files": []string{},
	}, err
}

type staticOutputAdapter struct {
	skill  string
	output map[string]any
}

func (a staticOutputAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "static-output",
		ModelProfile:    "static-output",
		SupportedSkills: []string{a.skill},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 32000,
			ReasoningLevel:   "medium",
		},
	}
}

func (a staticOutputAdapter) Execute(_ Manifest, _ *Tools) (map[string]any, error) {
	return a.output, nil
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
		Adapter:      negativeWriteAdapter{path: outsidePath},
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
		Adapter: staticOutputAdapter{
			skill: "policy_negative.verifier_policy_reject",
			output: map[string]any{
				"patch_summary": "noop",
				"changed_files": []string{},
			},
		},
	})
	if err == nil {
		t.Fatal("expected verifier policy violation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	if result.TracePath == "" {
		t.Fatal("expected trace path for verifier policy failure")
	}

	trace := readPolicyNegativeTrace(t, result.TracePath)
	event := traceEventNamed(t, trace, "verifier", "run_tests")
	if event.Status != "failed" {
		t.Fatalf("expected failed verifier event, got %#v", event)
	}
	if eventErrorClass(event.Error) != ErrPolicyViolation {
		t.Fatalf("expected verifier policy violation, got %#v", event)
	}
	request := requestMap(t, event.Request)
	for _, key := range []string{"cmd", "resolved_cmd", "cwd", "timeout_ms"} {
		if request[key] == nil {
			t.Fatalf("missing verifier request field %q in %#v", key, request)
		}
	}
}
