package agenix

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const ArtifactVersion = "agenix.artifact/v0.1"

type BuildOptions struct {
	SkillDir   string
	OutputPath string
}

type ArtifactSummary struct {
	Skill     string `json:"skill"`
	Version   string `json:"version"`
	Digest    string `json:"digest"`
	FileCount int    `json:"file_count"`
	Path      string `json:"path"`
}

type ArtifactLock struct {
	ArtifactVersion string             `json:"artifact_version"`
	CreatedAt       time.Time          `json:"created_at"`
	Skill           ArtifactSkill      `json:"skill"`
	ManifestDigest  string             `json:"manifest_digest"`
	ArtifactDigest  string             `json:"artifact_digest"`
	Files           []ArtifactFileLock `json:"files"`
}

type ArtifactSkill struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ArtifactFileLock struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

func BuildArtifact(options BuildOptions) (ArtifactSummary, error) {
	skillDir, err := filepath.Abs(options.SkillDir)
	if err != nil {
		return ArtifactSummary{}, WrapError(ErrInvalidInput, "normalize skill directory", err)
	}
	manifestPath := filepath.Join(skillDir, "manifest.yaml")
	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		return ArtifactSummary{}, err
	}
	outputPath := options.OutputPath
	if outputPath == "" {
		outputPath = filepath.Join(skillDir, manifest.Name+"-"+manifest.Version+".agenix")
	}
	outputPath, err = filepath.Abs(outputPath)
	if err != nil {
		return ArtifactSummary{}, WrapError(ErrInvalidInput, "normalize output path", err)
	}

	files, err := collectArtifactFiles(skillDir)
	if err != nil {
		return ArtifactSummary{}, err
	}
	manifestDigest, err := fileDigest(manifestPath)
	if err != nil {
		return ArtifactSummary{}, err
	}
	lock := ArtifactLock{
		ArtifactVersion: ArtifactVersion,
		CreatedAt:       time.Now().UTC(),
		Skill:           ArtifactSkill{Name: manifest.Name, Version: manifest.Version},
		ManifestDigest:  manifestDigest,
		Files: append([]ArtifactFileLock{{
			Path:   "manifest.yaml",
			Digest: manifestDigest,
			Size:   fileSize(manifestPath),
		}}, files...),
	}
	content, digest, err := renderArtifact(skillDir, manifestPath, lock)
	if err != nil {
		return ArtifactSummary{}, err
	}
	lock.ArtifactDigest = digest
	content, digest, err = renderArtifact(skillDir, manifestPath, lock)
	if err != nil {
		return ArtifactSummary{}, err
	}
	if err := ensureParent(outputPath); err != nil {
		return ArtifactSummary{}, WrapError(ErrDriverError, "create artifact parent", err)
	}
	if err := os.WriteFile(outputPath, content, 0o600); err != nil {
		return ArtifactSummary{}, WrapError(ErrDriverError, "write artifact", err)
	}
	return ArtifactSummary{Skill: manifest.Name, Version: manifest.Version, Digest: digest, FileCount: len(lock.Files), Path: outputPath}, nil
}

func InspectArtifact(path string) (ArtifactSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return ArtifactSummary{}, WrapError(ErrNotFound, "open artifact", err)
	}
	defer file.Close()
	hasher := sha256.New()
	raw, err := io.ReadAll(io.TeeReader(file, hasher))
	if err != nil {
		return ArtifactSummary{}, WrapError(ErrDriverError, "read artifact", err)
	}
	lock, err := readLockFromArtifact(bytes.NewReader(raw))
	if err != nil {
		return ArtifactSummary{}, err
	}
	abs, _ := filepath.Abs(path)
	return ArtifactSummary{
		Skill:     lock.Skill.Name,
		Version:   lock.Skill.Version,
		Digest:    "sha256:" + hex.EncodeToString(hasher.Sum(nil)),
		FileCount: len(lock.Files),
		Path:      abs,
	}, nil
}

func collectArtifactFiles(skillDir string) ([]ArtifactFileLock, error) {
	var files []ArtifactFileLock
	err := filepath.WalkDir(skillDir, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if shouldIgnoreArtifactPath(rel, entry.IsDir()) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.IsDir() || rel == "manifest.yaml" {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		digest, err := fileDigest(path)
		if err != nil {
			return err
		}
		files = append(files, ArtifactFileLock{Path: "files/" + rel, Digest: digest, Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, WrapError(ErrDriverError, "collect artifact files", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func renderArtifact(skillDir, manifestPath string, lock ArtifactLock) ([]byte, string, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Name = "agenix-artifact"
	gz.ModTime = time.Unix(0, 0).UTC()
	tw := tar.NewWriter(gz)

	if err := addFileToTar(tw, manifestPath, "manifest.yaml"); err != nil {
		return nil, "", err
	}
	for _, file := range lock.Files {
		if !strings.HasPrefix(file.Path, "files/") {
			continue
		}
		sourceRel := strings.TrimPrefix(file.Path, "files/")
		if err := addFileToTar(tw, filepath.Join(skillDir, filepath.FromSlash(sourceRel)), file.Path); err != nil {
			return nil, "", err
		}
	}
	lockJSON, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return nil, "", WrapError(ErrDriverError, "encode lockfile", err)
	}
	if err := addBytesToTar(tw, "agenix.lock.json", append(lockJSON, '\n'), 0o600); err != nil {
		return nil, "", err
	}
	if err := tw.Close(); err != nil {
		return nil, "", WrapError(ErrDriverError, "close tar", err)
	}
	if err := gz.Close(); err != nil {
		return nil, "", WrapError(ErrDriverError, "close gzip", err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), "sha256:" + hex.EncodeToString(sum[:]), nil
}

func addFileToTar(tw *tar.Writer, sourcePath, artifactPath string) error {
	raw, err := os.ReadFile(sourcePath)
	if err != nil {
		return WrapError(ErrDriverError, "read artifact source", err)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return WrapError(ErrDriverError, "stat artifact source", err)
	}
	return addBytesToTar(tw, artifactPath, raw, info.Mode().Perm())
}

func addBytesToTar(tw *tar.Writer, name string, raw []byte, mode os.FileMode) error {
	header := &tar.Header{
		Name:    name,
		Mode:    int64(mode),
		Size:    int64(len(raw)),
		ModTime: time.Unix(0, 0).UTC(),
	}
	if err := tw.WriteHeader(header); err != nil {
		return WrapError(ErrDriverError, "write tar header", err)
	}
	if _, err := tw.Write(raw); err != nil {
		return WrapError(ErrDriverError, "write tar body", err)
	}
	return nil
}

func readLockFromArtifact(reader io.Reader) (ArtifactLock, error) {
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return ArtifactLock{}, WrapError(ErrInvalidInput, "open gzip artifact", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return ArtifactLock{}, WrapError(ErrInvalidInput, "read tar artifact", err)
		}
		if header.Name != "agenix.lock.json" {
			continue
		}
		raw, err := io.ReadAll(tr)
		if err != nil {
			return ArtifactLock{}, WrapError(ErrDriverError, "read lockfile", err)
		}
		var lock ArtifactLock
		if err := json.Unmarshal(raw, &lock); err != nil {
			return ArtifactLock{}, WrapError(ErrInvalidInput, "decode lockfile", err)
		}
		return lock, nil
	}
	return ArtifactLock{}, NewError(ErrInvalidInput, "artifact missing agenix.lock.json")
}

func fileDigest(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", WrapError(ErrDriverError, "open digest source", err)
	}
	defer file.Close()
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", WrapError(ErrDriverError, "hash digest source", err)
	}
	return "sha256:" + hex.EncodeToString(hasher.Sum(nil)), nil
}

func fileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func shouldIgnoreArtifactPath(rel string, isDir bool) bool {
	name := filepath.Base(filepath.FromSlash(rel))
	if name == ".DS_Store" || name == ".agenix" || name == ".pytest_cache" || name == "__pycache__" {
		return true
	}
	if !isDir && strings.HasSuffix(name, ".pyc") {
		return true
	}
	return false
}
