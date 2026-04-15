package agenix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifierRunsCommandAndChecksOutputSchema(t *testing.T) {
	repo := t.TempDir()
	manifest := Manifest{
		Name:    "repo.fix_test_failure",
		Outputs: OutputSchema{Required: []string{"patch_summary", "changed_files"}},
		Verifiers: []Verifier{
			{Type: "command", Name: "run_tests", Command: "python3 -c 'print(42)'", CWD: repo, Success: VerifierSuccess{ExitCode: 0}},
			{Type: "schema", Name: "output_schema_check", SchemaRef: "outputs"},
		},
	}
	trace := NewTrace(manifest.Name, "fake-scripted", Permissions{})
	output := map[string]any{"patch_summary": "fixed", "changed_files": []string{"bug.py"}}

	if err := RunVerifiers(manifest, output, trace); err != nil {
		t.Fatalf("RunVerifiers returned error: %v", err)
	}
	if len(trace.Events) != 2 {
		t.Fatalf("expected verifier events, got %d", len(trace.Events))
	}
}

func TestVerifierFailsWhenSchemaOutputMissing(t *testing.T) {
	manifest := Manifest{
		Name:    "repo.fix_test_failure",
		Outputs: OutputSchema{Required: []string{"patch_summary", "changed_files"}},
		Verifiers: []Verifier{
			{Type: "schema", Name: "output_schema_check", SchemaRef: "outputs"},
		},
	}
	trace := NewTrace(manifest.Name, "fake-scripted", Permissions{})

	err := RunVerifiers(manifest, map[string]any{"patch_summary": "fixed"}, trace)
	if err == nil {
		t.Fatal("expected schema verifier failure")
	}
	if !IsErrorClass(err, ErrVerificationFailed) {
		t.Fatalf("expected VerificationFailed, got %v", err)
	}
}

func TestVerifierReportsCommandFailure(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, "fail.py")
	if err := os.WriteFile(path, []byte("raise SystemExit(7)\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	manifest := Manifest{
		Name: "repo.fix_test_failure",
		Verifiers: []Verifier{
			{Type: "command", Name: "run_tests", Command: "python3 fail.py", CWD: repo, Success: VerifierSuccess{ExitCode: 0}},
		},
	}

	err := RunVerifiers(manifest, map[string]any{}, NewTrace(manifest.Name, "fake-scripted", Permissions{}))
	if err == nil {
		t.Fatal("expected command verifier failure")
	}
	if !IsErrorClass(err, ErrVerificationFailed) {
		t.Fatalf("expected VerificationFailed, got %v", err)
	}
}

func TestVerifierRunsStructuredCommandWithoutShellParsing(t *testing.T) {
	repo := t.TempDir()
	manifest := Manifest{
		Name: "repo.fix_test_failure",
		Verifiers: []Verifier{
			{Type: "command", Name: "run_tests", Run: []string{"python3", "-c", "print(42)"}, CWD: repo, Success: VerifierSuccess{ExitCode: 0}},
		},
	}
	trace := NewTrace(manifest.Name, "fake-scripted", Permissions{})

	if err := RunVerifiers(manifest, map[string]any{}, trace); err != nil {
		t.Fatalf("RunVerifiers returned error: %v", err)
	}
	if len(trace.Events) != 1 {
		t.Fatalf("expected one verifier event, got %d", len(trace.Events))
	}
}
