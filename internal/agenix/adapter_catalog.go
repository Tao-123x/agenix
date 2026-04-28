package agenix

import (
	"os"
	"path/filepath"
	"strings"
)

const AdapterCompatibilityReportKind = "adapter_compatibility_report"

type AdapterCompatibilityOptions struct {
	Target       string
	RegistryRoot string
	WorkDir      string
}

type AdapterCompatibilityReport struct {
	Kind     string                 `json:"kind"`
	Target   string                 `json:"target"`
	Skill    string                 `json:"skill"`
	Version  string                 `json:"version"`
	Adapters []AdapterCompatibility `json:"adapters"`
}

type AdapterCompatibility struct {
	Name            string        `json:"name"`
	ModelProfile    string        `json:"model_profile"`
	Provider        string        `json:"provider,omitempty"`
	Transport       string        `json:"transport"`
	SupportedSkills []string      `json:"supported_skills,omitempty"`
	Capabilities    CapabilitySet `json:"capabilities"`
	Compatible      bool          `json:"compatible"`
	ErrorClass      string        `json:"error_class,omitempty"`
	ErrorMessage    string        `json:"error_message,omitempty"`
}

func ListBuiltinAdapters() []AdapterMetadata {
	adapters := builtinAdapters()
	out := make([]AdapterMetadata, 0, len(adapters))
	for _, adapter := range adapters {
		out = append(out, normalizeAdapterMetadata(adapter.Metadata()))
	}
	return out
}

func CheckBuiltinAdapterCompatibility(options AdapterCompatibilityOptions) (AdapterCompatibilityReport, error) {
	if strings.TrimSpace(options.Target) == "" {
		return AdapterCompatibilityReport{}, NewError(ErrInvalidInput, "adapter compatibility requires target")
	}
	manifest, cleanup, err := loadManifestForAdapterCompatibility(options)
	defer cleanup()
	if err != nil {
		return AdapterCompatibilityReport{}, err
	}
	report := AdapterCompatibilityReport{
		Kind:    AdapterCompatibilityReportKind,
		Target:  options.Target,
		Skill:   manifest.Name,
		Version: manifest.Version,
	}
	for _, adapter := range builtinAdapters() {
		metadata := normalizeAdapterMetadata(adapter.Metadata())
		result := AdapterCompatibility{
			Name:            metadata.Name,
			ModelProfile:    metadata.ModelProfile,
			Provider:        metadata.Provider,
			Transport:       metadata.Transport,
			SupportedSkills: append([]string(nil), metadata.SupportedSkills...),
			Capabilities:    metadata.Capabilities,
			Compatible:      true,
		}
		if err := validateAdapter(manifest, metadata); err != nil {
			result.Compatible = false
			result.ErrorClass = ErrorClass(err)
			result.ErrorMessage = err.Error()
		} else if err := validateAdapterPolicy(manifest, metadata); err != nil {
			result.Compatible = false
			result.ErrorClass = ErrorClass(err)
			result.ErrorMessage = err.Error()
		}
		report.Adapters = append(report.Adapters, result)
	}
	return report, nil
}

func builtinAdapters() []Adapter {
	return []Adapter{
		FakeFixTestFailureAdapter{},
		HeuristicAnalyzeTestFailuresAdapter{},
		OpenAIAnalyzeAdapter{},
		PythonPytestTemplateAdapter{},
		RepoFixTestFailureTemplateAdapter{},
	}
}

func normalizeAdapterMetadata(metadata AdapterMetadata) AdapterMetadata {
	if metadata.ModelProfile == "" {
		metadata.ModelProfile = fakeModelProfile
	}
	if metadata.Name == "" {
		metadata.Name = metadata.ModelProfile
	}
	metadata.Transport = normalizeTransport(metadata.Transport)
	metadata.SupportedSkills = append([]string(nil), metadata.SupportedSkills...)
	return metadata
}

func loadManifestForAdapterCompatibility(options AdapterCompatibilityOptions) (Manifest, func(), error) {
	cleanup := func() {}
	target, err := ResolveRegistryReference(options.Target, options.RegistryRoot)
	if err != nil {
		return Manifest{}, cleanup, err
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return Manifest{}, cleanup, WrapError(ErrInvalidInput, "normalize adapter compatibility target", err)
	}
	absTarget = filepath.Clean(absTarget)
	info, err := os.Stat(absTarget)
	if err != nil {
		return Manifest{}, cleanup, WrapError(ErrNotFound, "stat adapter compatibility target", err)
	}
	if !info.IsDir() && isArtifactTarget(absTarget) {
		workspaceDir, remove, err := adapterCompatibilityWorkspace(options.WorkDir)
		if err != nil {
			return Manifest{}, cleanup, err
		}
		manifestPath, _, err := MaterializeArtifact(absTarget, workspaceDir)
		if err != nil {
			remove()
			return Manifest{}, cleanup, err
		}
		manifest, err := LoadManifest(manifestPath)
		if err != nil {
			remove()
			return Manifest{}, cleanup, err
		}
		return manifest, remove, nil
	}
	manifestPath := absTarget
	if info.IsDir() {
		manifestPath = filepath.Join(absTarget, "manifest.yaml")
	}
	manifest, err := LoadManifest(manifestPath)
	return manifest, cleanup, err
}

func adapterCompatibilityWorkspace(workDir string) (string, func(), error) {
	if strings.TrimSpace(workDir) != "" {
		workspace := filepath.Join(workDir, newRunID(), "workspace")
		if err := os.MkdirAll(workspace, 0o755); err != nil {
			return "", func() {}, WrapError(ErrDriverError, "create adapter compatibility workspace", err)
		}
		return workspace, func() {}, nil
	}
	dir, err := os.MkdirTemp("", "agenix-adapter-compat-*")
	if err != nil {
		return "", func() {}, WrapError(ErrDriverError, "create adapter compatibility workspace", err)
	}
	return dir, func() { _ = os.RemoveAll(dir) }, nil
}
