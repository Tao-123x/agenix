package agenix

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type AcceptanceOptions struct {
	RootDir       string
	WorkDir       string
	ProviderSmoke bool
}

type AcceptanceSummary struct {
	Status                   string
	SkillCount               int
	RunCount                 int
	TemplateCount            int
	CheckCount               int
	FailureReportCount       int
	AdapterCount             int
	CompatibilityReportCount int
	SchemaCount              int
	ProviderSmokeStatus      string
	ProviderSmokeTracePath   string
}

type acceptanceSkill struct {
	name          string
	dirName       string
	expectedSkill string
	adapter       Adapter
	changedBase   string
	readOnly      bool
}

type authoringAcceptanceSkill struct {
	name          string
	template      string
	adapterName   string
	changedBase   string
	readOnly      bool
	expectedSkill string
}

type v03CompatibilityTarget struct {
	name         string
	target       string
	registryRoot string
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

func RunV02AcceptanceSweep(options AcceptanceOptions) (AcceptanceSummary, error) {
	if _, err := acceptanceRootDir(options.RootDir); err != nil {
		return AcceptanceSummary{Status: "failed"}, err
	}
	workDir, cleanup, err := acceptanceWorkDir(options.WorkDir)
	if err != nil {
		return AcceptanceSummary{Status: "failed"}, err
	}
	defer cleanup()

	templates := ListSkillTemplates()
	if err := validateAuthoringTemplates(templates); err != nil {
		return AcceptanceSummary{Status: "failed", TemplateCount: len(templates)}, err
	}

	skills := []authoringAcceptanceSkill{
		{
			name:          "python-pytest",
			template:      PythonPytestTemplate,
			adapterName:   "python-pytest-template",
			expectedSkill: "repo.demo_skill",
			readOnly:      true,
		},
		{
			name:          "repo-fix-test-failure",
			template:      RepoFixTestFailureTemplate,
			adapterName:   "repo-fix-test-failure-template",
			expectedSkill: "repo.demo_fix",
			changedBase:   "mathlib.py",
		},
	}

	summary := AcceptanceSummary{Status: "passed", TemplateCount: len(templates)}
	for _, skill := range skills {
		checks, err := runAuthoringAcceptanceSkill(workDir, skill)
		summary.CheckCount += checks
		if err != nil {
			summary.Status = "failed"
			return summary, err
		}
		summary.SkillCount++
		summary.RunCount += 2
	}
	checks, err := runAuthoringFailureReportAcceptance(workDir)
	summary.CheckCount += checks
	if err != nil {
		summary.Status = "failed"
		return summary, err
	}
	summary.FailureReportCount = 1
	summary.RunCount++
	return summary, nil
}

func RunV03AcceptanceSweep(options AcceptanceOptions) (AcceptanceSummary, error) {
	rootDir, err := acceptanceRootDir(options.RootDir)
	if err != nil {
		return AcceptanceSummary{Status: "failed"}, err
	}
	workDir, cleanup, err := acceptanceWorkDir(options.WorkDir)
	if err != nil {
		return AcceptanceSummary{Status: "failed"}, err
	}
	defer cleanup()

	adapters := ListBuiltinAdapters()
	summary := AcceptanceSummary{
		Status:              "passed",
		AdapterCount:        len(adapters),
		ProviderSmokeStatus: "skipped_offline",
	}
	if err := validateV03AdapterCatalog(adapters); err != nil {
		summary.Status = "failed"
		return summary, err
	}

	remoteSkillDir := filepath.Join(rootDir, "examples", "repo.analyze_test_failures.remote")
	remoteManifestPath := filepath.Join(remoteSkillDir, "manifest.yaml")
	v03WorkDir := filepath.Join(workDir, "v0.3", "adapter-readiness")
	reportsDir := filepath.Join(v03WorkDir, "reports")
	registryRoot := filepath.Join(v03WorkDir, "registry")
	artifactPath := filepath.Join(v03WorkDir, "repo.analyze_test_failures.remote.agenix")

	targets := []v03CompatibilityTarget{
		{name: "manifest", target: remoteManifestPath},
	}

	if _, err := BuildArtifact(BuildOptions{SkillDir: remoteSkillDir, OutputPath: artifactPath}); err != nil {
		summary.Status = "failed"
		return summary, WrapError(ErrorClass(err), "build v0.3 remote analysis artifact", err)
	}
	targets = append(targets, v03CompatibilityTarget{name: "artifact", target: artifactPath})

	if _, err := PublishArtifact(PublishOptions{ArtifactPath: artifactPath, RegistryRoot: registryRoot}); err != nil {
		summary.Status = "failed"
		return summary, WrapError(ErrorClass(err), "publish v0.3 remote analysis artifact", err)
	}
	targets = append(targets, v03CompatibilityTarget{name: "registry", target: "repo.analyze_test_failures.remote@0.1.0", registryRoot: registryRoot})

	for _, target := range targets {
		report, err := CheckBuiltinAdapterCompatibility(AdapterCompatibilityOptions{
			Target:       target.target,
			RegistryRoot: target.registryRoot,
			WorkDir:      filepath.Join(v03WorkDir, "compat-workspaces"),
		})
		if err != nil {
			summary.Status = "failed"
			return summary, WrapError(ErrorClass(err), "check v0.3 adapter compatibility for "+target.name, err)
		}
		summary.CompatibilityReportCount++
		if err := validateV03CompatibilityReport(report, len(adapters)); err != nil {
			summary.Status = "failed"
			return summary, err
		}
		reportPath := filepath.Join(reportsDir, target.name+"-adapter-compatibility.json")
		if err := writeAndValidateAdapterCompatibilityReport(report, reportPath); err != nil {
			summary.Status = "failed"
			return summary, err
		}
		summary.SchemaCount++
	}
	if options.ProviderSmoke {
		status, tracePath, err := runV03ProviderSmoke(rootDir, filepath.Join(v03WorkDir, "provider-smoke"))
		summary.ProviderSmokeStatus = status
		summary.ProviderSmokeTracePath = tracePath
		if err != nil {
			summary.Status = "failed"
			return summary, err
		}
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

func validateAuthoringTemplates(templates []SkillTemplateDescriptor) error {
	if len(templates) != 2 {
		return NewError(ErrVerificationFailed, fmt.Sprintf("expected 2 authoring templates, got %d", len(templates)))
	}
	seen := map[string]SkillTemplateDescriptor{}
	for _, template := range templates {
		if template.Name == "" || template.Adapter == "" || template.Description == "" {
			return NewError(ErrVerificationFailed, "authoring template descriptor is incomplete")
		}
		seen[template.Name] = template
	}
	if template, ok := seen[PythonPytestTemplate]; !ok || template.Adapter != "python-pytest-template" || template.Writes {
		return NewError(ErrVerificationFailed, "python-pytest authoring template descriptor is invalid")
	}
	if template, ok := seen[RepoFixTestFailureTemplate]; !ok || template.Adapter != "repo-fix-test-failure-template" || !template.Writes {
		return NewError(ErrVerificationFailed, "repo-fix-test-failure authoring template descriptor is invalid")
	}
	return nil
}

func runAuthoringAcceptanceSkill(workDir string, skill authoringAcceptanceSkill) (int, error) {
	skillWorkDir := filepath.Join(workDir, "v0.2", skill.name)
	skillDir := filepath.Join(skillWorkDir, skill.expectedSkill)
	artifactPath := filepath.Join(skillWorkDir, skill.name+".agenix")
	reportPath := filepath.Join(skillWorkDir, "check-report.json")

	initResult, err := InitSkill(InitSkillOptions{Name: skill.expectedSkill, Template: skill.template, OutputDir: skillDir})
	if err != nil {
		return 0, WrapError(ErrorClass(err), "init skill for "+skill.expectedSkill, err)
	}
	if initResult.Name != skill.expectedSkill || initResult.Template != skill.template {
		return 0, NewError(ErrVerificationFailed, "init skill returned unexpected identity for "+skill.expectedSkill)
	}
	if err := validateGeneratedAuthoringManifest(skillDir, skill.expectedSkill); err != nil {
		return 0, err
	}

	buildSummary, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath})
	if err != nil {
		return 0, WrapError(ErrorClass(err), "build generated authoring artifact for "+skill.expectedSkill, err)
	}
	if buildSummary.Skill != skill.expectedSkill {
		return 0, NewError(ErrVerificationFailed, fmt.Sprintf("build generated authoring artifact returned skill %q", buildSummary.Skill))
	}
	if inspectSummary, err := InspectArtifact(artifactPath); err != nil {
		return 0, WrapError(ErrorClass(err), "inspect generated authoring artifact for "+skill.expectedSkill, err)
	} else if inspectSummary.Skill != skill.expectedSkill {
		return 0, NewError(ErrVerificationFailed, fmt.Sprintf("inspect generated authoring artifact returned skill %q", inspectSummary.Skill))
	}

	adapter, err := ResolveBuiltinAdapter(skill.adapterName)
	if err != nil {
		return 0, err
	}
	runResult, err := Run(RunOptions{
		ManifestPath: artifactPath,
		RunDir:       filepath.Join(skillWorkDir, "runs"),
		Adapter:      adapter,
	})
	if err != nil {
		return 0, WrapError(ErrorClass(err), "run generated authoring artifact for "+skill.expectedSkill, err)
	}
	if err := validateAuthoringRunResult(skill, runResult); err != nil {
		return 0, err
	}
	if kind, _, err := ValidateTarget(runResult.TracePath); err != nil {
		return 0, WrapError(ErrorClass(err), "validate generated authoring trace for "+skill.expectedSkill, err)
	} else if kind != "trace" {
		return 0, NewError(ErrVerificationFailed, fmt.Sprintf("validate generated authoring trace returned kind %q", kind))
	}

	checkResult, err := CheckSkill(CheckOptions{
		Target:  skillDir,
		WorkDir: filepath.Join(skillWorkDir, "checks"),
		Adapter: adapter,
	})
	if err != nil {
		return 1, WrapError(ErrorClass(err), "check generated authoring skill for "+skill.expectedSkill, err)
	}
	if err := validateAuthoringCheckResult(skill, checkResult, "passed"); err != nil {
		return 1, err
	}
	if err := writeAndValidateCheckReport(checkResult, reportPath); err != nil {
		return 1, err
	}
	return 1, nil
}

func validateGeneratedAuthoringManifest(skillDir, expectedSkill string) error {
	manifestPath := filepath.Join(skillDir, "manifest.yaml")
	if kind, _, err := ValidateTarget(manifestPath); err != nil {
		return WrapError(ErrorClass(err), "validate generated authoring manifest for "+expectedSkill, err)
	} else if kind != "manifest" {
		return NewError(ErrVerificationFailed, fmt.Sprintf("validate generated authoring manifest returned kind %q", kind))
	}
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return err
	}
	if manifest.Name != expectedSkill {
		return NewError(ErrVerificationFailed, fmt.Sprintf("generated authoring manifest name = %q", manifest.Name))
	}
	return nil
}

func validateAuthoringRunResult(skill authoringAcceptanceSkill, result RunResult) error {
	if result.Status != "passed" {
		return NewError(ErrVerificationFailed, fmt.Sprintf("generated authoring run status = %q", result.Status))
	}
	if result.TracePath == "" {
		return NewError(ErrVerificationFailed, "generated authoring run did not write trace for "+skill.expectedSkill)
	}
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

func validateAuthoringCheckResult(skill authoringAcceptanceSkill, result CheckResult, expectedStatus string) error {
	if result.Kind != CheckReportKind {
		return NewError(ErrVerificationFailed, fmt.Sprintf("check report kind = %q", result.Kind))
	}
	if result.Status != expectedStatus {
		return NewError(ErrVerificationFailed, fmt.Sprintf("check report status = %q", result.Status))
	}
	if result.Skill != skill.expectedSkill {
		return NewError(ErrVerificationFailed, fmt.Sprintf("check report skill = %q", result.Skill))
	}
	if result.ArtifactPath == "" || result.RunID == "" || result.TracePath == "" {
		return NewError(ErrVerificationFailed, "check report is missing artifact, run id, or trace path")
	}
	if result.EventCount == 0 {
		return NewError(ErrVerificationFailed, "check report is missing event count")
	}
	if expectedStatus == "failed" {
		if result.ErrorClass == "" || result.ErrorMessage == "" {
			return NewError(ErrVerificationFailed, "failed check report is missing error fields")
		}
		return nil
	}
	if result.ErrorClass != "" || result.ErrorMessage != "" {
		return NewError(ErrVerificationFailed, "passed check report unexpectedly contains error fields")
	}
	return validateAuthoringRunResult(skill, RunResult{
		Status:       result.Status,
		TracePath:    result.TracePath,
		ChangedFiles: result.ChangedFiles,
	})
}

func runAuthoringFailureReportAcceptance(workDir string) (int, error) {
	skill := authoringAcceptanceSkill{
		name:          "failure-report",
		template:      PythonPytestTemplate,
		adapterName:   "python-pytest-template",
		expectedSkill: "repo.demo_broken",
		readOnly:      true,
	}
	skillWorkDir := filepath.Join(workDir, "v0.2", skill.name)
	skillDir := filepath.Join(skillWorkDir, skill.expectedSkill)
	reportPath := filepath.Join(skillWorkDir, "failed-check-report.json")
	if _, err := InitSkill(InitSkillOptions{Name: skill.expectedSkill, Template: skill.template, OutputDir: skillDir}); err != nil {
		return 0, WrapError(ErrorClass(err), "init failed authoring skill", err)
	}
	brokenSource := []byte("def normalize(value):\n    return value\n")
	if err := os.WriteFile(filepath.Join(skillDir, "fixture", "skill.py"), brokenSource, 0o600); err != nil {
		return 0, WrapError(ErrDriverError, "break generated authoring fixture", err)
	}
	adapter, err := ResolveBuiltinAdapter(skill.adapterName)
	if err != nil {
		return 0, err
	}
	checkResult, err := CheckSkill(CheckOptions{
		Target:  skillDir,
		WorkDir: filepath.Join(skillWorkDir, "checks"),
		Adapter: adapter,
	})
	if err == nil {
		return 1, NewError(ErrVerificationFailed, "broken authoring skill unexpectedly passed check")
	}
	report := checkResult.WithError(err)
	if report.ErrorClass != ErrVerificationFailed {
		return 1, NewError(ErrVerificationFailed, fmt.Sprintf("failed check report error_class = %q", report.ErrorClass))
	}
	if err := validateAuthoringCheckResult(skill, report, "failed"); err != nil {
		return 1, err
	}
	if err := writeAndValidateCheckReport(report, reportPath); err != nil {
		return 1, err
	}
	return 1, nil
}

func writeAndValidateCheckReport(result CheckResult, path string) error {
	result = result.ensureArrays()
	if err := ensureParent(path); err != nil {
		return WrapError(ErrDriverError, "create check report parent", err)
	}
	raw, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return WrapError(ErrDriverError, "encode check report", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return WrapError(ErrDriverError, "write check report", err)
	}
	if kind, _, err := ValidateTarget(path); err != nil {
		return WrapError(ErrorClass(err), "validate check report", err)
	} else if kind != "check_report" {
		return NewError(ErrVerificationFailed, fmt.Sprintf("validate check report returned kind %q", kind))
	}
	return nil
}

func runV03ProviderSmoke(rootDir, workDir string) (string, string, error) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		return "skipped_no_credentials", "", nil
	}
	result, err := Run(RunOptions{
		ManifestPath: filepath.Join(rootDir, "examples", "repo.analyze_test_failures.remote", "manifest.yaml"),
		RunDir:       workDir,
		Adapter:      OpenAIAnalyzeAdapter{},
	})
	if err != nil {
		return "failed", result.TracePath, WrapError(ErrorClass(err), "run v0.3 provider smoke", err)
	}
	if result.Status != "passed" {
		return "failed", result.TracePath, NewError(ErrVerificationFailed, fmt.Sprintf("v0.3 provider smoke status = %q", result.Status))
	}
	if result.TracePath == "" {
		return "failed", "", NewError(ErrDriverError, "v0.3 provider smoke did not write trace")
	}
	if kind, _, err := ValidateTarget(result.TracePath); err != nil {
		return "failed", result.TracePath, WrapError(ErrorClass(err), "validate v0.3 provider smoke trace", err)
	} else if kind != "trace" {
		return "failed", result.TracePath, NewError(ErrVerificationFailed, fmt.Sprintf("validate v0.3 provider smoke trace returned kind %q", kind))
	}
	return "passed", result.TracePath, nil
}

func validateV03AdapterCatalog(adapters []AdapterMetadata) error {
	if len(adapters) != 5 {
		return NewError(ErrVerificationFailed, fmt.Sprintf("expected 5 builtin adapters, got %d", len(adapters)))
	}
	openai, ok := adapterMetadataByName(adapters, "openai-analyze")
	if !ok {
		return NewError(ErrVerificationFailed, "v0.3 adapter catalog is missing openai-analyze")
	}
	if openai.Provider != "openai" || openai.Transport != "remote" {
		return NewError(ErrVerificationFailed, "openai-analyze adapter metadata is missing provider or remote transport")
	}
	if !openai.Capabilities.ToolCalling || !openai.Capabilities.StructuredOutput {
		return NewError(ErrVerificationFailed, "openai-analyze adapter metadata is missing required capabilities")
	}
	heuristic, ok := adapterMetadataByName(adapters, "heuristic-analyze")
	if !ok {
		return NewError(ErrVerificationFailed, "v0.3 adapter catalog is missing heuristic-analyze")
	}
	if heuristic.Transport != "local" || heuristic.Provider != "" {
		return NewError(ErrVerificationFailed, "heuristic-analyze adapter metadata should stay local")
	}
	return nil
}

func validateV03CompatibilityReport(report AdapterCompatibilityReport, adapterCount int) error {
	if report.Kind != AdapterCompatibilityReportKind {
		return NewError(ErrVerificationFailed, fmt.Sprintf("v0.3 compatibility report kind = %q", report.Kind))
	}
	if report.Skill != "repo.analyze_test_failures.remote" {
		return NewError(ErrVerificationFailed, fmt.Sprintf("v0.3 compatibility report skill = %q", report.Skill))
	}
	if report.Version != "0.1.0" {
		return NewError(ErrVerificationFailed, fmt.Sprintf("v0.3 compatibility report version = %q", report.Version))
	}
	if len(report.Adapters) != adapterCount {
		return NewError(ErrVerificationFailed, fmt.Sprintf("v0.3 compatibility adapter count = %d", len(report.Adapters)))
	}
	openai, ok := adapterCompatibilityByName(report.Adapters, "openai-analyze")
	if !ok {
		return NewError(ErrVerificationFailed, "v0.3 compatibility report is missing openai-analyze")
	}
	if !openai.Compatible || openai.ErrorClass != "" || openai.Transport != "remote" || openai.Provider != "openai" {
		return NewError(ErrVerificationFailed, "openai-analyze should pass remote compatibility preflight")
	}
	fake, ok := adapterCompatibilityByName(report.Adapters, "fake-scripted")
	if !ok {
		return NewError(ErrVerificationFailed, "v0.3 compatibility report is missing fake-scripted")
	}
	if fake.Compatible || fake.ErrorClass != ErrUnsupportedAdapter {
		return NewError(ErrVerificationFailed, "fake-scripted should reject unsupported remote analysis skill")
	}
	return nil
}

func writeAndValidateAdapterCompatibilityReport(report AdapterCompatibilityReport, path string) error {
	if err := ensureParent(path); err != nil {
		return WrapError(ErrDriverError, "create adapter compatibility report parent", err)
	}
	raw, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return WrapError(ErrDriverError, "encode adapter compatibility report", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		return WrapError(ErrDriverError, "write adapter compatibility report", err)
	}
	if kind, _, err := ValidateTarget(path); err != nil {
		return WrapError(ErrorClass(err), "validate adapter compatibility report", err)
	} else if kind != "adapter_compatibility_report" {
		return NewError(ErrVerificationFailed, fmt.Sprintf("validate adapter compatibility report returned kind %q", kind))
	}
	return nil
}

func adapterMetadataByName(adapters []AdapterMetadata, name string) (AdapterMetadata, bool) {
	for _, adapter := range adapters {
		if adapter.Name == name {
			return adapter, true
		}
	}
	return AdapterMetadata{}, false
}

func adapterCompatibilityByName(adapters []AdapterCompatibility, name string) (AdapterCompatibility, bool) {
	for _, adapter := range adapters {
		if adapter.Name == name {
			return adapter, true
		}
	}
	return AdapterCompatibility{}, false
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
