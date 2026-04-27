package agenix

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitSkillPythonPytestTemplateCreatesValidManifest(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_skill")

	result, err := InitSkill(InitSkillOptions{
		Name:      "repo.demo_skill",
		Template:  PythonPytestTemplate,
		OutputDir: skillDir,
	})
	if err != nil {
		t.Fatalf("InitSkill returned error: %v", err)
	}
	if result.Name != "repo.demo_skill" || result.Template != PythonPytestTemplate || result.Path != skillDir {
		t.Fatalf("unexpected init result: %#v", result)
	}

	manifest, err := LoadManifest(filepath.Join(skillDir, "manifest.yaml"))
	if err != nil {
		t.Fatalf("generated manifest did not load: %v", err)
	}
	if manifest.Name != "repo.demo_skill" {
		t.Fatalf("manifest name = %q", manifest.Name)
	}
	if _, err := os.Stat(filepath.Join(skillDir, "fixture", "test_skill.py")); err != nil {
		t.Fatalf("fixture test missing: %v", err)
	}
}

func TestInitSkillRejectsNonEmptyOutputDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "existing.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := InitSkill(InitSkillOptions{
		Name:      "repo.demo_skill",
		Template:  PythonPytestTemplate,
		OutputDir: root,
	})
	if err == nil {
		t.Fatal("expected non-empty output directory failure")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("error class = %s, want %s", ErrorClass(err), ErrInvalidInput)
	}
}

func TestInitSkillRejectsInvalidName(t *testing.T) {
	_, err := InitSkill(InitSkillOptions{
		Name:      "../escape",
		Template:  PythonPytestTemplate,
		OutputDir: t.TempDir(),
	})
	if err == nil {
		t.Fatal("expected invalid name failure")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("error class = %s, want %s", ErrorClass(err), ErrInvalidInput)
	}
}
