package agenix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateTargetRecognizesCheckReportJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check-report.json")
	report := `{
  "kind": "check_report",
  "status": "passed",
  "skill": "repo.fix_test_failure",
  "version": "0.1.0",
  "artifact_path": "/tmp/repo.fix_test_failure-0.1.0.tar.gz",
  "run_id": "run-1",
  "trace_path": "/tmp/run-1.json",
  "changed_files": ["internal/agenix/check.go"],
  "verifier_summary": ["run_tests: passed"],
  "event_count": 3
}`
	if err := os.WriteFile(path, []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}

	kind, schemaPath, err := ValidateTarget(path)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "check_report" {
		t.Fatalf("expected check_report kind, got %q", kind)
	}
	if filepath.Base(schemaPath) != "check-report.schema.json" {
		t.Fatalf("expected check-report schema path, got %q", schemaPath)
	}
}

func TestValidateTargetRecognizesFailedCheckReportJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "failed-check-report.json")
	report := `{
  "kind": "check_report",
  "status": "failed",
  "skill": "repo.demo_skill",
  "version": "0.1.0",
  "artifact_path": "/tmp/repo.demo_skill-0.1.0.agenix",
  "run_id": "run-1",
  "trace_path": "/tmp/run-1.json",
  "changed_files": [],
  "verifier_summary": [],
  "event_count": 3,
  "error_class": "VerificationFailed",
  "error_message": "VerificationFailed: verifier run_tests failed"
}`
	if err := os.WriteFile(path, []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}

	kind, schemaPath, err := ValidateTarget(path)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "check_report" {
		t.Fatalf("expected check_report kind, got %q", kind)
	}
	if filepath.Base(schemaPath) != "check-report.schema.json" {
		t.Fatalf("expected check-report schema path, got %q", schemaPath)
	}
}

func TestValidateTargetRejectsInvalidCheckReportWithSchemaError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "check-report.json")
	report := `{
  "kind": "check_report",
  "status": "passed",
  "skill": "repo.fix_test_failure",
  "version": "0.1.0",
  "artifact_path": "/tmp/repo.fix_test_failure-0.1.0.tar.gz",
  "run_id": "run-1",
  "trace_path": "/tmp/run-1.json",
  "changed_files": ["internal/agenix/check.go"],
  "verifier_summary": ["run_tests: passed"]
}`
	if err := os.WriteFile(path, []byte(report), 0o600); err != nil {
		t.Fatal(err)
	}

	_, _, err := ValidateTarget(path)
	if err == nil {
		t.Fatal("expected schema validation failure")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "schema validation failed: $.event_count missing required field") {
		t.Fatalf("expected event_count schema error, got %v", err)
	}
}

func TestPublishedCheckReportSchemaAcceptsCheckResult(t *testing.T) {
	result := CheckResult{
		Kind:            CheckReportKind,
		Status:          "passed",
		Skill:           "repo.fix_test_failure",
		Version:         "0.1.0",
		ArtifactPath:    "/tmp/repo.fix_test_failure-0.1.0.tar.gz",
		RunID:           "run-1",
		TracePath:       "/tmp/run-1.json",
		ChangedFiles:    []string{"internal/agenix/check.go"},
		VerifierSummary: []string{"run_tests: passed"},
		EventCount:      3,
	}
	doc, err := structToDocument(result)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateSchemaDocument(schemaCheckReport, doc); err != nil {
		t.Fatalf("ValidateSchemaDocument returned error: %v", err)
	}
}
