package agenix

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildArtifactCreatesPortableCapsule(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	out := filepath.Join(root, "repo.fix_test_failure.agenix")

	result, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: out})
	if err != nil {
		t.Fatalf("BuildArtifact returned error: %v", err)
	}
	if result.Skill != "repo.fix_test_failure" || result.Version != "0.1.0" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Digest == "" || !strings.HasPrefix(result.Digest, "sha256:") {
		t.Fatalf("missing digest: %#v", result)
	}
	if result.FileCount != 4 {
		t.Fatalf("FileCount = %d", result.FileCount)
	}

	entries := readTarGzEntries(t, out)
	want := []string{
		"agenix.lock.json",
		"files/README.md",
		"files/fixture/mathlib.py",
		"files/fixture/test_mathlib.py",
		"manifest.yaml",
	}
	for _, name := range want {
		if _, ok := entries[name]; !ok {
			t.Fatalf("artifact missing %s; entries=%v", name, sortedKeys(entries))
		}
	}
	for _, ignored := range []string{"files/.DS_Store", "files/fixture/.pytest_cache/cache", "files/fixture/__pycache__/mathlib.pyc"} {
		if _, ok := entries[ignored]; ok {
			t.Fatalf("artifact included ignored file %s", ignored)
		}
	}

	var lock ArtifactLock
	if err := json.Unmarshal(entries["agenix.lock.json"], &lock); err != nil {
		t.Fatalf("lockfile is not JSON: %v", err)
	}
	if lock.ArtifactVersion != "agenix.artifact/v0.1" || lock.Skill.Name != "repo.fix_test_failure" {
		t.Fatalf("bad lockfile: %#v", lock)
	}
	if len(lock.Files) != 4 {
		t.Fatalf("lock file digest count = %d", len(lock.Files))
	}
}

func TestInspectArtifactReadsCapsuleSummary(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	out := filepath.Join(root, "skill.agenix")
	buildResult, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: out})
	if err != nil {
		t.Fatal(err)
	}

	summary, err := InspectArtifact(out)
	if err != nil {
		t.Fatalf("InspectArtifact returned error: %v", err)
	}
	if summary.Skill != buildResult.Skill || summary.Version != buildResult.Version || summary.Digest != buildResult.Digest {
		t.Fatalf("summary mismatch: build=%#v inspect=%#v", buildResult, summary)
	}
	if summary.FileCount != 4 {
		t.Fatalf("FileCount = %d", summary.FileCount)
	}
}

func TestBuildArtifactRejectsDirectoryWithoutManifest(t *testing.T) {
	_, err := BuildArtifact(BuildOptions{SkillDir: t.TempDir(), OutputPath: filepath.Join(t.TempDir(), "out.agenix")})
	if err == nil {
		t.Fatal("expected missing manifest error")
	}
	if !IsErrorClass(err, ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func writeCapsuleSkill(t *testing.T, root string) string {
	t.Helper()
	skillDir := filepath.Join(root, "skill")
	fixture := filepath.Join(skillDir, "fixture")
	if err := os.MkdirAll(filepath.Join(fixture, ".pytest_cache"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(fixture, "__pycache__"), 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(skillDir, "manifest.yaml"): `apiVersion: agenix/v0.1
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
  - type: schema
    name: output_schema_check
    schemaRef: outputs
`,
		filepath.Join(skillDir, "README.md"):                     "# demo\n",
		filepath.Join(fixture, "mathlib.py"):                     "def add(a, b):\n    return a - b\n",
		filepath.Join(fixture, "test_mathlib.py"):                "from mathlib import add\n",
		filepath.Join(skillDir, ".DS_Store"):                     "junk",
		filepath.Join(fixture, ".pytest_cache", "cache"):         "junk",
		filepath.Join(fixture, "__pycache__", "mathlib.pyc"):     "junk",
		filepath.Join(skillDir, ".agenix", "runs", "trace.json"): "junk",
	}
	for path, content := range files {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return skillDir
}

func readTarGzEntries(t *testing.T, path string) map[string][]byte {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	entries := map[string][]byte{}
	for {
		header, err := tr.Next()
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Fatal(err)
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		content := make([]byte, header.Size)
		if _, err := tr.Read(content); err != nil && err.Error() != "EOF" {
			t.Fatal(err)
		}
		entries[header.Name] = content
	}
	return entries
}

func sortedKeys(entries map[string][]byte) []string {
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	return keys
}
