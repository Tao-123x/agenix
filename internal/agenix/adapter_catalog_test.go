package agenix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestListBuiltinAdaptersReturnsStableCatalog(t *testing.T) {
	adapters := ListBuiltinAdapters()
	if len(adapters) != 5 {
		t.Fatalf("adapter count = %d, want 5: %#v", len(adapters), adapters)
	}
	wantNames := []string{"fake-scripted", "heuristic-analyze", "openai-analyze", "python-pytest-template", "repo-fix-test-failure-template"}
	for i, want := range wantNames {
		if adapters[i].Name != want {
			t.Fatalf("adapter[%d].Name = %q, want %q", i, adapters[i].Name, want)
		}
		if adapters[i].Transport == "" {
			t.Fatalf("adapter[%d] missing normalized transport: %#v", i, adapters[i])
		}
		if !adapters[i].Capabilities.ToolCalling || !adapters[i].Capabilities.StructuredOutput {
			t.Fatalf("adapter[%d] missing core capabilities: %#v", i, adapters[i])
		}
	}
	if adapters[2].Provider != "openai" || adapters[2].Transport != "remote" {
		t.Fatalf("unexpected openai adapter metadata: %#v", adapters[2])
	}
}

func TestCheckBuiltinAdapterCompatibilityReportsFailuresBeforeExecution(t *testing.T) {
	report, err := CheckBuiltinAdapterCompatibility(AdapterCompatibilityOptions{
		Target: filepath.Join("..", "..", "examples", "repo.fix_test_failure", "manifest.yaml"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != "adapter_compatibility_report" || report.Skill != "repo.fix_test_failure" {
		t.Fatalf("unexpected report identity: %#v", report)
	}
	if len(report.Adapters) != 5 {
		t.Fatalf("adapter report count = %d, want 5: %#v", len(report.Adapters), report.Adapters)
	}
	fake := findCompatibility(t, report, "fake-scripted")
	if !fake.Compatible || fake.ErrorClass != "" {
		t.Fatalf("fake-scripted should be compatible: %#v", fake)
	}
	openai := findCompatibility(t, report, "openai-analyze")
	if openai.Compatible || openai.ErrorClass != ErrUnsupportedAdapter {
		t.Fatalf("openai-analyze should fail skill support preflight: %#v", openai)
	}
}

func TestCheckBuiltinAdapterCompatibilityAcceptsRemoteAdapterWhenManifestAllowsNetwork(t *testing.T) {
	report, err := CheckBuiltinAdapterCompatibility(AdapterCompatibilityOptions{
		Target: filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote", "manifest.yaml"),
	})
	if err != nil {
		t.Fatal(err)
	}
	openai := findCompatibility(t, report, "openai-analyze")
	if !openai.Compatible || openai.ErrorClass != "" || openai.Transport != "remote" {
		t.Fatalf("openai-analyze should pass remote preflight: %#v", openai)
	}
	fake := findCompatibility(t, report, "fake-scripted")
	if fake.Compatible || fake.ErrorClass != ErrUnsupportedAdapter {
		t.Fatalf("fake-scripted should reject unsupported remote skill: %#v", fake)
	}
}

func TestCheckBuiltinAdapterCompatibilityRejectsRemoteAdapterWhenNetworkDenied(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "remote-denied")
	if err := os.MkdirAll(filepath.Join(skillDir, "fixture"), 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `apiVersion: agenix/v0.1
kind: Skill
name: repo.analyze_test_failures.remote
version: 0.1.0
description: Remote adapter policy rejection fixture.
capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 32000
    reasoning_level: medium
tools:
  - fs
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write: []
inputs:
  repo_path: fixture
outputs:
  required:
    - analysis_summary
verifiers:
  - type: schema
    name: output_schema_check
    schemaRef: outputs
`
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	report, err := CheckBuiltinAdapterCompatibility(AdapterCompatibilityOptions{Target: skillDir})
	if err != nil {
		t.Fatal(err)
	}
	openai := findCompatibility(t, report, "openai-analyze")
	if openai.Compatible || openai.ErrorClass != ErrPolicyViolation {
		t.Fatalf("openai-analyze should fail network policy preflight: %#v", openai)
	}
}

func TestCheckBuiltinAdapterCompatibilityMaterializesArtifact(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	artifactPath := filepath.Join(root, "skill.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath}); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(skillDir); err != nil {
		t.Fatal(err)
	}

	report, err := CheckBuiltinAdapterCompatibility(AdapterCompatibilityOptions{Target: artifactPath})
	if err != nil {
		t.Fatal(err)
	}
	if report.Skill != "repo.fix_test_failure" {
		t.Fatalf("report skill = %q", report.Skill)
	}
	fake := findCompatibility(t, report, "fake-scripted")
	if !fake.Compatible {
		t.Fatalf("fake-scripted should be compatible with materialized artifact: %#v", fake)
	}
}

func TestCheckBuiltinAdapterCompatibilityRejectsTamperedArtifact(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	artifactPath := filepath.Join(root, "skill.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath}); err != nil {
		t.Fatal(err)
	}
	rewriteArtifactEntry(t, artifactPath, "files/fixture/mathlib.py", []byte("def add(a, b):\n    return a + b\n"))

	_, err := CheckBuiltinAdapterCompatibility(AdapterCompatibilityOptions{Target: artifactPath})
	if err == nil {
		t.Fatal("expected tampered artifact to fail compatibility preflight")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %v", err)
	}
}

func findCompatibility(t *testing.T, report AdapterCompatibilityReport, name string) AdapterCompatibility {
	t.Helper()
	for _, adapter := range report.Adapters {
		if adapter.Name == name {
			return adapter
		}
	}
	t.Fatalf("compatibility report missing adapter %q: %#v", name, report.Adapters)
	return AdapterCompatibility{}
}
