package agenix

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

const fakeModelProfile = "fake-scripted"

type RunOptions struct {
	ManifestPath string
	RunDir       string
	Adapter      Adapter
}

type RunResult struct {
	Status          string   `json:"status"`
	RunID           string   `json:"run_id"`
	TracePath       string   `json:"trace_path"`
	ChangedFiles    []string `json:"changed_files"`
	VerifierSummary []string `json:"verifier_summary"`
}

type Adapter interface {
	Execute(manifest Manifest, tools *Tools) (map[string]any, error)
}

type FakeFixTestFailureAdapter struct{}

type EscapeAdapter struct {
	Path string
}

func Run(options RunOptions) (RunResult, error) {
	runID := newRunID()
	manifestPath := options.ManifestPath
	if isArtifactTarget(manifestPath) {
		workspaceDir := filepath.Join(runRoot(options.RunDir), runID, "workspace")
		materializedManifest, _, err := MaterializeArtifact(manifestPath, workspaceDir)
		if err != nil {
			return RunResult{RunID: runID, TracePath: tracePathFor(options.RunDir, runID), Status: "failed"}, err
		}
		manifestPath = materializedManifest
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return RunResult{}, err
	}
	trace := NewTrace(manifest.Name, fakeModelProfile, manifest.Permissions)
	trace.RunID = runID
	trace.ManifestPath = manifestPath
	tracePath := tracePathFor(options.RunDir, runID)
	result := RunResult{RunID: runID, TracePath: tracePath}

	policy, err := NewPolicy(manifest.Permissions)
	if err != nil {
		trace.SetFinal("failed", nil, err.Error())
		_ = WriteTrace(tracePath, trace)
		return result, err
	}
	adapter := options.Adapter
	if adapter == nil {
		adapter = FakeFixTestFailureAdapter{}
	}

	output, err := adapter.Execute(manifest, NewTools(policy, trace))
	if err != nil {
		trace.SetFinal("failed", output, err.Error())
		_ = WriteTrace(tracePath, trace)
		result.Status = "failed"
		return result, err
	}
	if err := RunVerifiers(manifest, output, trace); err != nil {
		trace.SetFinal("failed", output, err.Error())
		_ = WriteTrace(tracePath, trace)
		result.Status = "failed"
		result.VerifierSummary = verifierSummary(trace)
		return result, err
	}
	trace.SetFinal("passed", output, "")
	if err := WriteTrace(tracePath, trace); err != nil {
		return result, err
	}
	result.Status = "passed"
	result.ChangedFiles = outputStrings(output, "changed_files")
	result.VerifierSummary = verifierSummary(trace)
	return result, nil
}

func (FakeFixTestFailureAdapter) Execute(manifest Manifest, tools *Tools) (map[string]any, error) {
	repoPath := manifest.Inputs["repo_path"]
	target := filepath.Join(repoPath, "mathlib.py")
	content, err := tools.FSRead(target)
	if err != nil {
		return nil, err
	}
	fixed := strings.Replace(content, "return a - b", "return a + b", 1)
	if fixed != content {
		if err := tools.FSWrite(target, fixed, true); err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"patch_summary": "Replaced subtraction with addition in mathlib.add.",
		"changed_files": []string{target},
	}, nil
}

func (adapter EscapeAdapter) Execute(_ Manifest, tools *Tools) (map[string]any, error) {
	err := tools.FSWrite(adapter.Path, "escape", true)
	return map[string]any{"patch_summary": "attempted escape", "changed_files": []string{adapter.Path}}, err
}

func Verify(path string) (RunResult, error) {
	trace, err := ReadTrace(path)
	if err != nil {
		return RunResult{}, err
	}
	if trace.ManifestPath == "" {
		return RunResult{}, NewError(ErrInvalidInput, "trace does not include manifest_path")
	}
	if trace.Final.Status != "passed" {
		return RunResult{Status: "failed", RunID: trace.RunID, TracePath: path}, NewError(ErrVerificationFailed, "cannot verify a trace whose final status is not passed")
	}
	if traceHasPolicyViolation(trace) {
		return RunResult{Status: "failed", RunID: trace.RunID, TracePath: path}, NewError(ErrVerificationFailed, "trace contains policy violation event")
	}
	manifest, err := LoadManifest(trace.ManifestPath)
	if err != nil {
		return RunResult{}, err
	}
	output, ok := trace.Final.Output.(map[string]any)
	if !ok {
		raw, _ := json.Marshal(trace.Final.Output)
		_ = json.Unmarshal(raw, &output)
	}
	policy, err := NewPolicy(manifest.Permissions)
	if err != nil {
		return RunResult{}, err
	}
	for _, changedFile := range outputStrings(output, "changed_files") {
		if err := policy.CheckWrite(changedFile); err != nil {
			return RunResult{Status: "failed", RunID: trace.RunID, TracePath: path}, NewError(ErrVerificationFailed, "changed file outside write scope: "+changedFile)
		}
	}
	verifyTrace := NewTrace(manifest.Name, trace.ModelProfile, manifest.Permissions)
	if err := RunVerifiers(manifest, output, verifyTrace); err != nil {
		return RunResult{Status: "failed", RunID: trace.RunID, TracePath: path, VerifierSummary: verifierSummary(verifyTrace)}, err
	}
	return RunResult{Status: "passed", RunID: trace.RunID, TracePath: path, ChangedFiles: outputStrings(output, "changed_files"), VerifierSummary: verifierSummary(verifyTrace)}, nil
}

type ReplaySummary struct {
	RunID       string
	Skill       string
	FinalStatus string
	EventCount  int
}

func Replay(path string) (ReplaySummary, error) {
	trace, err := ReadTrace(path)
	if err != nil {
		return ReplaySummary{}, err
	}
	return ReplaySummary{
		RunID:       trace.RunID,
		Skill:       trace.Skill,
		FinalStatus: trace.Final.Status,
		EventCount:  len(trace.Events),
	}, nil
}

func tracePathFor(runDir, runID string) string {
	return filepath.Join(runRoot(runDir), runID, "trace.json")
}

func runRoot(runDir string) string {
	if runDir == "" {
		return filepath.Join(".agenix", "runs")
	}
	return runDir
}

func isArtifactTarget(path string) bool {
	return strings.HasSuffix(path, ".agenix")
}

func verifierSummary(trace *Trace) []string {
	summary := []string{}
	for _, event := range trace.Events {
		if event.Type == "verifier" {
			summary = append(summary, event.Name+":"+event.Status)
		}
	}
	return summary
}

func traceHasPolicyViolation(trace *Trace) bool {
	for _, event := range trace.Events {
		if eventErrorClass(event.Error) == ErrPolicyViolation {
			return true
		}
	}
	return false
}

func eventErrorClass(value interface{}) string {
	switch typed := value.(type) {
	case map[string]string:
		return typed["class"]
	case map[string]interface{}:
		if class, ok := typed["class"].(string); ok {
			return class
		}
	}
	return ""
}
