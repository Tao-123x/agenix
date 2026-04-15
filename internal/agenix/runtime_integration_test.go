package agenix

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeRunsCanonicalFixTestFailureSkill(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, true)
	manifestPath := writeManifest(t, root, repo)
	runDir := filepath.Join(root, ".agenix-runs")

	result, err := Run(RunOptions{ManifestPath: manifestPath, RunDir: runDir})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("status = %q", result.Status)
	}
	if len(result.ChangedFiles) != 1 || result.ChangedFiles[0] != filepath.Join(repo, "mathlib.py") {
		t.Fatalf("changed files = %#v", result.ChangedFiles)
	}
	if result.TracePath == "" {
		t.Fatal("missing trace path")
	}

	raw, err := os.ReadFile(result.TracePath)
	if err != nil {
		t.Fatal(err)
	}
	var trace Trace
	if err := json.Unmarshal(raw, &trace); err != nil {
		t.Fatal(err)
	}
	if trace.Final.Status != "passed" {
		t.Fatalf("trace final = %#v", trace.Final)
	}
	if !traceHasEvent(trace, "tool_call", "fs.write") {
		t.Fatalf("trace does not contain fs.write event: %#v", trace.Events)
	}
	if !traceHasVerifier(trace, "run_tests", "passed") {
		t.Fatalf("trace does not contain passing run_tests verifier: %#v", trace.Events)
	}
	if !traceHasAdapterEvent(trace, "selection", "ok") {
		t.Fatalf("trace does not contain adapter selection event: %#v", trace.Events)
	}
	if !traceHasAdapterEvent(trace, "capability_check", "ok") {
		t.Fatalf("trace does not contain adapter capability_check event: %#v", trace.Events)
	}
	if !traceHasVerifierRequestField(trace, "run_tests", "resolved_cmd") {
		t.Fatalf("trace does not contain structured verifier request: %#v", trace.Events)
	}
}

func TestRuntimeRunsMovableArtifactCapsule(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill")
	writePythonFixture(t, skillDir, true)
	writeManifestAt(t, filepath.Join(skillDir, "manifest.yaml"), "repo")
	artifact := filepath.Join(root, "skill.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifact}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(skillDir); err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(root, ".agenix-runs")

	result, err := Run(RunOptions{ManifestPath: artifact, RunDir: runDir})
	if err != nil {
		t.Fatalf("Run artifact returned error: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("status = %q", result.Status)
	}
	if !strings.Contains(result.TracePath, runDir) {
		t.Fatalf("trace path %q is not under run dir %q", result.TracePath, runDir)
	}
	if len(result.ChangedFiles) != 1 || !strings.Contains(result.ChangedFiles[0], filepath.Join("workspace", "repo", "mathlib.py")) {
		t.Fatalf("changed files = %#v", result.ChangedFiles)
	}

	verifyResult, err := Verify(result.TracePath)
	if err != nil {
		t.Fatalf("Verify artifact run returned error: %v", err)
	}
	if verifyResult.Status != "passed" {
		t.Fatalf("verify status = %q", verifyResult.Status)
	}
}

func TestArtifactTraceStoresAbsoluteManifestPathForCrossCWDVerify(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill")
	writePythonFixture(t, skillDir, true)
	writeManifestAt(t, filepath.Join(skillDir, "manifest.yaml"), "repo")
	artifact := filepath.Join(root, "skill.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifact}); err != nil {
		t.Fatal(err)
	}

	originalCWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalCWD); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	}()

	result, err := Run(RunOptions{ManifestPath: artifact})
	if err != nil {
		t.Fatalf("Run artifact returned error: %v", err)
	}
	absTracePath, err := filepath.Abs(result.TracePath)
	if err != nil {
		t.Fatal(err)
	}
	trace, err := ReadTrace(absTracePath)
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(trace.ManifestPath) {
		t.Fatalf("trace manifest path should be absolute, got %q", trace.ManifestPath)
	}

	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	verifyResult, err := Verify(absTracePath)
	if err != nil {
		t.Fatalf("Verify from another cwd returned error: %v", err)
	}
	if verifyResult.Status != "passed" {
		t.Fatalf("verify status = %q", verifyResult.Status)
	}
}

func TestRuntimeRunsReadOnlyAnalyzeTestFailuresSkill(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "examples", "repo.analyze_test_failures", "manifest.yaml")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")

	result, err := Run(RunOptions{ManifestPath: manifestPath, RunDir: runDir})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("status = %q", result.Status)
	}
	if len(result.ChangedFiles) != 0 {
		t.Fatalf("read-only skill should not report changed files: %#v", result.ChangedFiles)
	}

	trace, err := ReadTrace(result.TracePath)
	if err != nil {
		t.Fatal(err)
	}
	if traceHasEvent(*trace, "tool_call", "fs.write") {
		t.Fatalf("read-only skill emitted fs.write event: %#v", trace.Events)
	}
	output, ok := trace.Final.Output.(map[string]any)
	if !ok {
		raw, _ := json.Marshal(trace.Final.Output)
		if err := json.Unmarshal(raw, &output); err != nil {
			t.Fatalf("decode final output: %v", err)
		}
	}
	for _, field := range []string{"analysis_summary", "failing_tests", "likely_root_cause", "changed_files"} {
		if _, ok := output[field]; !ok {
			t.Fatalf("final output missing %q: %#v", field, output)
		}
	}
}

func TestRuntimeRunsSmallRefactorSkillWithConstrainedWrite(t *testing.T) {
	skillDir := filepath.Join(t.TempDir(), "repo.apply_small_refactor")
	copyDir(t, filepath.Join("..", "..", "examples", "repo.apply_small_refactor"), skillDir)
	manifestPath := filepath.Join(skillDir, "manifest.yaml")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")

	result, err := Run(RunOptions{ManifestPath: manifestPath, RunDir: runDir})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("status = %q", result.Status)
	}
	if len(result.ChangedFiles) != 1 || filepath.Base(result.ChangedFiles[0]) != "greeter.py" {
		t.Fatalf("expected only greeter.py to change, got %#v", result.ChangedFiles)
	}

	trace, err := ReadTrace(result.TracePath)
	if err != nil {
		t.Fatal(err)
	}
	writePaths := toolRequestPaths(*trace, "fs.write")
	if len(writePaths) != 1 || filepath.Base(writePaths[0]) != "greeter.py" {
		t.Fatalf("expected one fs.write to greeter.py, got %#v", writePaths)
	}
	output, ok := trace.Final.Output.(map[string]any)
	if !ok {
		raw, _ := json.Marshal(trace.Final.Output)
		if err := json.Unmarshal(raw, &output); err != nil {
			t.Fatalf("decode final output: %v", err)
		}
	}
	for _, field := range []string{"patch_summary", "changed_files", "refactor_summary"} {
		if _, ok := output[field]; !ok {
			t.Fatalf("final output missing %q: %#v", field, output)
		}
	}
}

func TestRuntimeRecordsPolicyViolationTrace(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, true)
	manifestPath := writeManifest(t, root, repo)
	runDir := filepath.Join(root, ".agenix-runs")

	result, err := Run(RunOptions{ManifestPath: manifestPath, RunDir: runDir, Adapter: EscapeAdapter{Path: filepath.Join(root, "outside.txt")}})
	if err == nil {
		t.Fatal("expected policy violation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	if result.TracePath == "" {
		t.Fatal("expected trace path for failed run")
	}
	raw, err := os.ReadFile(result.TracePath)
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(raw) {
		t.Fatal("policy failure trace is not valid JSON")
	}
}

func TestRuntimeRejectsAdapterMissingCapabilitiesBeforeExecution(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, true)
	manifestPath := writeManifest(t, root, repo)
	runDir := filepath.Join(root, ".agenix-runs")
	called := false

	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter: capabilityLimitedAdapter{
			called: &called,
			metadata: AdapterMetadata{
				Name:            "limited",
				ModelProfile:    "limited",
				SupportedSkills: []string{"repo.fix_test_failure"},
				Capabilities: CapabilitySet{
					StructuredOutput: true,
					MaxContextTokens: 32000,
					ReasoningLevel:   "medium",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected adapter capability failure")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
	if called {
		t.Fatal("adapter Execute should not have been called")
	}
	trace, readErr := ReadTrace(result.TracePath)
	if readErr != nil {
		t.Fatalf("ReadTrace returned error: %v", readErr)
	}
	if !traceHasAdapterEvent(*trace, "capability_check", "failed") {
		t.Fatalf("trace does not contain failed adapter capability_check event: %#v", trace.Events)
	}
	if traceHasEvent(*trace, "tool_call", "fs.read") {
		t.Fatalf("capability failure should happen before tool calls: %#v", trace.Events)
	}
}

func TestVerifyExistingTraceRerunsVerifiers(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, false)
	manifestPath := writeManifest(t, root, repo)
	runDir := filepath.Join(root, ".agenix-runs")

	result, err := Run(RunOptions{ManifestPath: manifestPath, RunDir: runDir})
	if err != nil {
		t.Fatal(err)
	}

	verifyResult, err := Verify(result.TracePath)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if verifyResult.Status != "passed" {
		t.Fatalf("verify status = %q", verifyResult.Status)
	}
}

func TestVerifyRejectsFailedTrace(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, false)
	manifestPath := writeManifest(t, root, repo)
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{})
	trace.ManifestPath = manifestPath
	trace.SetFinal("failed", map[string]any{"patch_summary": "x", "changed_files": []string{}}, "previous failure")
	tracePath := filepath.Join(root, "trace.json")
	if err := WriteTrace(tracePath, trace); err != nil {
		t.Fatal(err)
	}

	_, err := Verify(tracePath)
	if err == nil {
		t.Fatal("expected failed trace to be rejected")
	}
	if !IsErrorClass(err, ErrVerificationFailed) {
		t.Fatalf("expected VerificationFailed, got %v", err)
	}
}

func TestVerifyRejectsPolicyViolationEvents(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, false)
	manifestPath := writeManifest(t, root, repo)
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{})
	trace.ManifestPath = manifestPath
	trace.AddToolEvent("fs.write", map[string]string{"path": filepath.Join(root, "outside.py")}, nil, NewError(ErrPolicyViolation, "escape"), 1)
	trace.SetFinal("passed", map[string]any{"patch_summary": "x", "changed_files": []string{}}, "")
	tracePath := filepath.Join(root, "trace.json")
	if err := WriteTrace(tracePath, trace); err != nil {
		t.Fatal(err)
	}

	_, err := Verify(tracePath)
	if err == nil {
		t.Fatal("expected policy violation trace to be rejected")
	}
	if !IsErrorClass(err, ErrVerificationFailed) {
		t.Fatalf("expected VerificationFailed, got %v", err)
	}
}

func TestVerifyRejectsChangedFilesOutsideWriteScope(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, false)
	manifestPath := writeManifest(t, root, repo)
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{})
	trace.ManifestPath = manifestPath
	trace.SetFinal("passed", map[string]any{"patch_summary": "x", "changed_files": []string{filepath.Join(root, "outside.py")}}, "")
	tracePath := filepath.Join(root, "trace.json")
	if err := WriteTrace(tracePath, trace); err != nil {
		t.Fatal(err)
	}

	_, err := Verify(tracePath)
	if err == nil {
		t.Fatal("expected outside changed file to be rejected")
	}
	if !IsErrorClass(err, ErrVerificationFailed) {
		t.Fatalf("expected VerificationFailed, got %v", err)
	}
}

func TestReplaySummarizesTrace(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, false)
	manifestPath := writeManifest(t, root, repo)
	result, err := Run(RunOptions{ManifestPath: manifestPath, RunDir: filepath.Join(root, ".agenix-runs")})
	if err != nil {
		t.Fatal(err)
	}

	summary, err := Replay(result.TracePath)
	if err != nil {
		t.Fatalf("Replay returned error: %v", err)
	}
	if summary.Skill != "repo.fix_test_failure" || summary.FinalStatus != "passed" || summary.EventCount == 0 {
		t.Fatalf("bad replay summary: %#v", summary)
	}
}

func writePythonFixture(t *testing.T, root string, broken bool) string {
	t.Helper()
	repo := filepath.Join(root, "repo")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	body := "def add(a, b):\n    return a + b\n"
	if broken {
		body = "def add(a, b):\n    return a - b\n"
	}
	if err := os.WriteFile(filepath.Join(repo, "mathlib.py"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "test_mathlib.py"), []byte("from mathlib import add\n\n\ndef test_adds_numbers():\n    assert add(2, 3) == 5\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return repo
}

func writeManifest(t *testing.T, root, repo string) string {
	t.Helper()
	path := filepath.Join(root, "manifest.yaml")
	writeManifestAt(t, path, repo)
	return path
}

func writeManifestAt(t *testing.T, path, repo string) {
	t.Helper()
	content := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 32000
    reasoning_level: medium
tools:
  - fs
  - shell
  - git
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
  shell:
    allow:
      - run: ["python3", "-m", "pytest", "-q"]
      - run: ["git", "status", "--short"]
      - run: ["git", "diff", "--", "."]
inputs:
  repo_path: ` + repo + `
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
recovery:
  strategy: checkpoint
  intervals: 5
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func traceHasEvent(trace Trace, eventType, name string) bool {
	for _, event := range trace.Events {
		if event.Type == eventType && event.Name == name {
			return true
		}
	}
	return false
}

func traceHasVerifier(trace Trace, name, status string) bool {
	for _, event := range trace.Events {
		if event.Type == "verifier" && event.Name == name && event.Status == status {
			return true
		}
	}
	return false
}

func traceHasAdapterEvent(trace Trace, name, status string) bool {
	for _, event := range trace.Events {
		if event.Type == "adapter" && event.Name == name && event.Status == status {
			return true
		}
	}
	return false
}

func traceHasVerifierRequestField(trace Trace, name, field string) bool {
	for _, event := range trace.Events {
		if event.Type != "verifier" || event.Name != name {
			continue
		}
		raw, _ := json.Marshal(event.Request)
		var request map[string]any
		if err := json.Unmarshal(raw, &request); err != nil {
			continue
		}
		if _, ok := request[field]; ok {
			return true
		}
	}
	return false
}

type capabilityLimitedAdapter struct {
	called   *bool
	metadata AdapterMetadata
}

func (a capabilityLimitedAdapter) Metadata() AdapterMetadata {
	return a.metadata
}

func (a capabilityLimitedAdapter) Execute(_ Manifest, _ *Tools) (map[string]any, error) {
	if a.called != nil {
		*a.called = true
	}
	return map[string]any{}, nil
}

func toolRequestPaths(trace Trace, name string) []string {
	paths := []string{}
	for _, event := range trace.Events {
		if event.Type != "tool_call" || event.Name != name {
			continue
		}
		raw, _ := json.Marshal(event.Request)
		var request map[string]any
		if err := json.Unmarshal(raw, &request); err != nil {
			continue
		}
		if path, ok := request["path"].(string); ok {
			paths = append(paths, path)
		}
	}
	return paths
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	if err := filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, raw, 0o600)
	}); err != nil {
		t.Fatal(err)
	}
}
