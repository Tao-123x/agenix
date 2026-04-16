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

func TestCLIBuildAndInspect(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
inputs:
  repo_path: fixture
outputs:
  required:
    - patch_summary
verifiers:
  - type: schema
    name: output_schema_check
    schemaRef: outputs
`
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "README.md"), []byte("# demo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(root, "skill.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}
	if !strings.Contains(string(buildOut), "artifact=") || !strings.Contains(string(buildOut), "digest=sha256:") {
		t.Fatalf("unexpected build output: %s", buildOut)
	}

	inspectOut, err := exec.Command("go", "run", ".", "inspect", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("inspect failed: %v\n%s", err, inspectOut)
	}
	if !strings.Contains(string(inspectOut), "skill=repo.fix_test_failure") || !strings.Contains(string(inspectOut), "files=2") {
		t.Fatalf("unexpected inspect output: %s", inspectOut)
	}
}

func TestCLIRunAcceptsArtifact(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill")
	fixture := filepath.Join(skillDir, "fixture")
	if err := os.MkdirAll(fixture, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
inputs:
  repo_path: fixture
outputs:
  required:
    - patch_summary
    - changed_files
verifiers:
  - type: command
    name: run_tests
    cmd: "python3 -m pytest -q"
    cwd: ${repo_path}
    success:
      exit_code: 0
  - type: schema
    name: output_schema_check
    schemaRef: outputs
`
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "mathlib.py"), []byte("def add(a, b):\n    return a - b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "test_mathlib.py"), []byte("from mathlib import add\n\n\ndef test_adds_numbers():\n    assert add(2, 3) == 5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(root, "skill.agenix")
	if out, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	if err := os.RemoveAll(skillDir); err != nil {
		t.Fatal(err)
	}

	runOut, err := exec.Command("go", "run", ".", "run", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("run artifact failed: %v\n%s", err, runOut)
	}
	text := string(runOut)
	if !strings.Contains(text, "status=passed") || !strings.Contains(text, "verifiers=run_tests:passed,output_schema_check:passed") {
		t.Fatalf("unexpected run output: %s", text)
	}
}

func TestCLIRunReadOnlyAnalyzeArtifact(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.analyze_test_failures")
	artifact := filepath.Join(root, "analyze.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	runOut, err := exec.Command("go", "run", ".", "run", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("run artifact failed: %v\n%s", err, runOut)
	}
	text := string(runOut)
	if !strings.Contains(text, "status=passed") ||
		!strings.Contains(text, "changed_files= ") ||
		!strings.Contains(text, "verifiers=fixture_still_fails:passed,output_schema_check:passed") {
		t.Fatalf("unexpected run output: %s", text)
	}
}

func TestCLIRunSmallRefactorArtifact(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.apply_small_refactor")
	artifact := filepath.Join(root, "refactor.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	runOut, err := exec.Command("go", "run", ".", "run", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("run artifact failed: %v\n%s", err, runOut)
	}
	text := string(runOut)
	if !strings.Contains(text, "status=passed") ||
		!strings.Contains(text, "greeter.py") ||
		!strings.Contains(text, "verifiers=run_tests:passed,refactor_shape:passed,output_schema_check:passed") {
		t.Fatalf("unexpected run output: %s", text)
	}
}

func TestCLIPublishAndPullArtifact(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: agenix/v0.1
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
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "README.md"), []byte("# demo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(root, "skill.agenix")
	registry := filepath.Join(root, "registry")
	pulled := filepath.Join(root, "pulled.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}
	publishOut, err := exec.Command("go", "run", ".", "publish", artifact, "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("publish failed: %v\n%s", err, publishOut)
	}
	if !strings.Contains(string(publishOut), "registry_artifact=") || !strings.Contains(string(publishOut), "digest=sha256:") {
		t.Fatalf("unexpected publish output: %s", publishOut)
	}

	pullOut, err := exec.Command("go", "run", ".", "pull", "repo.fix_test_failure@0.1.0", "-o", pulled, "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("pull failed: %v\n%s", err, pullOut)
	}
	if !strings.Contains(string(pullOut), "artifact="+pulled) || !strings.Contains(string(pullOut), "skill=repo.fix_test_failure") {
		t.Fatalf("unexpected pull output: %s", pullOut)
	}
}

func TestCLIInspectAcceptsRegistryReference(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: agenix/v0.1
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
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "README.md"), []byte("# demo\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(root, "skill.agenix")
	registry := filepath.Join(root, "registry")

	if out, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "run", ".", "publish", artifact, "--registry", registry).CombinedOutput(); err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out)
	}

	inspectOut, err := exec.Command("go", "run", ".", "inspect", "repo.fix_test_failure@0.1.0", "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("inspect failed: %v\n%s", err, inspectOut)
	}
	if !strings.Contains(string(inspectOut), "skill=repo.fix_test_failure") || !strings.Contains(string(inspectOut), "digest=sha256:") {
		t.Fatalf("unexpected inspect output: %s", inspectOut)
	}
}

func TestCLIRunAcceptsRegistryReference(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill")
	fixture := filepath.Join(skillDir, "fixture")
	if err := os.MkdirAll(fixture, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
inputs:
  repo_path: fixture
outputs:
  required:
    - patch_summary
    - changed_files
verifiers:
  - type: command
    name: run_tests
    cmd: "python3 -m pytest -q"
    cwd: ${repo_path}
    success:
      exit_code: 0
  - type: schema
    name: output_schema_check
    schemaRef: outputs
`
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "mathlib.py"), []byte("def add(a, b):\n    return a - b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fixture, "test_mathlib.py"), []byte("from mathlib import add\n\n\ndef test_adds_numbers():\n    assert add(2, 3) == 5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact := filepath.Join(root, "skill.agenix")
	registry := filepath.Join(root, "registry")
	if out, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "run", ".", "publish", artifact, "--registry", registry).CombinedOutput(); err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out)
	}
	if err := os.RemoveAll(skillDir); err != nil {
		t.Fatal(err)
	}

	runOut, err := exec.Command("go", "run", ".", "run", "repo.fix_test_failure@0.1.0", "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("run registry ref failed: %v\n%s", err, runOut)
	}
	text := string(runOut)
	if !strings.Contains(text, "status=passed") || !strings.Contains(text, "verifiers=run_tests:passed,output_schema_check:passed") {
		t.Fatalf("unexpected run output: %s", text)
	}
}

func TestCLIInspectRegistryReferenceFailures(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "registry")
	tests := []struct {
		name      string
		ref       string
		wantError string
	}{
		{name: "invalid-syntax", ref: "repo.fix_test_failure", wantError: "error=InvalidInput"},
		{name: "missing-entry", ref: "repo.missing@0.1.0", wantError: "error=NotFound"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := exec.Command("go", "run", ".", "inspect", tt.ref, "--registry", registry).CombinedOutput()
			if err == nil {
				t.Fatalf("expected inspect failure, got success: %s", out)
			}
			if !strings.Contains(string(out), tt.wantError) {
				t.Fatalf("expected %q in %s", tt.wantError, out)
			}
		})
	}
}

func TestCLIRunRegistryReferenceFailures(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "registry")
	tests := []struct {
		name      string
		ref       string
		wantError string
	}{
		{name: "invalid-syntax", ref: "repo.fix_test_failure", wantError: "error=InvalidInput"},
		{name: "missing-entry", ref: "repo.missing@0.1.0", wantError: "error=NotFound"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := exec.Command("go", "run", ".", "run", tt.ref, "--registry", registry).CombinedOutput()
			if err == nil {
				t.Fatalf("expected run failure, got success: %s", out)
			}
			if !strings.Contains(string(out), tt.wantError) {
				t.Fatalf("expected %q in %s", tt.wantError, out)
			}
		})
	}
}

func TestCLIValidateManifestAndTrace(t *testing.T) {
	root := t.TempDir()
	manifestPath := filepath.Join(root, "manifest.yaml")
	tracePath := filepath.Join(root, "trace.json")
	manifest := `apiVersion: agenix/v0.1
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
	trace := `{
  "run_id": "run-1",
  "skill": "repo.fix_test_failure",
  "model_profile": "fake-scripted",
  "started_at": "2026-04-16T00:00:00Z",
  "policy": {"network": false},
  "events": [{"type":"tool_call","name":"fs.read"}],
  "final": {"status": "passed"}
}`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tracePath, []byte(trace), 0o600); err != nil {
		t.Fatal(err)
	}

	manifestOut, err := exec.Command("go", "run", ".", "validate", manifestPath).CombinedOutput()
	if err != nil {
		t.Fatalf("validate manifest failed: %v\n%s", err, manifestOut)
	}
	if !strings.Contains(string(manifestOut), "kind=manifest") || !strings.Contains(string(manifestOut), "status=valid") {
		t.Fatalf("unexpected validate manifest output: %s", manifestOut)
	}

	traceOut, err := exec.Command("go", "run", ".", "validate", tracePath).CombinedOutput()
	if err != nil {
		t.Fatalf("validate trace failed: %v\n%s", err, traceOut)
	}
	if !strings.Contains(string(traceOut), "kind=trace") || !strings.Contains(string(traceOut), "status=valid") {
		t.Fatalf("unexpected validate trace output: %s", traceOut)
	}
}

func TestCLIValidateRejectsInvalidManifest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(path, []byte("apiVersion: agenix/v0.1\nkind: Skill\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("go", "run", ".", "validate", path).CombinedOutput()
	if err == nil {
		t.Fatalf("expected validate failure, got success: %s", out)
	}
	if !strings.Contains(string(out), "error=InvalidInput") {
		t.Fatalf("unexpected validate error: %s", out)
	}
}

func TestCLIValidateRejectsInvalidTrace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	if err := os.WriteFile(path, []byte(`{"run_id":"run-1","skill":"repo.fix_test_failure","model_profile":"fake-scripted","final":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("go", "run", ".", "validate", path).CombinedOutput()
	if err == nil {
		t.Fatalf("expected validate failure, got success: %s", out)
	}
	if !strings.Contains(string(out), "error=InvalidInput") {
		t.Fatalf("unexpected validate error: %s", out)
	}
}
