package agenix

import (
	"os"
	"path/filepath"
	"strings"
)

const CheckReportKind = "check_report"

type CheckOptions struct {
	Target       string
	RegistryRoot string
	WorkDir      string
	Adapter      Adapter
}

type CheckResult struct {
	Kind            string   `json:"kind"`
	Status          string   `json:"status"`
	Skill           string   `json:"skill"`
	Version         string   `json:"version"`
	ArtifactPath    string   `json:"artifact_path"`
	RunID           string   `json:"run_id"`
	TracePath       string   `json:"trace_path"`
	ChangedFiles    []string `json:"changed_files"`
	VerifierSummary []string `json:"verifier_summary"`
	EventCount      int      `json:"event_count"`
	ErrorClass      string   `json:"error_class,omitempty"`
	ErrorMessage    string   `json:"error_message,omitempty"`
}

func CheckSkill(options CheckOptions) (CheckResult, error) {
	result := newCheckResult()
	if strings.TrimSpace(options.Target) == "" {
		return result, NewError(ErrInvalidInput, "check requires target")
	}
	target, err := ResolveRegistryReference(options.Target, options.RegistryRoot)
	if err != nil {
		return result, err
	}
	artifactPath, skill, version, err := checkArtifactTarget(target, options.WorkDir)
	if err != nil {
		return result, err
	}
	result.Skill = skill
	result.Version = version
	result.ArtifactPath = artifactPath

	runResult, err := Run(RunOptions{ManifestPath: artifactPath, Adapter: options.Adapter})
	result.RunID = runResult.RunID
	result.TracePath = runResult.TracePath
	result.ChangedFiles = runResult.ChangedFiles
	result.VerifierSummary = runResult.VerifierSummary
	if err != nil {
		result = result.withTraceEventCount()
		return result, err
	}
	if _, _, err := ValidateTarget(runResult.TracePath); err != nil {
		return result, err
	}
	verifyResult, err := Verify(runResult.TracePath)
	result.ChangedFiles = verifyResult.ChangedFiles
	result.VerifierSummary = verifyResult.VerifierSummary
	if err != nil {
		return result, err
	}
	replay, err := Replay(runResult.TracePath)
	if err != nil {
		return result, err
	}
	if replay.FinalStatus != "passed" {
		return result, NewError(ErrVerificationFailed, "checked trace final status is not passed")
	}
	if replay.EventCount == 0 {
		return result, NewError(ErrVerificationFailed, "checked trace contains no events")
	}
	result.Status = "passed"
	result.EventCount = replay.EventCount
	return result.ensureArrays(), nil
}

func NewFailedCheckResult(err error) CheckResult {
	return newCheckResult().WithError(err)
}

func (result CheckResult) WithError(err error) CheckResult {
	result = result.ensureArrays()
	if strings.TrimSpace(result.Kind) == "" {
		result.Kind = CheckReportKind
	}
	result.Status = "failed"
	if err != nil {
		result.ErrorClass = ErrorClass(err)
		result.ErrorMessage = err.Error()
	}
	return result
}

func newCheckResult() CheckResult {
	return CheckResult{
		Kind:            CheckReportKind,
		Status:          "failed",
		ChangedFiles:    []string{},
		VerifierSummary: []string{},
	}
}

func (result CheckResult) ensureArrays() CheckResult {
	if result.ChangedFiles == nil {
		result.ChangedFiles = []string{}
	}
	if result.VerifierSummary == nil {
		result.VerifierSummary = []string{}
	}
	return result
}

func (result CheckResult) withTraceEventCount() CheckResult {
	if strings.TrimSpace(result.TracePath) == "" {
		return result
	}
	replay, err := Replay(result.TracePath)
	if err != nil {
		return result
	}
	result.EventCount = replay.EventCount
	return result
}

func checkArtifactTarget(target, workDir string) (string, string, string, error) {
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return "", "", "", WrapError(ErrInvalidInput, "normalize check target", err)
	}
	absTarget = filepath.Clean(absTarget)
	info, err := os.Stat(absTarget)
	if err != nil {
		return "", "", "", WrapError(ErrNotFound, "stat check target", err)
	}
	if !info.IsDir() && isArtifactTarget(absTarget) {
		summary, err := InspectArtifact(absTarget)
		if err != nil {
			return "", "", "", err
		}
		return absTarget, summary.Skill, summary.Version, nil
	}
	skillDir := absTarget
	manifestPath := filepath.Join(skillDir, "manifest.yaml")
	if !info.IsDir() {
		manifestPath = absTarget
		skillDir = filepath.Dir(absTarget)
	}
	if kind, _, err := ValidateTarget(manifestPath); err != nil {
		return "", "", "", err
	} else if kind != "manifest" {
		return "", "", "", NewError(ErrInvalidInput, "check target is not a manifest")
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return "", "", "", err
	}
	checkDir := filepath.Join(checkRoot(workDir), newRunID())
	artifactPath := filepath.Join(checkDir, checkArtifactFilename(manifest.Name, manifest.Version))
	summary, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath})
	if err != nil {
		return "", "", "", err
	}
	return summary.Path, summary.Skill, summary.Version, nil
}

func checkRoot(workDir string) string {
	if strings.TrimSpace(workDir) == "" {
		return filepath.Join(".agenix", "checks")
	}
	return workDir
}

func checkArtifactFilename(skill, version string) string {
	name := strings.Map(func(char rune) rune {
		if char >= 'a' && char <= 'z' {
			return char
		}
		if char >= 'A' && char <= 'Z' {
			return char
		}
		if char >= '0' && char <= '9' {
			return char
		}
		if char == '.' || char == '_' || char == '-' {
			return char
		}
		return '_'
	}, skill+"-"+version)
	if strings.Trim(name, "._-") == "" {
		name = "skill"
	}
	return name + ".agenix"
}
