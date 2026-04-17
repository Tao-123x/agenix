package agenix

type HeuristicAnalyzeTestFailuresAdapter struct{}

func (HeuristicAnalyzeTestFailuresAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "heuristic-analyze",
		ModelProfile:    "heuristic-analyze",
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
	default:
		return nil, NewError(ErrUnsupportedAdapter, "unknown adapter: "+name)
	}
}
