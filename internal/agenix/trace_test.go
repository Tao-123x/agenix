package agenix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestTraceWriterPersistsRequiredShape(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{Network: false})
	trace.AddToolEvent("fs.read", map[string]any{"path": "demo.py"}, map[string]any{"content": "x"}, nil, 12)
	trace.AddVerifierEvent("output_schema_check", "schema", "passed", map[string]string{"type": "schema"}, ShellResult{}, nil)
	trace.SetFinal("passed", map[string]any{"changed_files": []string{"demo.py"}}, "")

	if err := WriteTrace(path, trace); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("trace is not valid JSON: %v", err)
	}
	if decoded["run_id"] == "" || decoded["skill"] != "repo.fix_test_failure" {
		t.Fatalf("missing required trace fields: %#v", decoded)
	}
	events := decoded["events"].([]any)
	if len(events) != 2 {
		t.Fatalf("events len = %d", len(events))
	}
	final := decoded["final"].(map[string]any)
	if final["status"] != "passed" {
		t.Fatalf("final status = %#v", final)
	}
}

func TestTraceWriterPersistsVerifierRequestFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{Network: false})
	trace.AddVerifierEvent(
		"run_tests",
		"command",
		"passed",
		map[string]any{
			"type":         "command",
			"cmd":          []string{"python3", "-m", "pytest", "-q"},
			"resolved_cmd": []string{"python3", "-m", "pytest", "-q"},
			"cwd":          "/tmp/repo",
			"timeout_ms":   120000,
		},
		ShellResult{ExitCode: 0},
		nil,
	)
	trace.SetFinal("passed", map[string]any{"changed_files": []string{"demo.py"}}, "")

	if err := WriteTrace(path, trace); err != nil {
		t.Fatal(err)
	}

	decoded, err := ReadTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(decoded.Events) != 1 {
		t.Fatalf("events len = %d", len(decoded.Events))
	}
	request, ok := decoded.Events[0].Request.(map[string]any)
	if !ok {
		t.Fatalf("request = %#v", decoded.Events[0].Request)
	}
	if request["cwd"] != "/tmp/repo" {
		t.Fatalf("request cwd = %#v", request)
	}
	if request["timeout_ms"] != float64(120000) {
		t.Fatalf("request timeout_ms = %#v", request)
	}
}
