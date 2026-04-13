package agenix

import (
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
