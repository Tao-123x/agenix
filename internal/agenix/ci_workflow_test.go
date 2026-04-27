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
		"go test -count=1 ./...",
		"go vet ./...",
		"go build ./cmd/agenix",
	} {
		if !strings.Contains(text, command) {
			t.Fatalf("workflow missing %q:\n%s", command, text)
		}
	}
}
