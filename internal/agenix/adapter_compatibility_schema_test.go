package agenix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateTargetRecognizesAdapterCompatibilityReportJSON(t *testing.T) {
	report, err := CheckBuiltinAdapterCompatibility(AdapterCompatibilityOptions{
		Target: filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote", "manifest.yaml"),
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "adapter-compatibility-report.json")
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}

	kind, schemaPath, err := ValidateTarget(path)
	if err != nil {
		t.Fatal(err)
	}
	if kind != "adapter_compatibility_report" {
		t.Fatalf("expected adapter_compatibility_report kind, got %q", kind)
	}
	if filepath.Base(schemaPath) != "adapter-compatibility-report.schema.json" {
		t.Fatalf("expected adapter compatibility schema path, got %q", schemaPath)
	}
}

func TestValidateTargetRejectsInvalidAdapterCompatibilityReportWithSchemaError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "adapter-compatibility-report.json")
	report := `{
  "kind": "adapter_compatibility_report",
  "target": "examples/repo.fix_test_failure/manifest.yaml",
  "skill": "repo.fix_test_failure",
  "version": "0.1.0"
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
	if !strings.Contains(err.Error(), "schema validation failed: $.adapters missing required field") {
		t.Fatalf("expected adapters schema error, got %v", err)
	}
}

func TestPublishedAdapterCompatibilitySchemaAcceptsReport(t *testing.T) {
	report, err := CheckBuiltinAdapterCompatibility(AdapterCompatibilityOptions{
		Target: filepath.Join("..", "..", "examples", "repo.fix_test_failure", "manifest.yaml"),
	})
	if err != nil {
		t.Fatal(err)
	}
	doc, err := structToDocument(report)
	if err != nil {
		t.Fatal(err)
	}

	if err := ValidateSchemaDocument(schemaAdapterCompatibilityReport, doc); err != nil {
		t.Fatalf("ValidateSchemaDocument returned error: %v", err)
	}
}
