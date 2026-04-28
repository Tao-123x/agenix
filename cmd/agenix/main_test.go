package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
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

func TestUsageMentionsAcceptanceCommand(t *testing.T) {
	err := usage()
	if err == nil {
		t.Fatal("expected usage error")
	}
	if !strings.Contains(err.Error(), "acceptance [--v0.2|--v0.3]") {
		t.Fatalf("usage missing acceptance command: %v", err)
	}
}

func TestCLIAcceptanceRunsV0Sweep(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "acceptance").CombinedOutput()
	if err != nil {
		t.Fatalf("acceptance failed: %v\n%s", err, out)
	}
	text := strings.TrimSpace(string(out))
	if text != "status=passed skills=3 runs=6" {
		t.Fatalf("unexpected acceptance output: %s", text)
	}
}

func TestCLIAcceptanceRunsV02AuthoringSweep(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "acceptance", "--v0.2").CombinedOutput()
	if err != nil {
		t.Fatalf("v0.2 acceptance failed: %v\n%s", err, out)
	}
	text := strings.TrimSpace(string(out))
	if text != "status=passed release=v0.2 templates=2 skills=2 checks=3 failure_reports=1" {
		t.Fatalf("unexpected v0.2 acceptance output: %s", text)
	}
}

func TestCLIAcceptanceRunsV03AdapterReadinessSweep(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "acceptance", "--v0.3").CombinedOutput()
	if err != nil {
		t.Fatalf("v0.3 acceptance failed: %v\n%s", err, out)
	}
	text := strings.TrimSpace(string(out))
	if text != "status=passed release=v0.3 adapters=5 compatibility_reports=3 schemas=3 provider_smoke=skipped_offline" {
		t.Fatalf("unexpected v0.3 acceptance output: %s", text)
	}
}

func TestCLIInitTemplatesListsBuiltins(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "init", "templates").CombinedOutput()
	if err != nil {
		t.Fatalf("init templates failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{
		"template=python-pytest adapter=python-pytest-template writes=false",
		"template=repo-fix-test-failure adapter=repo-fix-test-failure-template writes=true",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("init templates output missing %q: %s", want, text)
		}
	}
}

func TestCLIInitTemplatesPrintsJSON(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "init", "templates", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("init templates --json failed: %v\n%s", err, out)
	}
	var templates []struct {
		Name    string `json:"name"`
		Adapter string `json:"adapter"`
		Writes  bool   `json:"writes"`
	}
	if err := json.Unmarshal(out, &templates); err != nil {
		t.Fatalf("init templates output is not JSON: %v\n%s", err, out)
	}
	if len(templates) != 2 {
		t.Fatalf("template count = %d, want 2: %#v", len(templates), templates)
	}
	if templates[0].Name != "python-pytest" || templates[0].Adapter != "python-pytest-template" || templates[0].Writes {
		t.Fatalf("unexpected first template: %#v", templates[0])
	}
	if templates[1].Name != "repo-fix-test-failure" || templates[1].Adapter != "repo-fix-test-failure-template" || !templates[1].Writes {
		t.Fatalf("unexpected second template: %#v", templates[1])
	}
}

func TestCLIAdaptersListsBuiltins(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "adapters").CombinedOutput()
	if err != nil {
		t.Fatalf("adapters failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{
		"adapter=fake-scripted",
		"adapter=openai-analyze",
		"transport=remote",
		"supported_skills=repo.analyze_test_failures.remote",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("adapters output missing %q: %s", want, text)
		}
	}
}

func TestCLIAdaptersPrintsJSON(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "adapters", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("adapters --json failed: %v\n%s", err, out)
	}
	var adapters []struct {
		Name      string `json:"name"`
		Provider  string `json:"provider"`
		Transport string `json:"transport"`
	}
	if err := json.Unmarshal(out, &adapters); err != nil {
		t.Fatalf("adapters output is not JSON: %v\n%s", err, out)
	}
	if len(adapters) != 5 {
		t.Fatalf("adapter count = %d, want 5: %#v", len(adapters), adapters)
	}
	if adapters[0].Name != "fake-scripted" || adapters[0].Transport != "local" {
		t.Fatalf("unexpected first adapter: %#v", adapters[0])
	}
	if adapters[2].Name != "openai-analyze" || adapters[2].Provider != "openai" || adapters[2].Transport != "remote" {
		t.Fatalf("unexpected OpenAI adapter descriptor: %#v", adapters[2])
	}
}

func TestCLIAdaptersCompatiblePreflightsManifest(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "adapters", "compatible", filepath.Join("..", "..", "examples", "repo.fix_test_failure", "manifest.yaml")).CombinedOutput()
	if err != nil {
		t.Fatalf("adapters compatible failed: %v\n%s", err, out)
	}
	text := string(out)
	for _, want := range []string{
		"skill=repo.fix_test_failure",
		"adapter=fake-scripted compatible=true",
		"adapter=openai-analyze compatible=false error_class=UnsupportedAdapter",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("compatible output missing %q: %s", want, text)
		}
	}
}

func TestCLIAdaptersCompatiblePrintsJSON(t *testing.T) {
	out, err := exec.Command("go", "run", ".", "adapters", "compatible", filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote", "manifest.yaml"), "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("adapters compatible --json failed: %v\n%s", err, out)
	}
	var report struct {
		Kind     string `json:"kind"`
		Skill    string `json:"skill"`
		Adapters []struct {
			Name       string `json:"name"`
			Transport  string `json:"transport"`
			Compatible bool   `json:"compatible"`
			ErrorClass string `json:"error_class,omitempty"`
		} `json:"adapters"`
	}
	if err := json.Unmarshal(out, &report); err != nil {
		t.Fatalf("compatible output is not JSON: %v\n%s", err, out)
	}
	if report.Kind != "adapter_compatibility_report" || report.Skill != "repo.analyze_test_failures.remote" {
		t.Fatalf("unexpected compatibility report identity: %#v", report)
	}
	var sawOpenAI bool
	for _, adapter := range report.Adapters {
		if adapter.Name == "openai-analyze" {
			sawOpenAI = true
			if !adapter.Compatible || adapter.Transport != "remote" || adapter.ErrorClass != "" {
				t.Fatalf("unexpected openai compatibility: %#v", adapter)
			}
		}
	}
	if !sawOpenAI {
		t.Fatalf("compatibility report missing openai adapter: %#v", report.Adapters)
	}
}

func TestCLIAdaptersCompatibilityReportCanBeValidated(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "adapter-compatibility-report.json")

	out, err := exec.Command("go", "run", ".", "adapters", "compatible", filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote", "manifest.yaml"), "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("adapters compatible --json failed: %v\n%s", err, out)
	}
	if err := os.WriteFile(reportPath, out, 0o600); err != nil {
		t.Fatal(err)
	}

	validateOut, err := exec.Command("go", "run", ".", "validate", reportPath).CombinedOutput()
	if err != nil {
		t.Fatalf("validate adapter compatibility report failed: %v\n%s", err, validateOut)
	}
	text := string(validateOut)
	if !strings.Contains(text, "status=valid kind=adapter_compatibility_report") ||
		!strings.Contains(text, "adapter-compatibility-report.schema.json") {
		t.Fatalf("unexpected validate output: %s", text)
	}
}

func TestCLIAdaptersCompatibleAcceptsRegistryReference(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_fix")
	artifact := filepath.Join(root, "repo.demo_fix.agenix")
	registry := filepath.Join(root, "registry")
	if out, err := exec.Command("go", "run", ".", "init", "skill", "repo.demo_fix", "--template", "repo-fix-test-failure", "-o", skillDir).CombinedOutput(); err != nil {
		t.Fatalf("init skill failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput(); err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	if out, err := exec.Command("go", "run", ".", "publish", artifact, "--registry", registry).CombinedOutput(); err != nil {
		t.Fatalf("publish failed: %v\n%s", err, out)
	}
	if err := os.RemoveAll(skillDir); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command("go", "run", ".", "adapters", "compatible", "repo.demo_fix@0.1.0", "--registry", registry).CombinedOutput()
	if err != nil {
		t.Fatalf("adapters compatible registry ref failed: %v\n%s", err, out)
	}
	text := string(out)
	if !strings.Contains(text, "skill=repo.demo_fix") || !strings.Contains(text, "adapter=repo-fix-test-failure-template compatible=true") {
		t.Fatalf("unexpected registry compatibility output: %s", text)
	}
}

func TestCLIInitSkillCreatesRunnablePythonPytestSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_skill")
	artifact := filepath.Join(root, "repo.demo_skill.agenix")

	initOut, err := exec.Command("go", "run", ".", "init", "skill", "repo.demo_skill", "--template", "python-pytest", "-o", skillDir).CombinedOutput()
	if err != nil {
		t.Fatalf("init skill failed: %v\n%s", err, initOut)
	}
	initText := string(initOut)
	for _, want := range []string{"status=created", "skill=repo.demo_skill", "template=python-pytest", "path=" + skillDir} {
		if !strings.Contains(initText, want) {
			t.Fatalf("init output missing %q: %s", want, initText)
		}
	}

	for _, rel := range []string{"manifest.yaml", "README.md", "fixture/skill.py", "fixture/test_skill.py"} {
		if _, err := os.Stat(filepath.Join(skillDir, rel)); err != nil {
			t.Fatalf("generated file %s missing: %v", rel, err)
		}
	}

	validateOut, err := exec.Command("go", "run", ".", "validate", filepath.Join(skillDir, "manifest.yaml")).CombinedOutput()
	if err != nil {
		t.Fatalf("validate generated manifest failed: %v\n%s", err, validateOut)
	}

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build generated skill failed: %v\n%s", err, buildOut)
	}

	runOut, err := exec.Command("go", "run", ".", "run", artifact, "--adapter", "python-pytest-template").CombinedOutput()
	if err != nil {
		t.Fatalf("run generated artifact failed: %v\n%s", err, runOut)
	}
	runText := string(runOut)
	if !strings.Contains(runText, "status=passed") ||
		!strings.Contains(runText, "verifiers=run_tests:passed,output_schema_check:passed") {
		t.Fatalf("unexpected run output: %s", runText)
	}
}

func TestCLICheckRunsGeneratedPythonPytestSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_skill")

	initOut, err := exec.Command("go", "run", ".", "init", "skill", "repo.demo_skill", "--template", "python-pytest", "-o", skillDir).CombinedOutput()
	if err != nil {
		t.Fatalf("init skill failed: %v\n%s", err, initOut)
	}

	checkOut, err := exec.Command("go", "run", ".", "check", skillDir, "--adapter", "python-pytest-template").CombinedOutput()
	if err != nil {
		t.Fatalf("check generated skill failed: %v\n%s", err, checkOut)
	}
	text := string(checkOut)
	for _, want := range []string{
		"status=passed",
		"skill=repo.demo_skill",
		"artifact=",
		"trace=",
		"verifiers=run_tests:passed,output_schema_check:passed",
		"events=",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("check output missing %q: %s", want, text)
		}
	}
}

func TestCLICheckPrintsJSONReport(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_skill")

	initOut, err := exec.Command("go", "run", ".", "init", "skill", "repo.demo_skill", "--template", "python-pytest", "-o", skillDir).CombinedOutput()
	if err != nil {
		t.Fatalf("init skill failed: %v\n%s", err, initOut)
	}

	checkOut, err := exec.Command("go", "run", ".", "check", skillDir, "--adapter", "python-pytest-template", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("check generated skill failed: %v\n%s", err, checkOut)
	}

	var report struct {
		Kind            string   `json:"kind"`
		Status          string   `json:"status"`
		Skill           string   `json:"skill"`
		Version         string   `json:"version"`
		ArtifactPath    string   `json:"artifact_path"`
		RunID           string   `json:"run_id"`
		TracePath       string   `json:"trace_path"`
		VerifierSummary []string `json:"verifier_summary"`
		EventCount      int      `json:"event_count"`
	}
	if err := json.Unmarshal(checkOut, &report); err != nil {
		t.Fatalf("check output is not JSON: %v\n%s", err, checkOut)
	}
	if report.Kind != "check_report" {
		t.Fatalf("report kind = %q", report.Kind)
	}
	if report.Status != "passed" || report.Skill != "repo.demo_skill" || report.Version != "0.1.0" {
		t.Fatalf("unexpected JSON report identity: %#v", report)
	}
	if report.ArtifactPath == "" || report.RunID == "" || report.TracePath == "" {
		t.Fatalf("JSON report missing paths or run id: %#v", report)
	}
	if report.EventCount == 0 {
		t.Fatalf("JSON report missing event count: %#v", report)
	}
	if got := strings.Join(report.VerifierSummary, ","); got != "run_tests:passed,output_schema_check:passed" {
		t.Fatalf("verifier summary = %q", got)
	}
}

func TestCLICheckJSONReportCanBeValidated(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_skill")
	reportPath := filepath.Join(root, "check-report.json")

	initOut, err := exec.Command("go", "run", ".", "init", "skill", "repo.demo_skill", "--template", "python-pytest", "-o", skillDir).CombinedOutput()
	if err != nil {
		t.Fatalf("init skill failed: %v\n%s", err, initOut)
	}

	checkOut, err := exec.Command("go", "run", ".", "check", skillDir, "--adapter", "python-pytest-template", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("check generated skill failed: %v\n%s", err, checkOut)
	}
	if err := os.WriteFile(reportPath, checkOut, 0o600); err != nil {
		t.Fatal(err)
	}

	validateOut, err := exec.Command("go", "run", ".", "validate", reportPath).CombinedOutput()
	if err != nil {
		t.Fatalf("validate check report failed: %v\n%s", err, validateOut)
	}
	text := string(validateOut)
	if !strings.Contains(text, "status=valid kind=check_report") ||
		!strings.Contains(text, "check-report.schema.json") {
		t.Fatalf("unexpected validate output: %s", text)
	}
}

func TestCLICheckJSONFailurePrintsValidReport(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_skill")
	reportPath := filepath.Join(root, "failed-check-report.json")

	initOut, err := exec.Command("go", "run", ".", "init", "skill", "repo.demo_skill", "--template", "python-pytest", "-o", skillDir).CombinedOutput()
	if err != nil {
		t.Fatalf("init skill failed: %v\n%s", err, initOut)
	}
	brokenSource := `def normalize(value):
    return value
`
	if err := os.WriteFile(filepath.Join(skillDir, "fixture", "skill.py"), []byte(brokenSource), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "run", ".", "check", skillDir, "--adapter", "python-pytest-template", "--json")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	checkOut, err := cmd.Output()
	if err == nil {
		t.Fatalf("expected check to fail, stdout=%s stderr=%s", checkOut, stderr.String())
	}

	var report struct {
		Kind         string   `json:"kind"`
		Status       string   `json:"status"`
		Skill        string   `json:"skill"`
		RunID        string   `json:"run_id"`
		TracePath    string   `json:"trace_path"`
		ChangedFiles []string `json:"changed_files"`
		EventCount   int      `json:"event_count"`
		ErrorClass   string   `json:"error_class"`
		ErrorMessage string   `json:"error_message"`
	}
	if err := json.Unmarshal(checkOut, &report); err != nil {
		t.Fatalf("failed check stdout is not JSON: %v\nstdout=%s\nstderr=%s", err, checkOut, stderr.String())
	}
	if report.Kind != "check_report" || report.Status != "failed" || report.Skill != "repo.demo_skill" {
		t.Fatalf("unexpected failed report identity: %#v", report)
	}
	if report.RunID == "" || report.TracePath == "" {
		t.Fatalf("failed report missing run evidence: %#v", report)
	}
	if report.ChangedFiles == nil {
		t.Fatalf("failed report changed_files should be an empty JSON array, got nil")
	}
	if report.EventCount == 0 {
		t.Fatalf("failed report missing event count: %#v", report)
	}
	if report.ErrorClass != "VerificationFailed" || report.ErrorMessage == "" {
		t.Fatalf("failed report missing stable error fields: %#v", report)
	}
	if !strings.Contains(stderr.String(), "error=VerificationFailed") {
		t.Fatalf("stderr missing stable error class: %s", stderr.String())
	}
	if err := os.WriteFile(reportPath, checkOut, 0o600); err != nil {
		t.Fatal(err)
	}
	validateOut, err := exec.Command("go", "run", ".", "validate", reportPath).CombinedOutput()
	if err != nil {
		t.Fatalf("validate failed check report failed: %v\n%s", err, validateOut)
	}
	if !strings.Contains(string(validateOut), "status=valid kind=check_report") {
		t.Fatalf("unexpected validate output: %s", validateOut)
	}
}

func TestCLIInitRepoFixTestFailureTemplateCreatesWritableSkill(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_fix")
	reportPath := filepath.Join(root, "check-report.json")

	initOut, err := exec.Command("go", "run", ".", "init", "skill", "repo.demo_fix", "--template", "repo-fix-test-failure", "-o", skillDir).CombinedOutput()
	if err != nil {
		t.Fatalf("init skill failed: %v\n%s", err, initOut)
	}
	initText := string(initOut)
	for _, want := range []string{"status=created", "skill=repo.demo_fix", "template=repo-fix-test-failure", "path=" + skillDir} {
		if !strings.Contains(initText, want) {
			t.Fatalf("init output missing %q: %s", want, initText)
		}
	}

	pytestOut, err := exec.Command("python3", "-m", "pytest", "-q", filepath.Join(skillDir, "fixture")).CombinedOutput()
	if err == nil {
		t.Fatalf("generated fixture should start failing: %s", pytestOut)
	}

	checkOut, err := exec.Command("go", "run", ".", "check", skillDir, "--adapter", "repo-fix-test-failure-template", "--json").CombinedOutput()
	if err != nil {
		t.Fatalf("check generated skill failed: %v\n%s", err, checkOut)
	}
	if err := os.WriteFile(reportPath, checkOut, 0o600); err != nil {
		t.Fatal(err)
	}

	var report struct {
		Kind            string   `json:"kind"`
		Status          string   `json:"status"`
		Skill           string   `json:"skill"`
		ChangedFiles    []string `json:"changed_files"`
		VerifierSummary []string `json:"verifier_summary"`
	}
	if err := json.Unmarshal(checkOut, &report); err != nil {
		t.Fatalf("check output is not JSON: %v\n%s", err, checkOut)
	}
	if report.Kind != "check_report" || report.Status != "passed" || report.Skill != "repo.demo_fix" {
		t.Fatalf("unexpected report: %#v", report)
	}
	if len(report.ChangedFiles) != 1 || !strings.HasSuffix(filepath.ToSlash(report.ChangedFiles[0]), "/fixture/mathlib.py") {
		t.Fatalf("unexpected changed files: %#v", report.ChangedFiles)
	}
	if got := strings.Join(report.VerifierSummary, ","); got != "run_tests:passed,output_schema_check:passed" {
		t.Fatalf("verifier summary = %q", got)
	}

	validateOut, err := exec.Command("go", "run", ".", "validate", reportPath).CombinedOutput()
	if err != nil {
		t.Fatalf("validate check report failed: %v\n%s", err, validateOut)
	}
	if !strings.Contains(string(validateOut), "status=valid kind=check_report") {
		t.Fatalf("unexpected validate output: %s", validateOut)
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

func TestCLIRunRemoteAnalyzeArtifactWithStubProvider(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "output": [
    {
      "type": "message",
      "content": [
        {
          "type": "output_text",
          "text": "{\"analysis_summary\":\"fixture fails\",\"failing_tests\":[\"test_mathlib.py::test_adds_numbers\"],\"likely_root_cause\":\"mathlib.add subtracts instead of adding\",\"changed_files\":[]}"
        }
      ]
    }
  ]
}`))
	}))
	defer server.Close()

	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote")
	artifact := filepath.Join(root, "analyze.remote.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	cmd := exec.Command("go", "run", ".", "run", artifact, "--adapter", "openai-analyze")
	cmd.Env = append(os.Environ(), "OPENAI_API_KEY=test-key", "AGENIX_OPENAI_BASE_URL="+server.URL)
	runOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run artifact failed: %v\n%s", err, runOut)
	}
	if atomic.LoadInt32(&callCount) == 0 {
		t.Fatal("stub provider server was not called")
	}
	text := string(runOut)
	if !strings.Contains(text, "status=passed") {
		t.Fatalf("unexpected run output: %s", text)
	}
}

func TestCLIRunRemoteAnalyzeArtifactWithStubProviderRateLimitFailure(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "120")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{
  "error": {
    "message": "rate limit exceeded",
    "type": "rate_limit_error",
    "code": "rate_limit_exceeded"
  }
}`))
	}))
	defer server.Close()

	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote")
	artifact := filepath.Join(root, "analyze.remote.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	cmd := exec.Command("go", "run", ".", "run", artifact, "--adapter", "openai-analyze")
	cmd.Env = append(os.Environ(), "OPENAI_API_KEY=test-key", "AGENIX_OPENAI_BASE_URL="+server.URL)
	runOut, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected run failure, got success: %s", runOut)
	}
	if atomic.LoadInt32(&callCount) == 0 {
		t.Fatal("stub provider server was not called")
	}

	text := string(runOut)
	if !strings.Contains(text, "error=DriverError") {
		t.Fatalf("missing driver error class: %s", text)
	}
	if !strings.Contains(text, "message=DriverError: OpenAI responses API returned 429 Too Many Requests: rate limit exceeded (retry after 120s)") {
		t.Fatalf("missing mapped provider details: %s", text)
	}
}

func TestCLIRunRemoteAnalyzeArtifactWithStubProviderTimeout(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		atomic.AddInt32(&callCount, 1)
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[]}`))
	}))
	defer server.Close()

	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote")
	artifact := filepath.Join(root, "analyze.remote.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	cmd := exec.Command("go", "run", ".", "run", artifact, "--adapter", "openai-analyze")
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=test-key",
		"AGENIX_OPENAI_BASE_URL="+server.URL,
		"AGENIX_OPENAI_TIMEOUT_MS=5",
	)
	runOut, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected run failure, got success: %s", runOut)
	}
	if atomic.LoadInt32(&callCount) == 0 {
		t.Fatal("stub provider server was not called")
	}

	text := string(runOut)
	if !strings.Contains(text, "error=Timeout") {
		t.Fatalf("missing timeout error class: %s", text)
	}
	if !strings.Contains(text, "message=Timeout: OpenAI responses API timed out") {
		t.Fatalf("missing timeout details: %s", text)
	}
}

func TestCLIRunRemoteAnalyzeArtifactWithOversizedProviderResponse(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"content":[{"type":"output_text","text":"` + strings.Repeat("x", 128) + `"}]}]}`))
	}))
	defer server.Close()

	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote")
	artifact := filepath.Join(root, "analyze.remote.agenix")

	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	cmd := exec.Command("go", "run", ".", "run", artifact, "--adapter", "openai-analyze")
	cmd.Env = append(os.Environ(),
		"OPENAI_API_KEY=test-key",
		"AGENIX_OPENAI_BASE_URL="+server.URL,
		"AGENIX_OPENAI_MAX_RESPONSE_BYTES=64",
	)
	runOut, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected run failure, got success: %s", runOut)
	}
	if atomic.LoadInt32(&callCount) == 0 {
		t.Fatal("stub provider server was not called")
	}

	text := string(runOut)
	if !strings.Contains(text, "error=DriverError") {
		t.Fatalf("missing driver error class: %s", text)
	}
	if !strings.Contains(text, "message=DriverError: OpenAI response body exceeded 64 bytes") {
		t.Fatalf("missing response size details: %s", text)
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
