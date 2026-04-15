package agenix

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadManifestExpandsRuntimeSubstitutions(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	manifestPath := filepath.Join(dir, "manifest.yaml")
	manifest := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
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
inputs:
  repo_path: ` + repo + `
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
recovery:
  strategy: checkpoint
  intervals: 5
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if got.Name != "repo.fix_test_failure" {
		t.Fatalf("Name = %q", got.Name)
	}
	if got.Inputs["repo_path"] != repo {
		t.Fatalf("repo_path = %q", got.Inputs["repo_path"])
	}
	if got.Permissions.Filesystem.Read[0] != repo {
		t.Fatalf("read scope was not expanded: %#v", got.Permissions.Filesystem.Read)
	}
	if got.Verifiers[0].CWD != repo {
		t.Fatalf("verifier cwd was not expanded: %q", got.Verifiers[0].CWD)
	}
}

func TestLoadManifestResolvesRepoPathRelativeToManifest(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "skill")
	repo := filepath.Join(skillDir, "fixture")
	if err := os.MkdirAll(repo, 0o755); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(skillDir, "manifest.yaml")
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
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(oldwd); err != nil {
			t.Fatal(err)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}

	got, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if got.Inputs["repo_path"] != repo {
		t.Fatalf("repo_path = %q, want %q", got.Inputs["repo_path"], repo)
	}
	if got.Permissions.Filesystem.Write[0] != repo {
		t.Fatalf("write scope = %q, want %q", got.Permissions.Filesystem.Write[0], repo)
	}
}

func TestLoadManifestRejectsMissingRequiredFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
	if err := os.WriteFile(path, []byte("apiVersion: agenix/v0.1\nkind: Skill\n"), 0o600); err != nil {
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

func TestLoadManifestParsesStructuredCommandVerifierRun(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	manifestPath := filepath.Join(dir, "manifest.yaml")
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
  repo_path: ` + repo + `
outputs:
  required:
    - patch_summary
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
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if got.Verifiers[0].Command != "" {
		t.Fatalf("expected string command to stay empty, got %q", got.Verifiers[0].Command)
	}
	want := []string{"python3", "-m", "pytest", "-q"}
	if !reflect.DeepEqual(got.Verifiers[0].Run, want) {
		t.Fatalf("verifier run = %#v, want %#v", got.Verifiers[0].Run, want)
	}
}

func TestLoadManifestParsesStructuredCommandVerifierPolicy(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
	manifestPath := filepath.Join(dir, "manifest.yaml")
	manifest := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
permissions:
  network: false
inputs:
  repo_path: ` + repo + `
outputs:
  required:
    - patch_summary
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
`
	if err := os.WriteFile(manifestPath, []byte(manifest), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if got.Verifiers[0].Policy == nil {
		t.Fatal("expected verifier policy to be parsed")
	}
	if got.Verifiers[0].Policy.Executable != "python3" {
		t.Fatalf("verifier policy executable = %q", got.Verifiers[0].Policy.Executable)
	}
	if got.Verifiers[0].Policy.CWD != repo {
		t.Fatalf("verifier policy cwd = %q, want %q", got.Verifiers[0].Policy.CWD, repo)
	}
	if got.Verifiers[0].Policy.TimeoutMS != 120000 {
		t.Fatalf("verifier policy timeout_ms = %d", got.Verifiers[0].Policy.TimeoutMS)
	}
}

func TestLoadManifestParsesTopLevelRedactionBlock(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	raw := `apiVersion: agenix/v0.1
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
redaction:
  keys:
    - session_token
  patterns:
    - name: customer-bearer
      regex: '(?i)(x-customer-token:\\s*)([^\\s]+)'
      secret_group: 2
`
	if err := os.WriteFile(manifestPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if len(manifest.Redaction.Keys) != 1 || manifest.Redaction.Keys[0] != "session_token" {
		t.Fatalf("redaction keys = %#v", manifest.Redaction.Keys)
	}
	if len(manifest.Redaction.Patterns) != 1 {
		t.Fatalf("redaction patterns = %#v", manifest.Redaction.Patterns)
	}
	if manifest.Redaction.Patterns[0].Name != "customer-bearer" {
		t.Fatalf("pattern name = %#v", manifest.Redaction.Patterns[0])
	}
}
