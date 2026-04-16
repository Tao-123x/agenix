package agenix

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PublishOptions struct {
	ArtifactPath string
	RegistryRoot string
}

type PullOptions struct {
	Reference    string
	OutputPath   string
	RegistryRoot string
}

type RegistryEntry struct {
	Skill        string    `json:"skill"`
	Version      string    `json:"version"`
	Digest       string    `json:"digest"`
	ArtifactPath string    `json:"artifact_path"`
	PublishedAt  time.Time `json:"published_at"`
}

type RegistryIndex struct {
	Entries []RegistryEntry `json:"entries"`
}

func ResolveRegistryReference(reference, registry string) (string, error) {
	if pathExists(reference) {
		abs, err := filepath.Abs(reference)
		if err != nil {
			return "", WrapError(ErrInvalidInput, "normalize reference path", err)
		}
		return abs, nil
	}
	if !looksLikeRegistryReference(reference) {
		if registry != "" && !looksLikePathTarget(reference) {
			return "", NewError(ErrInvalidInput, "registry reference must be skill@version or sha256:digest")
		}
		return reference, nil
	}
	registryRoot, err := registryRoot(registry)
	if err != nil {
		return "", err
	}
	index, err := loadRegistryIndex(registryRoot)
	if err != nil {
		return "", err
	}
	entry, err := findRegistryEntry(index, reference)
	if err != nil {
		return "", err
	}
	return entry.ArtifactPath, nil
}

func PublishArtifact(options PublishOptions) (RegistryEntry, error) {
	if options.ArtifactPath == "" {
		return RegistryEntry{}, NewError(ErrInvalidInput, "publish requires artifact path")
	}
	artifactPath, err := filepath.Abs(options.ArtifactPath)
	if err != nil {
		return RegistryEntry{}, WrapError(ErrInvalidInput, "normalize artifact path", err)
	}
	summary, err := InspectArtifact(artifactPath)
	if err != nil {
		return RegistryEntry{}, err
	}
	registryRoot, err := registryRoot(options.RegistryRoot)
	if err != nil {
		return RegistryEntry{}, err
	}
	index, err := loadRegistryIndex(registryRoot)
	if err != nil {
		return RegistryEntry{}, err
	}
	for _, entry := range index.Entries {
		if entry.Skill == summary.Skill && entry.Version == summary.Version && entry.Digest != summary.Digest {
			return RegistryEntry{}, NewError(ErrInvalidInput, "registry already contains a different digest for "+summary.Skill+"@"+summary.Version)
		}
		if entry.Digest == summary.Digest {
			if _, statErr := os.Stat(entry.ArtifactPath); statErr == nil {
				return entry, nil
			}
		}
	}

	entry := RegistryEntry{
		Skill:        summary.Skill,
		Version:      summary.Version,
		Digest:       summary.Digest,
		ArtifactPath: filepath.Join(registryRoot, registryArtifactRelPath(summary.Skill, summary.Version, summary.Digest)),
		PublishedAt:  time.Now().UTC(),
	}
	if err := copyFile(artifactPath, entry.ArtifactPath); err != nil {
		return RegistryEntry{}, err
	}
	index.Entries = append(index.Entries, entry)
	sort.Slice(index.Entries, func(i, j int) bool {
		if index.Entries[i].Skill != index.Entries[j].Skill {
			return index.Entries[i].Skill < index.Entries[j].Skill
		}
		if index.Entries[i].Version != index.Entries[j].Version {
			return index.Entries[i].Version < index.Entries[j].Version
		}
		return index.Entries[i].Digest < index.Entries[j].Digest
	})
	if err := writeRegistryIndex(registryRoot, index); err != nil {
		return RegistryEntry{}, err
	}
	return entry, nil
}

func PullArtifact(options PullOptions) (ArtifactSummary, error) {
	if options.Reference == "" {
		return ArtifactSummary{}, NewError(ErrInvalidInput, "pull requires reference")
	}
	if options.OutputPath == "" {
		return ArtifactSummary{}, NewError(ErrInvalidInput, "pull requires output path")
	}
	registryRoot, err := registryRoot(options.RegistryRoot)
	if err != nil {
		return ArtifactSummary{}, err
	}
	index, err := loadRegistryIndex(registryRoot)
	if err != nil {
		return ArtifactSummary{}, err
	}
	entry, err := findRegistryEntry(index, options.Reference)
	if err != nil {
		return ArtifactSummary{}, err
	}
	summary, err := InspectArtifact(entry.ArtifactPath)
	if err != nil {
		return ArtifactSummary{}, err
	}
	outputPath, err := filepath.Abs(options.OutputPath)
	if err != nil {
		return ArtifactSummary{}, WrapError(ErrInvalidInput, "normalize output path", err)
	}
	if err := copyFile(entry.ArtifactPath, outputPath); err != nil {
		return ArtifactSummary{}, err
	}
	summary.Path = outputPath
	return summary, nil
}

func registryRoot(override string) (string, error) {
	root := override
	if root == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", WrapError(ErrDriverError, "resolve home directory", err)
		}
		root = filepath.Join(home, ".agenix", "registry")
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", WrapError(ErrInvalidInput, "normalize registry root", err)
	}
	return abs, nil
}

func loadRegistryIndex(root string) (RegistryIndex, error) {
	path := filepath.Join(root, "index.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return RegistryIndex{}, nil
		}
		return RegistryIndex{}, WrapError(ErrDriverError, "read registry index", err)
	}
	var index RegistryIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		return RegistryIndex{}, WrapError(ErrInvalidInput, "decode registry index", err)
	}
	return index, nil
}

func writeRegistryIndex(root string, index RegistryIndex) error {
	path := filepath.Join(root, "index.json")
	if err := ensureParent(path); err != nil {
		return WrapError(ErrDriverError, "create registry parent", err)
	}
	raw, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return WrapError(ErrDriverError, "encode registry index", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		return WrapError(ErrDriverError, "write registry index", err)
	}
	return nil
}

func findRegistryEntry(index RegistryIndex, reference string) (RegistryEntry, error) {
	if strings.HasPrefix(reference, "sha256:") {
		for _, entry := range index.Entries {
			if entry.Digest == reference {
				return entry, nil
			}
		}
		return RegistryEntry{}, NewError(ErrNotFound, "registry digest not found: "+reference)
	}
	skill, version, ok := strings.Cut(reference, "@")
	if !ok || skill == "" || version == "" {
		return RegistryEntry{}, NewError(ErrInvalidInput, "registry reference must be skill@version or sha256:digest")
	}
	for _, entry := range index.Entries {
		if entry.Skill == skill && entry.Version == version {
			return entry, nil
		}
	}
	return RegistryEntry{}, NewError(ErrNotFound, "registry artifact not found: "+reference)
}

func registryArtifactRelPath(skill, version, digest string) string {
	return filepath.Join("artifacts", filepath.FromSlash(skill), version, strings.ReplaceAll(digest, ":", "-")+".agenix")
}

func looksLikeRegistryReference(value string) bool {
	if strings.HasPrefix(value, "sha256:") {
		return true
	}
	if strings.ContainsAny(value, `/\`) {
		return false
	}
	skill, version, ok := strings.Cut(value, "@")
	return ok && skill != "" && version != ""
}

func looksLikePathTarget(value string) bool {
	return strings.ContainsAny(value, `/\`) || strings.HasSuffix(value, ".agenix") || strings.HasSuffix(value, ".yaml") || strings.HasSuffix(value, ".json")
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return WrapError(ErrNotFound, "open source file", err)
	}
	defer in.Close()
	if err := ensureParent(dst); err != nil {
		return WrapError(ErrDriverError, "create destination parent", err)
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return WrapError(ErrDriverError, "create destination file", err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return WrapError(ErrDriverError, "copy file", err)
	}
	if err := out.Close(); err != nil {
		return WrapError(ErrDriverError, "close destination file", err)
	}
	return nil
}
