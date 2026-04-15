package agenix

import (
	"strings"
	"testing"
)

func TestRedactValueMasksSensitiveKeysAndPreservesShape(t *testing.T) {
	config, err := compileRedactionConfig(RedactionConfig{
		Keys: []string{"session_token"},
	})
	if err != nil {
		t.Fatal(err)
	}

	input := map[string]any{
		"Authorization": "Bearer abc123",
		"session_token": "secret-token",
		"path":          "repo/demo.py",
		"nested": map[string]any{
			"api_key": "xyz",
		},
	}

	got := redactValue(input, config).(map[string]any)
	if got["Authorization"] != "[REDACTED]" {
		t.Fatalf("Authorization = %#v", got["Authorization"])
	}
	if got["session_token"] != "[REDACTED]" {
		t.Fatalf("session_token = %#v", got["session_token"])
	}
	if got["path"] != "repo/demo.py" {
		t.Fatalf("path = %#v", got["path"])
	}
	nested := got["nested"].(map[string]any)
	if nested["api_key"] != "[REDACTED]" {
		t.Fatalf("nested api_key = %#v", nested["api_key"])
	}
}

func TestRedactTextMasksBuiltInAndCustomPatternsPrecisely(t *testing.T) {
	config, err := compileRedactionConfig(RedactionConfig{
		Patterns: []RedactionPattern{
			{
				Name:        "customer-bearer",
				Regex:       `(?i)(x-customer-token:\s*)([^\s]+)`,
				SecretGroup: 2,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	text := "Authorization: Bearer topsecret\nOPENAI_API_KEY=sk-abc\nx-customer-token: hello"
	got := redactText(text, config)

	for _, want := range []string{
		"Authorization: Bearer [REDACTED]",
		"OPENAI_API_KEY=[REDACTED]",
		"x-customer-token: [REDACTED]",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in %q", want, got)
		}
	}
}
