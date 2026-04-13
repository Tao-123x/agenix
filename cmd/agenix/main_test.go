package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIReplayPrintsSummary(t *testing.T) {
	root := t.TempDir()
	tracePath := filepath.Join(root, "trace.json")
	trace := `{"run_id":"run-test","skill":"repo.fix_test_failure","model_profile":"fake-scripted","events":[{"type":"tool_call","name":"fs.read"}],"final":{"status":"passed"}}`
	if err := os.WriteFile(tracePath, []byte(trace), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", ".", "replay", tracePath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("go run replay failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "skill=repo.fix_test_failure") || !strings.Contains(text, "status=passed") {
		t.Fatalf("unexpected replay output: %s", text)
	}
}

func TestFormatRunResultIncludesVerifierSummary(t *testing.T) {
	out := formatRunResult("passed", "run-1", "trace.json", []string{"a.py"}, []string{"run_tests:passed", "output_schema_check:passed"})
	if !strings.Contains(out, "verifiers=run_tests:passed,output_schema_check:passed") {
		t.Fatalf("missing verifier summary: %s", out)
	}
}
