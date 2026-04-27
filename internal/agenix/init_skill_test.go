package agenix

import (
	"os"
	"path/filepath"
	"strings"
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

func TestListSkillTemplatesReturnsStableCatalog(t *testing.T) {
	templates := ListSkillTemplates()
	if len(templates) != 2 {
		t.Fatalf("template count = %d", len(templates))
	}
	if templates[0].Name != PythonPytestTemplate || templates[0].Adapter != "python-pytest-template" || templates[0].Writes {
		t.Fatalf("unexpected first template: %#v", templates[0])
	}
	if templates[1].Name != RepoFixTestFailureTemplate || templates[1].Adapter != "repo-fix-test-failure-template" || !templates[1].Writes {
		t.Fatalf("unexpected second template: %#v", templates[1])
	}
}

func TestInitSkillRepoFixTestFailureTemplateCreatesWritableManifest(t *testing.T) {
	root := t.TempDir()
	skillDir := filepath.Join(root, "repo.demo_fix")

	result, err := InitSkill(InitSkillOptions{
		Name:      "repo.demo_fix",
		Template:  RepoFixTestFailureTemplate,
		OutputDir: skillDir,
	})
	if err != nil {
		t.Fatalf("InitSkill returned error: %v", err)
	}
	if result.Name != "repo.demo_fix" || result.Template != RepoFixTestFailureTemplate || result.Path != skillDir {
		t.Fatalf("unexpected init result: %#v", result)
	}

	manifest, err := LoadManifest(filepath.Join(skillDir, "manifest.yaml"))
	if err != nil {
		t.Fatalf("generated manifest did not load: %v", err)
	}
	if manifest.Name != "repo.demo_fix" {
		t.Fatalf("manifest name = %q", manifest.Name)
	}
	if len(manifest.Permissions.Filesystem.Write) != 1 {
		t.Fatalf("write scopes = %#v", manifest.Permissions.Filesystem.Write)
	}
	source, err := os.ReadFile(filepath.Join(skillDir, "fixture", "mathlib.py"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(source), "return a - b") {
		t.Fatalf("fixture should start broken: %s", source)
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
