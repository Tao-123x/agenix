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
	trace.AddVerifierEvent("output_schema_check", "schema", "passed", map[string]any{"type": "schema"}, ShellResult{}, nil)
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
	verifier := events[1].(map[string]any)
	request := verifier["request"].(map[string]any)
	if request["type"] != "schema" {
		t.Fatalf("verifier request = %#v", request)
	}
	final := decoded["final"].(map[string]any)
	if final["status"] != "passed" {
		t.Fatalf("final status = %#v", final)
	}
}
