package agenix

import (
	"fmt"
	"os"
	"path/filepath"
)

type AcceptanceOptions struct {
	RootDir string
	WorkDir string
}

type AcceptanceSummary struct {
	Status     string
	SkillCount int
	RunCount   int
}

type acceptanceSkill struct {
	name          string
	dirName       string
	expectedSkill string
	adapter       Adapter
	changedBase   string
	readOnly      bool
}

func RunV0AcceptanceSweep(options AcceptanceOptions) (AcceptanceSummary, error) {
	rootDir, err := acceptanceRootDir(options.RootDir)
	if err != nil {
		return AcceptanceSummary{Status: "failed"}, err
	}
	workDir, cleanup, err := acceptanceWorkDir(options.WorkDir)
	if err != nil {
		return AcceptanceSummary{Status: "failed"}, err
	}
	defer cleanup()

	skills := []acceptanceSkill{
		{
			name:          "fix-test-failure",
			dirName:       "repo.fix_test_failure",
			expectedSkill: "repo.fix_test_failure",
			changedBase:   "mathlib.py",
		},
		{
			name:          "analyze-test-failures",
			dirName:       "repo.analyze_test_failures",
			expectedSkill: "repo.analyze_test_failures",
			adapter:       HeuristicAnalyzeTestFailuresAdapter{},
			readOnly:      true,
		},
		{
			name:          "apply-small-refactor",
			dirName:       "repo.apply_small_refactor",
			expectedSkill: "repo.apply_small_refactor",
			changedBase:   "greeter.py",
		},
	}

	summary := AcceptanceSummary{Status: "passed", SkillCount: len(skills)}
	for _, skill := range skills {
		runs, err := runAcceptanceSkill(rootDir, workDir, skill)
		if err != nil {
			summary.Status = "failed"
			summary.RunCount += runs
			return summary, err
		}
		summary.RunCount += runs
	}
	return summary, nil
}

func acceptanceRootDir(rootDir string) (string, error) {
	if rootDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", WrapError(ErrDriverError, "get working directory", err)
		}
		rootDir, err = findProjectRoot(cwd)
		if err != nil {
			return "", err
		}
	}
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return "", WrapError(ErrInvalidInput, "normalize acceptance root", err)
	}
	return abs, nil
}

func acceptanceWorkDir(workDir string) (string, func(), error) {
	if workDir == "" {
		tempDir, err := os.MkdirTemp("", "agenix-acceptance-*")
		if err != nil {
			return "", func() {}, WrapError(ErrDriverError, "create acceptance work directory", err)
		}
		return tempDir, func() { _ = os.RemoveAll(tempDir) }, nil
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", func() {}, WrapError(ErrDriverError, "create acceptance work directory", err)
	}
	return workDir, func() {}, nil
}

func findProjectRoot(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", WrapError(ErrInvalidInput, "normalize working directory", err)
	}
	for {
		if fileExists(filepath.Join(dir, "go.mod")) && dirExists(filepath.Join(dir, "examples")) {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", NewError(ErrNotFound, "could not find agenix project root")
		}
		dir = parent
	}
}

func runAcceptanceSkill(rootDir, workDir string, skill acceptanceSkill) (int, error) {
	skillDir := filepath.Join(rootDir, "examples", skill.dirName)
	skillWorkDir := filepath.Join(workDir, skill.name)
	artifactPath := filepath.Join(skillWorkDir, skill.name+".agenix")
	registryRoot := filepath.Join(skillWorkDir, "registry")
	pulledPath := filepath.Join(skillWorkDir, "pulled.agenix")

	if kind, _, err := ValidateTarget(filepath.Join(skillDir, "manifest.yaml")); err != nil {
		return 0, WrapError(ErrorClass(err), "validate manifest for "+skill.expectedSkill, err)
	} else if kind != "manifest" {
		return 0, NewError(ErrInvalidInput, fmt.Sprintf("validate manifest for %s returned kind %q", skill.expectedSkill, kind))
	}

	buildSummary, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath})
	if err != nil {
		return 0, WrapError(ErrorClass(err), "build artifact for "+skill.expectedSkill, err)
	}
	if buildSummary.Skill != skill.expectedSkill {
		return 0, NewError(ErrInvalidInput, fmt.Sprintf("build artifact for %s returned skill %q", skill.expectedSkill, buildSummary.Skill))
	}

	inspectSummary, err := InspectArtifact(artifactPath)
	if err != nil {
		return 0, WrapError(ErrorClass(err), "inspect artifact for "+skill.expectedSkill, err)
	}
	if inspectSummary.Skill != skill.expectedSkill {
		return 0, NewError(ErrInvalidInput, fmt.Sprintf("inspect artifact for %s returned skill %q", skill.expectedSkill, inspectSummary.Skill))
	}

	runCount := 0
	if _, err := runAcceptanceTargetOnce(artifactPath, filepath.Join(skillWorkDir, ".agenix-runs"), "", skill); err != nil {
		return runCount, err
	}
	runCount++

	entry, err := PublishArtifact(PublishOptions{ArtifactPath: artifactPath, RegistryRoot: registryRoot})
	if err != nil {
		return runCount, WrapError(ErrorClass(err), "publish artifact for "+skill.expectedSkill, err)
	}
	if entry.Skill != skill.expectedSkill {
		return runCount, NewError(ErrInvalidInput, fmt.Sprintf("publish artifact for %s returned skill %q", skill.expectedSkill, entry.Skill))
	}

	pulledSummary, err := PullArtifact(PullOptions{
		Reference:    skill.expectedSkill + "@0.1.0",
		OutputPath:   pulledPath,
		RegistryRoot: registryRoot,
	})
	if err != nil {
		return runCount, WrapError(ErrorClass(err), "pull artifact for "+skill.expectedSkill, err)
	}
	if pulledSummary.Skill != skill.expectedSkill {
		return runCount, NewError(ErrInvalidInput, fmt.Sprintf("pull artifact for %s returned skill %q", skill.expectedSkill, pulledSummary.Skill))
	}
	if _, err := InspectArtifact(pulledPath); err != nil {
		return runCount, WrapError(ErrorClass(err), "inspect pulled artifact for "+skill.expectedSkill, err)
	}

	if _, err := ListRegistryEntries(registryRoot); err != nil {
		return runCount, WrapError(ErrorClass(err), "list registry for "+skill.expectedSkill, err)
	}
	if _, err := ShowRegistrySkill(skill.expectedSkill, registryRoot); err != nil {
		return runCount, WrapError(ErrorClass(err), "show registry skill for "+skill.expectedSkill, err)
	}
	if _, err := ResolveRegistryEntry(skill.expectedSkill+"@0.1.0", registryRoot); err != nil {
		return runCount, WrapError(ErrorClass(err), "resolve registry skill for "+skill.expectedSkill, err)
	}

	if _, err := runAcceptanceTargetOnce(skill.expectedSkill+"@0.1.0", filepath.Join(skillWorkDir, ".registry-runs"), registryRoot, skill); err != nil {
		return runCount, err
	}
	runCount++

	return runCount, nil
}

func runAcceptanceTargetOnce(target, runDir, registryRoot string, skill acceptanceSkill) (*Trace, error) {
	result, err := Run(RunOptions{
		ManifestPath: target,
		RunDir:       runDir,
		RegistryRoot: registryRoot,
		Adapter:      skill.adapter,
	})
	if err != nil {
		return nil, WrapError(ErrorClass(err), "run "+skill.expectedSkill, err)
	}
	if result.Status != "passed" {
		return nil, NewError(ErrVerificationFailed, fmt.Sprintf("run %s status = %q", skill.expectedSkill, result.Status))
	}
	if result.TracePath == "" {
		return nil, NewError(ErrDriverError, "run "+skill.expectedSkill+" did not write a trace")
	}
	if err := validateAcceptanceRun(skill, result); err != nil {
		return nil, err
	}
	if kind, _, err := ValidateTarget(result.TracePath); err != nil {
		return nil, WrapError(ErrorClass(err), "validate trace for "+skill.expectedSkill, err)
	} else if kind != "trace" {
		return nil, NewError(ErrInvalidInput, fmt.Sprintf("validate trace for %s returned kind %q", skill.expectedSkill, kind))
	}
	if verify, err := Verify(result.TracePath); err != nil {
		return nil, WrapError(ErrorClass(err), "verify trace for "+skill.expectedSkill, err)
	} else if verify.Status != "passed" {
		return nil, NewError(ErrVerificationFailed, fmt.Sprintf("verify trace for %s status = %q", skill.expectedSkill, verify.Status))
	}
	replay, err := Replay(result.TracePath)
	if err != nil {
		return nil, WrapError(ErrorClass(err), "replay trace for "+skill.expectedSkill, err)
	}
	if replay.FinalStatus != "passed" {
		return nil, NewError(ErrVerificationFailed, fmt.Sprintf("replay trace for %s status = %q", skill.expectedSkill, replay.FinalStatus))
	}
	if len(replay.Events) == 0 {
		return nil, NewError(ErrVerificationFailed, "replay trace for "+skill.expectedSkill+" returned no events")
	}

	trace, err := ReadTrace(result.TracePath)
	if err != nil {
		return nil, WrapError(ErrorClass(err), "read trace for "+skill.expectedSkill, err)
	}
	if trace.Final.Status != "passed" {
		return nil, NewError(ErrVerificationFailed, fmt.Sprintf("trace for %s status = %q", skill.expectedSkill, trace.Final.Status))
	}
	if err := validateAcceptanceTrace(skill, trace); err != nil {
		return nil, err
	}
	return trace, nil
}

func validateAcceptanceRun(skill acceptanceSkill, result RunResult) error {
	if skill.readOnly {
		if len(result.ChangedFiles) != 0 {
			return NewError(ErrVerificationFailed, fmt.Sprintf("expected %s to report no changed files, got %v", skill.expectedSkill, result.ChangedFiles))
		}
		return nil
	}
	if len(result.ChangedFiles) != 1 || filepath.Base(result.ChangedFiles[0]) != skill.changedBase {
		return NewError(ErrVerificationFailed, fmt.Sprintf("expected %s changed_files to contain %s, got %v", skill.expectedSkill, skill.changedBase, result.ChangedFiles))
	}
	return nil
}

func validateAcceptanceTrace(skill acceptanceSkill, trace *Trace) error {
	if skill.readOnly {
		if !acceptanceTraceHasAdapterEvent(*trace, "execute", "ok") {
			return NewError(ErrVerificationFailed, "expected successful adapter.execute event for "+skill.expectedSkill)
		}
		if acceptanceTraceHasEvent(*trace, "tool_call", "fs.write") {
			return NewError(ErrVerificationFailed, "read-only skill emitted fs.write: "+skill.expectedSkill)
		}
		return nil
	}
	if !acceptanceTraceHasEvent(*trace, "tool_call", "fs.write") {
		return NewError(ErrVerificationFailed, "expected fs.write event for "+skill.expectedSkill)
	}
	return nil
}

func acceptanceTraceHasEvent(trace Trace, eventType, name string) bool {
	for _, event := range trace.Events {
		if event.Type == eventType && event.Name == name {
			return true
		}
	}
	return false
}

func acceptanceTraceHasAdapterEvent(trace Trace, name, status string) bool {
	for _, event := range trace.Events {
		if event.Type == "adapter" && event.Name == name && event.Status == status {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
