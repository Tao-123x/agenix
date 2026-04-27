package agenix

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type OpenAIAnalyzeAdapter struct{}

func (OpenAIAnalyzeAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "openai-analyze",
		ModelProfile:    "openai:gpt-5.4-mini",
		Provider:        "openai",
		Transport:       "remote",
		SupportedSkills: []string{"repo.analyze_test_failures.remote"},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 32000,
			ReasoningLevel:   "medium",
		},
	}
}

func (OpenAIAnalyzeAdapter) Execute(manifest Manifest, tools *Tools) (map[string]any, error) {
	repoPath := manifest.Inputs["repo_path"]
	if _, err := tools.FSList(repoPath); err != nil {
		return nil, err
	}
	mathlibPath := filepath.Join(repoPath, "mathlib.py")
	testPath := filepath.Join(repoPath, "test_mathlib.py")
	mathlibContent, err := tools.FSRead(mathlibPath)
	if err != nil {
		return nil, err
	}
	testContent, err := tools.FSRead(testPath)
	if err != nil {
		return nil, err
	}

	var context strings.Builder
	context.WriteString("repo_path=")
	context.WriteString(repoPath)
	context.WriteString("\nmathlib.py:\n")
	context.WriteString(mathlibContent)
	context.WriteString("\ntest_mathlib.py:\n")
	context.WriteString(testContent)

	client := OpenAIAnalyzeClient{
		BaseURL:          os.Getenv("AGENIX_OPENAI_BASE_URL"),
		APIKey:           os.Getenv("OPENAI_API_KEY"),
		Model:            "gpt-5.4-mini",
		Timeout:          openAIAnalyzeTimeoutFromEnv(),
		MaxResponseBytes: openAIAnalyzeMaxResponseBytesFromEnv(),
	}
	result, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill:   manifest.Name,
		Context: context.String(),
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"analysis_summary":  result.AnalysisSummary,
		"failing_tests":     result.FailingTests,
		"likely_root_cause": result.LikelyRootCause,
		"changed_files":     result.ChangedFiles,
	}, nil
}

func openAIAnalyzeTimeoutFromEnv() time.Duration {
	value := strings.TrimSpace(os.Getenv("AGENIX_OPENAI_TIMEOUT_MS"))
	if value == "" {
		return 0
	}
	milliseconds, err := strconv.Atoi(value)
	if err != nil || milliseconds <= 0 {
		return 0
	}
	return time.Duration(milliseconds) * time.Millisecond
}

func openAIAnalyzeMaxResponseBytesFromEnv() int64 {
	value := strings.TrimSpace(os.Getenv("AGENIX_OPENAI_MAX_RESPONSE_BYTES"))
	if value == "" {
		return 0
	}
	bytes, err := strconv.ParseInt(value, 10, 64)
	if err != nil || bytes <= 0 {
		return 0
	}
	return bytes
}

type HeuristicAnalyzeTestFailuresAdapter struct{}

func (HeuristicAnalyzeTestFailuresAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "heuristic-analyze",
		ModelProfile:    "heuristic-analyze",
		Transport:       "local",
		SupportedSkills: []string{"repo.analyze_test_failures"},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 32000,
			ReasoningLevel:   "medium",
		},
	}
}

func (HeuristicAnalyzeTestFailuresAdapter) Execute(manifest Manifest, tools *Tools) (map[string]any, error) {
	return executeAnalyzeTestFailures(manifest, tools)
}

type PythonPytestTemplateAdapter struct{}

func (PythonPytestTemplateAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:         "python-pytest-template",
		ModelProfile: "python-pytest-template",
		Transport:    "local",
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 4000,
			ReasoningLevel:   "minimal",
		},
	}
}

func (PythonPytestTemplateAdapter) Execute(manifest Manifest, tools *Tools) (map[string]any, error) {
	repoPath := strings.TrimSpace(manifest.Inputs["repo_path"])
	if repoPath == "" {
		return nil, NewError(ErrInvalidInput, "python-pytest-template requires input repo_path")
	}
	if _, err := tools.FSList(repoPath); err != nil {
		return nil, err
	}
	return map[string]any{
		"analysis_summary": "Generated python-pytest skill executed without code changes.",
		"changed_files":    []string{},
	}, nil
}

type RepoFixTestFailureTemplateAdapter struct{}

func (RepoFixTestFailureTemplateAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:         "repo-fix-test-failure-template",
		ModelProfile: "repo-fix-test-failure-template",
		Transport:    "local",
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 4000,
			ReasoningLevel:   "minimal",
		},
	}
}

func (RepoFixTestFailureTemplateAdapter) Execute(manifest Manifest, tools *Tools) (map[string]any, error) {
	return executeFixTestFailure(manifest, tools)
}

func ResolveBuiltinAdapter(name string) (Adapter, error) {
	switch name {
	case "", "fake-scripted":
		return FakeFixTestFailureAdapter{}, nil
	case "heuristic-analyze":
		return HeuristicAnalyzeTestFailuresAdapter{}, nil
	case "openai-analyze":
		return OpenAIAnalyzeAdapter{}, nil
	case "python-pytest-template":
		return PythonPytestTemplateAdapter{}, nil
	case "repo-fix-test-failure-template":
		return RepoFixTestFailureTemplateAdapter{}, nil
	default:
		return nil, NewError(ErrUnsupportedAdapter, "unknown adapter: "+name)
	}
}
