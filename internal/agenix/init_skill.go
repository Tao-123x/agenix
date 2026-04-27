package agenix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	PythonPytestTemplate       = "python-pytest"
	RepoFixTestFailureTemplate = "repo-fix-test-failure"
)

type InitSkillOptions struct {
	Name      string
	Template  string
	OutputDir string
}

type InitSkillResult struct {
	Name     string
	Template string
	Path     string
}

type SkillTemplateDescriptor struct {
	Name        string `json:"name"`
	Adapter     string `json:"adapter"`
	Writes      bool   `json:"writes"`
	Description string `json:"description"`
}

type skillTemplateFile struct {
	RelPath string
	Content string
}

func InitSkill(options InitSkillOptions) (InitSkillResult, error) {
	name := strings.TrimSpace(options.Name)
	if !validSkillName(name) {
		return InitSkillResult{}, NewError(ErrInvalidInput, "invalid skill name: "+options.Name)
	}
	template := strings.TrimSpace(options.Template)
	if template == "" {
		template = PythonPytestTemplate
	}
	files, err := skillTemplateFiles(name, template)
	if err != nil {
		return InitSkillResult{}, NewError(ErrInvalidInput, "unsupported skill template: "+template)
	}
	if strings.TrimSpace(options.OutputDir) == "" {
		return InitSkillResult{}, NewError(ErrInvalidInput, "init skill requires output directory")
	}
	outputDir, err := filepath.Abs(options.OutputDir)
	if err != nil {
		return InitSkillResult{}, WrapError(ErrInvalidInput, "normalize output directory", err)
	}
	outputDir = filepath.Clean(outputDir)
	if err := ensureWritableSkillTarget(outputDir); err != nil {
		return InitSkillResult{}, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return InitSkillResult{}, WrapError(ErrDriverError, "create skill directory", err)
	}
	for _, file := range files {
		target, err := safeJoin(outputDir, file.RelPath)
		if err != nil {
			return InitSkillResult{}, err
		}
		if err := ensureParent(target); err != nil {
			return InitSkillResult{}, WrapError(ErrDriverError, "create template parent", err)
		}
		if err := os.WriteFile(target, []byte(file.Content), 0o600); err != nil {
			return InitSkillResult{}, WrapError(ErrDriverError, "write template file", err)
		}
	}
	return InitSkillResult{Name: name, Template: template, Path: outputDir}, nil
}

func ListSkillTemplates() []SkillTemplateDescriptor {
	return []SkillTemplateDescriptor{
		{
			Name:        PythonPytestTemplate,
			Adapter:     "python-pytest-template",
			Writes:      false,
			Description: "Minimal read-only pytest skill skeleton.",
		},
		{
			Name:        RepoFixTestFailureTemplate,
			Adapter:     "repo-fix-test-failure-template",
			Writes:      true,
			Description: "Writable failing-test repair skill skeleton.",
		},
	}
}

func validSkillName(name string) bool {
	if name == "" || strings.ContainsAny(name, `/\`) {
		return false
	}
	for _, segment := range strings.Split(name, ".") {
		if segment == "" {
			return false
		}
		for _, char := range segment {
			if char >= 'a' && char <= 'z' {
				continue
			}
			if char >= 'A' && char <= 'Z' {
				continue
			}
			if char >= '0' && char <= '9' {
				continue
			}
			if char == '_' || char == '-' {
				continue
			}
			return false
		}
	}
	return true
}

func ensureWritableSkillTarget(outputDir string) error {
	info, err := os.Stat(outputDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return WrapError(ErrInvalidInput, "stat output directory", err)
	}
	if !info.IsDir() {
		return NewError(ErrInvalidInput, "output path exists and is not a directory")
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return WrapError(ErrInvalidInput, "read output directory", err)
	}
	if len(entries) != 0 {
		return NewError(ErrInvalidInput, "output directory is not empty")
	}
	return nil
}

func skillTemplateFiles(name, template string) ([]skillTemplateFile, error) {
	switch template {
	case PythonPytestTemplate:
		return pythonPytestSkillTemplate(name), nil
	case RepoFixTestFailureTemplate:
		return repoFixTestFailureSkillTemplate(name), nil
	default:
		return nil, NewError(ErrInvalidInput, "unsupported skill template: "+template)
	}
}

func pythonPytestSkillTemplate(name string) []skillTemplateFile {
	return []skillTemplateFile{
		{RelPath: "manifest.yaml", Content: pythonPytestManifest(name)},
		{RelPath: "README.md", Content: pythonPytestREADME(name)},
		{RelPath: filepath.ToSlash(filepath.Join("fixture", "skill.py")), Content: pythonPytestFixtureSource()},
		{RelPath: filepath.ToSlash(filepath.Join("fixture", "test_skill.py")), Content: pythonPytestFixtureTest()},
	}
}

func pythonPytestManifest(name string) string {
	return fmt.Sprintf(`apiVersion: agenix/v0.1
kind: Skill
name: %s
version: 0.1.0
description: Generated python pytest skill.
capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 4000
    reasoning_level: minimal
tools:
  - fs
  - shell
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write: []
  shell:
    allow:
      - run: ["python3", "-m", "pytest", "-q"]
inputs:
  repo_path: fixture
outputs:
  required:
    - analysis_summary
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
`, name)
}

func pythonPytestREADME(name string) string {
	return fmt.Sprintf(`# %s

This skill was generated from the Agenix python-pytest template.

From the Agenix repository root:

`+"```bash"+`
go run ./cmd/agenix validate /path/to/%s/manifest.yaml
go run ./cmd/agenix build /path/to/%s -o %s.agenix
go run ./cmd/agenix run %s.agenix --adapter python-pytest-template
`+"```"+`

The template adapter is intentionally local and deterministic. It lists the
fixture through the runtime filesystem tool, returns structured output, and lets
the manifest verifier decide whether the skill is actually valid.
`, name, name, name, name, name)
}

func pythonPytestFixtureSource() string {
	return `def normalize(value):
    return value.strip().lower()
`
}

func pythonPytestFixtureTest() string {
	return `from skill import normalize


def test_normalize_trims_and_lowercases():
    assert normalize(" Hello ") == "hello"
`
}

func repoFixTestFailureSkillTemplate(name string) []skillTemplateFile {
	return []skillTemplateFile{
		{RelPath: "manifest.yaml", Content: repoFixTestFailureManifest(name)},
		{RelPath: "README.md", Content: repoFixTestFailureREADME(name)},
		{RelPath: filepath.ToSlash(filepath.Join("fixture", "mathlib.py")), Content: repoFixTestFailureSource()},
		{RelPath: filepath.ToSlash(filepath.Join("fixture", "test_mathlib.py")), Content: repoFixTestFailureTest()},
	}
}

func repoFixTestFailureManifest(name string) string {
	return fmt.Sprintf(`apiVersion: agenix/v0.1
kind: Skill
name: %s
version: 0.1.0
description: Generated skill that fixes a failing pytest fixture.
capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 4000
    reasoning_level: minimal
tools:
  - fs
  - shell
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
`, name)
}

func repoFixTestFailureREADME(name string) string {
	return fmt.Sprintf(`# %s

This skill was generated from the Agenix repo-fix-test-failure template.

From the Agenix repository root:

`+"```bash"+`
python3 -m pytest -q /path/to/%s/fixture
go run ./cmd/agenix check /path/to/%s --adapter repo-fix-test-failure-template
go run ./cmd/agenix check /path/to/%s --adapter repo-fix-test-failure-template --json
`+"```"+`

The fixture starts with a failing test. The template adapter fixes the source
file only through the runtime filesystem tool, then the verifier decides
whether the patch is valid.
`, name, name, name, name)
}

func repoFixTestFailureSource() string {
	return `def add(a, b):
    return a - b
`
}

func repoFixTestFailureTest() string {
	return `from mathlib import add


def test_adds_numbers():
    assert add(2, 3) == 5
`
}
