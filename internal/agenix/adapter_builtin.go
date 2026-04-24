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
		BaseURL: os.Getenv("AGENIX_OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-5.4-mini",
		Timeout: openAIAnalyzeTimeoutFromEnv(),
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

func ResolveBuiltinAdapter(name string) (Adapter, error) {
	switch name {
	case "", "fake-scripted":
		return FakeFixTestFailureAdapter{}, nil
	case "heuristic-analyze":
		return HeuristicAnalyzeTestFailuresAdapter{}, nil
	case "openai-analyze":
		return OpenAIAnalyzeAdapter{}, nil
	default:
		return nil, NewError(ErrUnsupportedAdapter, "unknown adapter: "+name)
	}
}
