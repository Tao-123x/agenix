package agenix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadManifestRejectsImplementedMinimumMissingFields(t *testing.T) {
	valid := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
permissions:
  network: false
outputs:
  required:
    - patch_summary
verifiers:
  - type: schema
    name: output_schema_check
    schemaRef: outputs
`
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{name: "description", old: "description: Fix a failing pytest suite.\n", new: ""},
		{name: "tools", old: "tools:\n  - fs\n", new: ""},
		{name: "outputs.required", old: "outputs:\n  required:\n    - patch_summary\n", new: ""},
		{name: "verifiers", old: "verifiers:\n  - type: schema\n    name: output_schema_check\n    schemaRef: outputs\n", new: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "manifest.yaml")
			raw := strings.Replace(valid, tt.old, tt.new, 1)
			if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := LoadManifest(path)
			if err == nil {
				t.Fatal("expected InvalidInput error")
			}
			if !IsErrorClass(err, ErrInvalidInput) {
				t.Fatalf("expected InvalidInput, got %v", err)
			}
		})
	}
}

func TestTraceReaderRejectsImplementedMinimumMissingFields(t *testing.T) {
	valid := `{
  "run_id": "run-1",
  "skill": "repo.fix_test_failure",
  "model_profile": "fake-scripted",
  "events": [
    {"type": "tool_call", "name": "fs.read"}
  ],
  "final": {"status": "passed"}
}`
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{name: "run_id", old: `"run_id": "run-1",`, new: ""},
		{name: "skill", old: `"skill": "repo.fix_test_failure",`, new: ""},
		{name: "model_profile", old: `"model_profile": "fake-scripted",`, new: ""},
		{name: "final.status", old: `"final": {"status": "passed"}`, new: `"final": {}`},
		{name: "event.type", old: `{"type": "tool_call", "name": "fs.read"}`, new: `{"name": "fs.read"}`},
		{name: "event.name", old: `{"type": "tool_call", "name": "fs.read"}`, new: `{"type": "tool_call"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "trace.json")
			raw := strings.Replace(valid, tt.old, tt.new, 1)
			if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := ReadTrace(path)
			if err == nil {
				t.Fatal("expected InvalidInput error")
			}
			if !IsErrorClass(err, ErrInvalidInput) {
				t.Fatalf("expected InvalidInput, got %v", err)
			}
		})
	}
}

func TestVerifyRejectsMalformedTraceAsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(path, []byte(`{"skill":"repo.fix_test_failure","model_profile":"fake-scripted","final":{"status":"passed"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Verify(path)
	if err == nil {
		t.Fatal("expected InvalidInput error")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
}

func TestReplayRejectsMalformedTraceAsInvalidInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(path, []byte(`{"run_id":"run-1","model_profile":"fake-scripted","final":{"status":"passed"}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := Replay(path)
	if err == nil {
		t.Fatal("expected InvalidInput error")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
}

func TestLoadManifestRejectsStructuredVerifierWithoutPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	raw := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
outputs:
  required:
    - patch_summary
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: fixture
    success:
      exit_code: 0
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadManifest(path)
	if err == nil {
		t.Fatal("expected InvalidInput error")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
}

func TestLoadManifestRejectsStructuredVerifierPolicyMissingFields(t *testing.T) {
	base := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
outputs:
  required:
    - patch_summary
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: fixture
    policy:
      executable: python3
      cwd: fixture
      timeout_ms: 120000
    success:
      exit_code: 0
`
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{name: "policy", old: "    policy:\n      executable: python3\n      cwd: fixture\n      timeout_ms: 120000\n", new: ""},
		{name: "policy.executable", old: "      executable: python3\n", new: ""},
		{name: "policy.cwd", old: "      cwd: fixture\n", new: ""},
		{name: "policy.timeout_ms", old: "      timeout_ms: 120000\n", new: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "manifest.yaml")
			raw := strings.Replace(base, tt.old, tt.new, 1)
			if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := LoadManifest(path)
			if err == nil {
				t.Fatal("expected InvalidInput error")
			}
			if !IsErrorClass(err, ErrInvalidInput) {
				t.Fatalf("expected InvalidInput, got %v", err)
			}
		})
	}
}

func TestPublishedManifestSchemaAcceptsLoadedManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	raw := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
outputs:
  required:
    - patch_summary
verifiers:
  - type: schema
    name: output_schema_check
    schemaRef: outputs
`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateManifestDocument(manifest); err != nil {
		t.Fatalf("ValidateManifestDocument returned error: %v", err)
	}
}

func TestPublishedTraceSchemaAcceptsReadTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	trace := `{
  "run_id": "run-1",
  "skill": "repo.fix_test_failure",
  "model_profile": "fake-scripted",
  "started_at": "2026-04-16T00:00:00Z",
  "policy": {"network": false},
  "events": [
    {"type": "tool_call", "name": "fs.read"}
  ],
  "final": {"status": "passed"}
}`
	if err := os.WriteFile(path, []byte(trace), 0o600); err != nil {
		t.Fatal(err)
	}

	decoded, err := ReadTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateTraceDocument(*decoded); err != nil {
		t.Fatalf("ValidateTraceDocument returned error: %v", err)
	}
}

func TestPublishedManifestSchemaRejectsWrongType(t *testing.T) {
	manifest := Manifest{
		APIVersion:  "agenix/v0.1",
		Kind:        "Skill",
		Name:        "repo.fix_test_failure",
		Version:     "0.1.0",
		Description: "Fix a failing pytest suite.",
		Tools:       []string{"fs"},
		Outputs:     OutputSchema{Required: []string{"patch_summary"}},
		Verifiers:   []Verifier{{Type: "schema", Name: "output_schema_check", SchemaRef: "outputs"}},
	}

	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["tools"] = "fs"

	if err := ValidateSchemaDocument(schemaManifest, doc); err == nil {
		t.Fatal("expected schema validation failure")
	} else if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
}
