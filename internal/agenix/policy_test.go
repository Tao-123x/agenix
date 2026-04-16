package agenix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPolicyAllowsScopedWriteAndRejectsEscape(t *testing.T) {
	root := t.TempDir()
	policy, err := NewPolicy(Permissions{
		Network: false,
		Filesystem: FilesystemPermissions{
			Read:  []string{root},
			Write: []string{root},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := policy.CheckWrite(filepath.Join(root, "pkg", "file.py")); err != nil {
		t.Fatalf("expected scoped write to pass: %v", err)
	}

	err = policy.CheckWrite(filepath.Join(root, "..", "outside.py"))
	if err == nil {
		t.Fatal("expected escaped write to fail")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
}

func TestPolicyRequiresExactShellAllowlistMatch(t *testing.T) {
	policy, err := NewPolicy(Permissions{
		Shell: ShellPermissions{
			Allow: []ShellCommand{{Run: []string{"python3", "-m", "pytest", "-q"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := policy.CheckShell([]string{"python3", "-m", "pytest", "-q"}); err != nil {
		t.Fatalf("expected allowed command: %v", err)
	}

	err = policy.CheckShell([]string{"python3", "-m", "pip", "install", "pytest"})
	if err == nil {
		t.Fatal("expected unlisted command to fail")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
}

func TestPolicyDoesNotGrantAliasCommandDirectly(t *testing.T) {
	policy, err := NewPolicy(Permissions{
		Shell: ShellPermissions{
			Allow: []ShellCommand{{Run: []string{"python3", "-m", "pytest", "-q"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = policy.CheckShell([]string{"python", "-m", "pytest", "-q"})
	if err == nil {
		t.Fatal("expected direct python request to fail when only python3 is allowlisted")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
}

func TestPolicyRejectsReadAndWriteThroughScopedSymlink(t *testing.T) {
	root := t.TempDir()
	outsideDir := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(root, "escape")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		if isSymlinkUnsupported(err) {
			t.Skipf("symlink unsupported on this host: %v", err)
		}
		t.Fatal(err)
	}

	policy, err := NewPolicy(Permissions{
		Network: false,
		Filesystem: FilesystemPermissions{
			Read:  []string{root},
			Write: []string{root},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	readPath := filepath.Join(linkPath, "secret.txt")
	if err := policy.CheckRead(readPath); err == nil {
		t.Fatal("expected symlinked read to fail")
	} else if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation for read, got %v", err)
	}

	writePath := filepath.Join(linkPath, "escape.txt")
	if err := policy.CheckWrite(writePath); err == nil {
		t.Fatal("expected symlinked write to fail")
	} else if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation for write, got %v", err)
	}
}

func TestToolsFSWriteRejectsScopedSymlinkEscapeWithoutWritingOutside(t *testing.T) {
	root := t.TempDir()
	outsideDir := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatal(err)
	}
	linkPath := filepath.Join(root, "escape")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		if isSymlinkUnsupported(err) {
			t.Skipf("symlink unsupported on this host: %v", err)
		}
		t.Fatal(err)
	}

	policy, err := NewPolicy(Permissions{
		Filesystem: FilesystemPermissions{
			Read:  []string{root},
			Write: []string{root},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	trace := NewTrace("test", "fake-scripted", Permissions{})
	tools := NewTools(policy, trace)
	target := filepath.Join(linkPath, "escape.txt")

	err = tools.FSWrite(target, "escape", true)
	if err == nil {
		t.Fatal("expected fs.write to fail")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(outsideDir, "escape.txt")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no file to be written outside scope, stat err=%v", statErr)
	}
}

func TestPolicyWithBaseResolvesRepoRelativePathAcrossProcessCWD(t *testing.T) {
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
	loaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := NewPolicyWithBase(loaded.Permissions, filepath.Dir(manifestPath))
	if err != nil {
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
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	if err := policy.CheckWrite(filepath.Join("fixture", "in_scope.py")); err != nil {
		t.Fatalf("expected repo-relative in-scope path to pass, got %v", err)
	}
	if err := policy.CheckWrite(filepath.Join("fixture", "..", "..", "outside.py")); err == nil {
		t.Fatal("expected repo-relative escape path to fail")
	} else if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation for escaped repo-relative path, got %v", err)
	}
}

func isSymlinkUnsupported(err error) bool {
	return os.IsPermission(err)
}
