package agenix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestV0ReleaseGateWorkflowRunsCanonicalVerification(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "v0-release-gate.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read v0 release gate workflow: %v", err)
	}
	text := string(raw)
	for _, command := range []string{
		"go run ./cmd/agenix acceptance",
		"go run ./cmd/agenix acceptance --v0.2",
		"go run ./cmd/agenix acceptance --v0.3",
		"go test -count=1 ./...",
		"go vet ./...",
		"go build ./cmd/agenix",
	} {
		if !strings.Contains(text, command) {
			t.Fatalf("workflow missing %q:\n%s", command, text)
		}
	}
}

func TestV0ReleaseGateWorkflowRunsOnAllSupportedHosts(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "v0-release-gate.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read v0 release gate workflow: %v", err)
	}
	text := string(raw)
	for _, snippet := range []string{
		"strategy:",
		"fail-fast: false",
		"runs-on: ${{ matrix.os }}",
		"ubuntu-latest",
		"macos-latest",
		"windows-latest",
	} {
		if !strings.Contains(text, snippet) {
			t.Fatalf("workflow missing cross-OS matrix snippet %q:\n%s", snippet, text)
		}
	}
}

func TestV0ReleaseGateWorkflowDisablesGoCacheWithoutGoSum(t *testing.T) {
	path := filepath.Join("..", "..", ".github", "workflows", "v0-release-gate.yml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read v0 release gate workflow: %v", err)
	}
	text := string(raw)
	if !strings.Contains(text, "cache: false") {
		t.Fatalf("workflow should disable setup-go cache when go.sum is absent:\n%s", text)
	}
}
