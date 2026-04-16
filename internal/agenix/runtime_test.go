package agenix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteApplySmallRefactorHandlesCRLFInput(t *testing.T) {
	repo := t.TempDir()
	target := filepath.Join(repo, "greeter.py")
	content := "def greeting(first, last):\r\n    return \"Hello, \" + first.strip() + \" \" + last.strip() + \"!\"\r\n"
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	permissions := Permissions{}
	permissions.Filesystem.Read = []string{repo}
	permissions.Filesystem.Write = []string{repo}
	policy, err := NewPolicy(permissions)
	if err != nil {
		t.Fatal(err)
	}
	trace := NewTrace("repo.apply_small_refactor", "fake", permissions)
	manifest := Manifest{Inputs: map[string]string{"repo_path": repo}}

	output, err := executeApplySmallRefactor(manifest, NewTools(policy, trace))
	if err != nil {
		t.Fatalf("executeApplySmallRefactor returned error: %v", err)
	}

	updated, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	text := string(updated)
	if !strings.Contains(text, "def full_name(first, last):") {
		t.Fatalf("expected full_name helper in %q", text)
	}
	if !strings.Contains(text, "\r\n") {
		t.Fatalf("expected CRLF line endings to be preserved in %q", text)
	}

	changed, ok := output["changed_files"].([]string)
	if !ok || len(changed) != 1 || changed[0] != target {
		t.Fatalf("changed_files = %#v", output["changed_files"])
	}
}
