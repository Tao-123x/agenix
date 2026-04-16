package agenix

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
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
	if lock.CreatedAt.IsZero() {
		t.Fatalf("missing created_at: %#v", lock)
	}
	if lock.Provenance.BuiltBy == "" || lock.Provenance.BuildHost == "" {
		t.Fatalf("missing build provenance: %#v", lock.Provenance)
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
	if summary.CreatedAt.IsZero() {
		t.Fatalf("missing summary created_at: %#v", summary)
	}
	if summary.BuiltBy == "" || summary.BuildHost == "" {
		t.Fatalf("missing summary provenance: %#v", summary)
	}
}

func TestInspectArtifactRejectsTamperedPayload(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	out := filepath.Join(root, "skill.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: out}); err != nil {
		t.Fatal(err)
	}
	rewriteArtifactEntry(t, out, "files/README.md", []byte("# hack\n"))

	_, err := InspectArtifact(out)
	if err == nil {
		t.Fatal("expected InspectArtifact to reject tampered artifact")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %v", err)
	}
}

func TestMaterializeArtifactRejectsTamperedPayload(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	out := filepath.Join(root, "skill.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: out}); err != nil {
		t.Fatal(err)
	}
	rewriteArtifactEntry(t, out, "files/fixture/mathlib.py", []byte("def add(a, b):\n    return a + b\n"))

	_, _, err := MaterializeArtifact(out, filepath.Join(root, "workspace"))
	if err == nil {
		t.Fatal("expected MaterializeArtifact to reject tampered artifact")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "digest mismatch") {
		t.Fatalf("expected digest mismatch error, got %v", err)
	}
}

func TestMaterializeArtifactRejectsPreexistingWorkspaceSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	out := filepath.Join(root, "skill.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: out}); err != nil {
		t.Fatal(err)
	}

	workspace := filepath.Join(root, "workspace")
	outsideDir := filepath.Join(root, "outside")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outsideDir, filepath.Join(workspace, "fixture")); err != nil {
		if isSymlinkUnsupported(err) {
			t.Skipf("symlink unsupported on this host: %v", err)
		}
		t.Fatal(err)
	}

	_, _, err := MaterializeArtifact(out, workspace)
	if err == nil {
		t.Fatal("expected symlinked workspace escape to be rejected")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(outsideDir, "mathlib.py")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no payload to be written outside workspace, stat err=%v", statErr)
	}
}

func TestInspectArtifactRejectsUnlockedPayload(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	out := filepath.Join(root, "skill.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: out}); err != nil {
		t.Fatal(err)
	}
	appendArtifactEntry(t, out, "files/fixture/backdoor.py", []byte("print('unexpected')\n"))

	_, err := InspectArtifact(out)
	if err == nil {
		t.Fatal("expected InspectArtifact to reject unlocked payload")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
	if !strings.Contains(err.Error(), "unexpected artifact payload") {
		t.Fatalf("expected unexpected payload error, got %v", err)
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

func rewriteArtifactEntry(t *testing.T, path, entryName string, replacement []byte) {
	t.Helper()
	rewriteArtifact(t, path, func(entries []tarEntry) []tarEntry {
		found := false
		for i := range entries {
			if entries[i].header.Name == entryName {
				entries[i].body = replacement
				entries[i].header.Size = int64(len(replacement))
				found = true
			}
		}
		if !found {
			t.Fatalf("artifact missing entry %s", entryName)
		}
		return entries
	})
}

func appendArtifactEntry(t *testing.T, path, entryName string, body []byte) {
	t.Helper()
	rewriteArtifact(t, path, func(entries []tarEntry) []tarEntry {
		return append(entries, tarEntry{
			header: tar.Header{Name: entryName, Mode: 0o600, Size: int64(len(body))},
			body:   body,
		})
	})
}

type tarEntry struct {
	header tar.Header
	body   []byte
}

func rewriteArtifact(t *testing.T, path string, mutate func([]tarEntry) []tarEntry) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	var entries []tarEntry
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		clone := *header
		entries = append(entries, tarEntry{header: clone, body: body})
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	entries = mutate(entries)

	var buf bytes.Buffer
	outGz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(outGz)
	for _, entry := range entries {
		entry.header.Size = int64(len(entry.body))
		if err := tw.WriteHeader(&entry.header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(entry.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := outGz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
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
