package agenix

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishArtifactCopiesCapsuleAndIndexesIt(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	artifactPath := filepath.Join(root, "repo.fix_test_failure.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath}); err != nil {
		t.Fatal(err)
	}

	entry, err := PublishArtifact(PublishOptions{
		ArtifactPath: artifactPath,
		RegistryRoot: filepath.Join(root, "registry"),
	})
	if err != nil {
		t.Fatalf("PublishArtifact returned error: %v", err)
	}
	if entry.Skill != "repo.fix_test_failure" || entry.Version != "0.1.0" {
		t.Fatalf("unexpected entry: %#v", entry)
	}
	if entry.Digest == "" || entry.ArtifactPath == "" {
		t.Fatalf("missing digest/path: %#v", entry)
	}
	if _, err := os.Stat(entry.ArtifactPath); err != nil {
		t.Fatalf("published artifact missing: %v", err)
	}

	indexPath := filepath.Join(root, "registry", "index.json")
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatalf("expected index.json: %v", err)
	}
	var index RegistryIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatalf("decode index: %v", err)
	}
	if len(index.Entries) != 1 || index.Entries[0].Digest != entry.Digest {
		t.Fatalf("unexpected index: %#v", index)
	}
}

func TestPublishArtifactIsIdempotentForSameDigest(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	artifactPath := filepath.Join(root, "repo.fix_test_failure.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath}); err != nil {
		t.Fatal(err)
	}
	registryRoot := filepath.Join(root, "registry")

	first, err := PublishArtifact(PublishOptions{ArtifactPath: artifactPath, RegistryRoot: registryRoot})
	if err != nil {
		t.Fatal(err)
	}
	second, err := PublishArtifact(PublishOptions{ArtifactPath: artifactPath, RegistryRoot: registryRoot})
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest != second.Digest || first.ArtifactPath != second.ArtifactPath {
		t.Fatalf("republish mismatch: first=%#v second=%#v", first, second)
	}

	raw, err := os.ReadFile(filepath.Join(registryRoot, "index.json"))
	if err != nil {
		t.Fatal(err)
	}
	var index RegistryIndex
	if err := json.Unmarshal(raw, &index); err != nil {
		t.Fatal(err)
	}
	if len(index.Entries) != 1 {
		t.Fatalf("expected one index entry, got %#v", index)
	}
}

func TestPublishArtifactRejectsDifferentDigestForSameSkillVersion(t *testing.T) {
	root := t.TempDir()
	registryRoot := filepath.Join(root, "registry")

	firstSkill := writeCapsuleSkill(t, filepath.Join(root, "first"))
	firstArtifact := filepath.Join(root, "first.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: firstSkill, OutputPath: firstArtifact}); err != nil {
		t.Fatal(err)
	}
	if _, err := PublishArtifact(PublishOptions{ArtifactPath: firstArtifact, RegistryRoot: registryRoot}); err != nil {
		t.Fatal(err)
	}

	secondSkill := writeCapsuleSkill(t, filepath.Join(root, "second"))
	if err := os.WriteFile(filepath.Join(secondSkill, "README.md"), []byte("# changed\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	secondArtifact := filepath.Join(root, "second.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: secondSkill, OutputPath: secondArtifact}); err != nil {
		t.Fatal(err)
	}

	_, err := PublishArtifact(PublishOptions{ArtifactPath: secondArtifact, RegistryRoot: registryRoot})
	if err == nil {
		t.Fatal("expected publish conflict error")
	}
	if !IsErrorClass(err, ErrInvalidInput) {
		t.Fatalf("expected InvalidInput, got %v", err)
	}
}

func TestPullArtifactByDigestCopiesRequestedArtifact(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	artifactPath := filepath.Join(root, "repo.fix_test_failure.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath}); err != nil {
		t.Fatal(err)
	}
	registryRoot := filepath.Join(root, "registry")
	entry, err := PublishArtifact(PublishOptions{ArtifactPath: artifactPath, RegistryRoot: registryRoot})
	if err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(root, "pulled.agenix")
	summary, err := PullArtifact(PullOptions{
		Reference:    entry.Digest,
		OutputPath:   outputPath,
		RegistryRoot: registryRoot,
	})
	if err != nil {
		t.Fatalf("PullArtifact returned error: %v", err)
	}
	if summary.Digest != entry.Digest {
		t.Fatalf("summary digest = %q, want %q", summary.Digest, entry.Digest)
	}
	if summary.Path != outputPath {
		t.Fatalf("summary path = %q, want %q", summary.Path, outputPath)
	}
}

func TestPullArtifactBySkillVersionCopiesRequestedArtifact(t *testing.T) {
	root := t.TempDir()
	skillDir := writeCapsuleSkill(t, root)
	artifactPath := filepath.Join(root, "repo.fix_test_failure.agenix")
	if _, err := BuildArtifact(BuildOptions{SkillDir: skillDir, OutputPath: artifactPath}); err != nil {
		t.Fatal(err)
	}
	registryRoot := filepath.Join(root, "registry")
	if _, err := PublishArtifact(PublishOptions{ArtifactPath: artifactPath, RegistryRoot: registryRoot}); err != nil {
		t.Fatal(err)
	}

	outputPath := filepath.Join(root, "pulled.agenix")
	summary, err := PullArtifact(PullOptions{
		Reference:    "repo.fix_test_failure@0.1.0",
		OutputPath:   outputPath,
		RegistryRoot: registryRoot,
	})
	if err != nil {
		t.Fatalf("PullArtifact returned error: %v", err)
	}
	if summary.Skill != "repo.fix_test_failure" || summary.Version != "0.1.0" {
		t.Fatalf("unexpected summary: %#v", summary)
	}
	if _, err := InspectArtifact(outputPath); err != nil {
		t.Fatalf("pulled artifact should inspect cleanly: %v", err)
	}
}
