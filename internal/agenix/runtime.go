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
	RegistryRoot string
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
	Metadata() AdapterMetadata
	Execute(manifest Manifest, tools *Tools) (map[string]any, error)
}

type AdapterMetadata struct {
	Name            string        `json:"name"`
	ModelProfile    string        `json:"model_profile"`
	SupportedSkills []string      `json:"supported_skills,omitempty"`
	Capabilities    CapabilitySet `json:"capabilities"`
}

type FakeFixTestFailureAdapter struct{}

type EscapeAdapter struct {
	Path string
}

func Run(options RunOptions) (RunResult, error) {
	runID := newRunID()
	manifestPath, err := ResolveRegistryReference(options.ManifestPath, options.RegistryRoot)
	if err != nil {
		return RunResult{RunID: runID, TracePath: tracePathFor(options.RunDir, runID), Status: "failed"}, err
	}
	if isArtifactTarget(manifestPath) {
		workspaceDir := filepath.Join(runRoot(options.RunDir), runID, "workspace")
		materializedManifest, _, err := MaterializeArtifact(manifestPath, workspaceDir)
		if err != nil {
			return RunResult{RunID: runID, TracePath: tracePathFor(options.RunDir, runID), Status: "failed"}, err
		}
		manifestPath = materializedManifest
	}
	absoluteManifestPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return RunResult{}, WrapError(ErrInvalidInput, "normalize manifest path", err)
	}
	manifestPath = filepath.Clean(absoluteManifestPath)
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return RunResult{}, err
	}
	adapter := options.Adapter
	if adapter == nil {
		adapter = FakeFixTestFailureAdapter{}
	}
	metadata := adapter.Metadata()
	if metadata.ModelProfile == "" {
		metadata.ModelProfile = fakeModelProfile
	}
	if metadata.Name == "" {
		metadata.Name = metadata.ModelProfile
	}
	trace := NewTrace(manifest.Name, metadata.ModelProfile, manifest.Permissions)
	trace.RunID = runID
	trace.ManifestPath = manifestPath
	trace.SetRedaction(manifest.Redaction)
	tracePath := tracePathFor(options.RunDir, runID)
	result := RunResult{RunID: runID, TracePath: tracePath}
	trace.AddAdapterEvent("selection", "ok", map[string]string{"skill": manifest.Name}, metadata, nil)
	if err := validateAdapter(manifest, metadata); err != nil {
		trace.AddAdapterEvent("capability_check", "failed", manifest.Capabilities.Requires, metadata, err)
		trace.SetFinal("failed", nil, err.Error())
		_ = WriteTrace(tracePath, trace)
		result.Status = "failed"
		return result, err
	}
	trace.AddAdapterEvent("capability_check", "ok", manifest.Capabilities.Requires, metadata.Capabilities, nil)

	policy, err := NewPolicy(manifest.Permissions)
	if err != nil {
		trace.SetFinal("failed", nil, err.Error())
		_ = WriteTrace(tracePath, trace)
		return result, err
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

func (FakeFixTestFailureAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "fake-scripted",
		ModelProfile:    fakeModelProfile,
		SupportedSkills: []string{"repo.fix_test_failure", "repo.analyze_test_failures", "repo.apply_small_refactor"},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 32000,
			ReasoningLevel:   "medium",
		},
	}
}

func (FakeFixTestFailureAdapter) Execute(manifest Manifest, tools *Tools) (map[string]any, error) {
	switch manifest.Name {
	case "repo.fix_test_failure":
		return executeFixTestFailure(manifest, tools)
	case "repo.analyze_test_failures":
		return executeAnalyzeTestFailures(manifest, tools)
	case "repo.apply_small_refactor":
		return executeApplySmallRefactor(manifest, tools)
	default:
		return nil, NewError(ErrInvalidInput, "fake adapter does not support skill: "+manifest.Name)
	}
}

func validateAdapter(manifest Manifest, metadata AdapterMetadata) error {
	if len(metadata.SupportedSkills) > 0 && !containsString(metadata.SupportedSkills, manifest.Name) {
		return NewError(ErrInvalidInput, "adapter "+metadata.Name+" does not support skill: "+manifest.Name)
	}
	required := manifest.Capabilities.Requires
	if required.ToolCalling && !metadata.Capabilities.ToolCalling {
		return NewError(ErrInvalidInput, "adapter "+metadata.Name+" missing capability: tool_calling")
	}
	if required.StructuredOutput && !metadata.Capabilities.StructuredOutput {
		return NewError(ErrInvalidInput, "adapter "+metadata.Name+" missing capability: structured_output")
	}
	if required.MaxContextTokens > 0 && metadata.Capabilities.MaxContextTokens < required.MaxContextTokens {
		return NewError(ErrInvalidInput, "adapter "+metadata.Name+" max_context_tokens too small")
	}
	if required.ReasoningLevel != "" && reasoningRank(metadata.Capabilities.ReasoningLevel) < reasoningRank(required.ReasoningLevel) {
		return NewError(ErrInvalidInput, "adapter "+metadata.Name+" reasoning_level too low")
	}
	return nil
}

func reasoningRank(level string) int {
	switch strings.ToLower(level) {
	case "minimal":
		return 1
	case "low":
		return 2
	case "medium":
		return 3
	case "high":
		return 4
	case "xhigh":
		return 5
	default:
		return 0
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func executeFixTestFailure(manifest Manifest, tools *Tools) (map[string]any, error) {
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

func executeAnalyzeTestFailures(manifest Manifest, tools *Tools) (map[string]any, error) {
	repoPath := manifest.Inputs["repo_path"]
	if _, err := tools.FSList(repoPath); err != nil {
		return nil, err
	}
	sourcePath := filepath.Join(repoPath, "mathlib.py")
	testPath := filepath.Join(repoPath, "test_mathlib.py")
	source, err := tools.FSRead(sourcePath)
	if err != nil {
		return nil, err
	}
	testContent, err := tools.FSRead(testPath)
	if err != nil {
		return nil, err
	}
	rootCause := "Unable to identify the root cause from the fixture."
	if strings.Contains(source, "return a - b") && strings.Contains(testContent, "assert add(2, 3) == 5") {
		rootCause = "mathlib.add returns a - b while the test expects addition."
	}
	return map[string]any{
		"analysis_summary":  "The pytest fixture fails because the add helper subtracts instead of adding.",
		"failing_tests":     []string{"test_mathlib.py::test_adds_numbers"},
		"likely_root_cause": rootCause,
		"changed_files":     []string{},
	}, nil
}

func executeApplySmallRefactor(manifest Manifest, tools *Tools) (map[string]any, error) {
	repoPath := manifest.Inputs["repo_path"]
	target := filepath.Join(repoPath, "greeter.py")
	content, err := tools.FSRead(target)
	if err != nil {
		return nil, err
	}
	refactored, changed := rewriteGreeterForRefactor(content)
	if changed {
		if err := tools.FSWrite(target, refactored, true); err != nil {
			return nil, err
		}
	}
	return map[string]any{
		"patch_summary":    "Extracted repeated name formatting into full_name.",
		"refactor_summary": "greeting now delegates name formatting to full_name without changing behavior.",
		"changed_files":    []string{target},
	}, nil
}

func rewriteGreeterForRefactor(content string) (string, bool) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	refactored := strings.Replace(normalized, `def greeting(first, last):
    return "Hello, " + first.strip() + " " + last.strip() + "!"
`, `def full_name(first, last):
    return first.strip() + " " + last.strip()


def greeting(first, last):
    return "Hello, " + full_name(first, last) + "!"
`, 1)
	if refactored == normalized {
		return content, false
	}
	if strings.Contains(content, "\r\n") {
		refactored = strings.ReplaceAll(refactored, "\n", "\r\n")
	}
	return refactored, true
}

func (EscapeAdapter) Metadata() AdapterMetadata {
	return FakeFixTestFailureAdapter{}.Metadata()
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
	verifyTrace.SetRedaction(manifest.Redaction)
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
	Events      []TraceEvent
	FinalOutput interface{}
	FinalError  string
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
		Events:      append([]TraceEvent(nil), trace.Events...),
		FinalOutput: trace.Final.Output,
		FinalError:  trace.Final.Error,
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
