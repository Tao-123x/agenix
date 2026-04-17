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
	trace := `{"run_id":"run-test","skill":"repo.fix_test_failure","model_profile":"fake-scripted","events":[{"type":"tool_call","name":"fs.read"},{"type":"verifier","name":"run_tests","status":"passed","exit_code":0}],"final":{"status":"passed","output":{"patch_summary":"done","changed_files":["fixture/mathlib.py"]}}}`
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
	if !strings.Contains(text, "event[0] type=tool_call name=fs.read") {
		t.Fatalf("missing replay event output: %s", text)
	}
	if !strings.Contains(text, "event[1] type=verifier name=run_tests status=passed exit_code=0") {
		t.Fatalf("missing verifier replay output: %s", text)
	}
	if !strings.Contains(text, `final_output={"changed_files":["fixture/mathlib.py"],"patch_summary":"done"}`) {
		t.Fatalf("missing final output replay output: %s", text)
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
	if !strings.Contains(string(inspectOut), "skill=repo.fix_test_failure") || !strings.Contains(string(inspectOut), "files=2") || !strings.Contains(string(inspectOut), "built_by=") {
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
    run: ["python3", "-m", "pytest", "-q"]
    cwd: ${repo_path}
    policy:
      executable: python3
      cwd: ${repo_path}
      timeout_ms: 120000
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

func TestCLIRunReadOnlyAnalyzeArtifactWithHeuristicAdapter(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.analyze_test_failures")
	artifact := filepath.Join(root, "analyze.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	runOut, err := exec.Command("go", "run", ".", "run", artifact, "--adapter", "heuristic-analyze").CombinedOutput()
	if err != nil {
		t.Fatalf("run artifact with heuristic adapter failed: %v\n%s", err, runOut)
	}
	text := string(runOut)
	if !strings.Contains(text, "status=passed") ||
		!strings.Contains(text, "verifiers=fixture_still_fails:passed,output_schema_check:passed") {
		t.Fatalf("unexpected run output: %s", text)
	}
}

func TestCLIRunRejectsUnknownAdapter(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.fix_test_failure")
	artifact := filepath.Join(root, "fix.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	runOut, err := exec.Command("go", "run", ".", "run", artifact, "--adapter", "missing-adapter").CombinedOutput()
	if err == nil {
		t.Fatalf("expected unknown adapter failure, got success: %s", runOut)
	}
	if !strings.Contains(string(runOut), "error=UnsupportedAdapter") {
		t.Fatalf("unexpected run error: %s", runOut)
	}
}

func TestCLIRunRejectsUnsupportedAdapterForSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.fix_test_failure")
	artifact := filepath.Join(root, "fix.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	runOut, err := exec.Command("go", "run", ".", "run", artifact, "--adapter", "heuristic-analyze").CombinedOutput()
	if err == nil {
		t.Fatalf("expected unsupported adapter failure, got success: %s", runOut)
	}
	if !strings.Contains(string(runOut), "error=UnsupportedAdapter") {
		t.Fatalf("unexpected run error: %s", runOut)
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
    run: ["python3", "-m", "pytest", "-q"]
    cwd: ${repo_path}
    policy:
      executable: python3
      cwd: ${repo_path}
      timeout_ms: 120000
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

func TestCLIRegistryListShowAndResolve(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "registry")

	firstSkill := filepath.Join(root, "first-skill")
	if err := os.MkdirAll(firstSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	firstManifest := `apiVersion: agenix/v0.1
kind: Skill
name: repo.alpha
version: 0.1.0
description: First registry entry.
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
	if err := os.WriteFile(filepath.Join(firstSkill, "manifest.yaml"), []byte(firstManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(firstSkill, "README.md"), []byte("# alpha\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	secondSkill := filepath.Join(root, "second-skill")
	if err := os.MkdirAll(secondSkill, 0o755); err != nil {
		t.Fatal(err)
	}
	secondManifest := strings.Replace(strings.Replace(firstManifest, "repo.alpha", "repo.beta", 1), "0.1.0", "0.2.0", 1)
	if err := os.WriteFile(filepath.Join(secondSkill, "manifest.yaml"), []byte(secondManifest), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(secondSkill, "README.md"), []byte("# beta\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	firstArtifact := filepath.Join(root, "alpha.agenix")
	if out, err := exec.Command("go", "run", ".", "build", firstSkill, "-o", firstArtifact).CombinedOutput(); err != nil {
		t.Fatalf("build alpha failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "run", ".", "publish", firstArtifact, "--registry", registry).CombinedOutput(); err != nil {
		t.Fatalf("publish alpha failed: %v\n%s", err, out)
	}

	secondArtifact := filepath.Join(root, "beta.agenix")
	if out, err := exec.Command("go", "run", ".", "build", secondSkill, "-o", secondArtifact).CombinedOutput(); err != nil {
		t.Fatalf("build beta failed: %v\n%s", err, out)
	}
	publishBetaOut, err := exec.Command("go", "run", ".", "publish", secondArtifact, "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("publish beta failed: %v\n%s", err, publishBetaOut)
	}

	listOut, err := exec.Command("go", "run", ".", "registry", "list", "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("registry list failed: %v\n%s", err, listOut)
	}
	listText := string(listOut)
	if !strings.Contains(listText, "skill=repo.alpha version=0.1.0") || !strings.Contains(listText, "skill=repo.beta version=0.2.0") {
		t.Fatalf("unexpected registry list output: %s", listText)
	}

	showOut, err := exec.Command("go", "run", ".", "registry", "show", "repo.beta", "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("registry show failed: %v\n%s", err, showOut)
	}
	if !strings.Contains(string(showOut), "skill=repo.beta version=0.2.0") {
		t.Fatalf("unexpected registry show output: %s", showOut)
	}

	digest := extractField(string(publishBetaOut), "digest")
	resolveOut, err := exec.Command("go", "run", ".", "registry", "resolve", digest, "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("registry resolve failed: %v\n%s", err, resolveOut)
	}
	if !strings.Contains(string(resolveOut), "registry_artifact=") || !strings.Contains(string(resolveOut), "digest="+digest) {
		t.Fatalf("unexpected registry resolve output: %s", resolveOut)
	}
}

func TestCLIRegistryCommandFailures(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "registry")
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "show-missing", args: []string{"registry", "show", "repo.missing", "--registry", registry}, want: "error=NotFound"},
		{name: "resolve-invalid", args: []string{"registry", "resolve", "repo.missing", "--registry", registry}, want: "error=InvalidInput"},
		{name: "resolve-missing", args: []string{"registry", "resolve", "repo.missing@0.1.0", "--registry", registry}, want: "error=NotFound"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commandArgs := append([]string{"run", "."}, tt.args...)
			out, err := exec.Command("go", commandArgs...).CombinedOutput()
			if err == nil {
				t.Fatalf("expected registry command failure, got success: %s", out)
			}
			if !strings.Contains(string(out), tt.want) {
				t.Fatalf("expected %q in %s", tt.want, out)
			}
		})
	}
}

func TestCLIRegistryShowSortsVersionsSemantically(t *testing.T) {
	root := t.TempDir()
	registry := filepath.Join(root, "registry")

	makeSkill := func(dir, version string) string {
		skillDir := filepath.Join(root, dir)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatal(err)
		}
		manifest := `apiVersion: agenix/v0.1
kind: Skill
name: repo.semver
version: ` + version + `
description: Semver ordering entry.
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
		if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "README.md"), []byte("# semver\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		return skillDir
	}

	first := makeSkill("skill-v10", "0.10.0")
	firstArtifact := filepath.Join(root, "v10.agenix")
	if out, err := exec.Command("go", "run", ".", "build", first, "-o", firstArtifact).CombinedOutput(); err != nil {
		t.Fatalf("build v10 failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "run", ".", "publish", firstArtifact, "--registry", registry).CombinedOutput(); err != nil {
		t.Fatalf("publish v10 failed: %v\n%s", err, out)
	}

	second := makeSkill("skill-v2", "0.2.0")
	secondArtifact := filepath.Join(root, "v2.agenix")
	if out, err := exec.Command("go", "run", ".", "build", second, "-o", secondArtifact).CombinedOutput(); err != nil {
		t.Fatalf("build v2 failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "run", ".", "publish", secondArtifact, "--registry", registry).CombinedOutput(); err != nil {
		t.Fatalf("publish v2 failed: %v\n%s", err, out)
	}

	showOut, err := exec.Command("go", "run", ".", "registry", "show", "repo.semver", "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("registry show failed: %v\n%s", err, showOut)
	}
	text := string(showOut)
	firstPos := strings.Index(text, "version=0.2.0")
	secondPos := strings.Index(text, "version=0.10.0")
	if firstPos == -1 || secondPos == -1 || firstPos > secondPos {
		t.Fatalf("unexpected semver show order: %s", text)
	}
}

func extractField(output, key string) string {
	for _, field := range strings.Fields(output) {
		prefix := key + "="
		if strings.HasPrefix(field, prefix) {
			return strings.TrimPrefix(field, prefix)
		}
	}
	return ""
}
