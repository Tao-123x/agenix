package agenix

import (
	"regexp"
	"strings"
)

type compiledPattern struct {
	name        string
	regex       *regexp.Regexp
	secretGroup int
}

type compiledRedactionConfig struct {
	keys     map[string]struct{}
	patterns []compiledPattern
}

func compileRedactionConfig(extra RedactionConfig) (compiledRedactionConfig, error) {
	config := compiledRedactionConfig{
		keys: map[string]struct{}{},
	}
	for _, key := range defaultRedactionKeys() {
		config.keys[normalizeRedactionKey(key)] = struct{}{}
	}
	for _, key := range extra.Keys {
		config.keys[normalizeRedactionKey(key)] = struct{}{}
	}
	patterns := append(defaultRedactionPatterns(), extra.Patterns...)
	for _, pattern := range patterns {
		compiled, err := regexp.Compile(pattern.Regex)
		if err != nil {
			return compiledRedactionConfig{}, WrapError(ErrInvalidInput, "compile redaction regex", err)
		}
		config.patterns = append(config.patterns, compiledPattern{
			name:        pattern.Name,
			regex:       compiled,
			secretGroup: pattern.SecretGroup,
		})
	}
	return config, nil
}

func redactValue(value any, config compiledRedactionConfig) any {
	switch typed := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(typed))
		for key, item := range typed {
			if _, ok := config.keys[normalizeRedactionKey(key)]; ok {
				if _, ok := item.(string); ok {
					out[key] = "[REDACTED]"
					continue
				}
			}
			out[key] = redactValue(item, config)
		}
		return out
	case map[string]string:
		out := make(map[string]string, len(typed))
		for key, item := range typed {
			if _, ok := config.keys[normalizeRedactionKey(key)]; ok {
				out[key] = "[REDACTED]"
				continue
			}
			out[key] = redactText(item, config)
		}
		return out
	case []any:
		out := make([]any, 0, len(typed))
		for _, item := range typed {
			out = append(out, redactValue(item, config))
		}
		return out
	case []string:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, redactText(item, config))
		}
		return out
	case string:
		return redactText(typed, config)
	default:
		return value
	}
}

func redactText(text string, config compiledRedactionConfig) string {
	out := text
	for _, pattern := range config.patterns {
		matches := pattern.regex.FindAllStringSubmatchIndex(out, -1)
		if len(matches) == 0 {
			continue
		}
		var builder strings.Builder
		cursor := 0
		for _, match := range matches {
			groupStart := match[2*pattern.secretGroup]
			groupEnd := match[2*pattern.secretGroup+1]
			builder.WriteString(out[cursor:groupStart])
			builder.WriteString("[REDACTED]")
			cursor = groupEnd
		}
		builder.WriteString(out[cursor:])
		out = builder.String()
	}
	return out
}

func normalizeRedactionKey(key string) string {
	trimmed := strings.TrimSpace(strings.ToLower(key))
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	return trimmed
}

func defaultRedactionKeys() []string {
	return []string{
		"authorization",
		"api_key",
		"access_token",
		"refresh_token",
		"session_token",
		"token",
		"secret",
		"password",
	}
}

func defaultRedactionPatterns() []RedactionPattern {
	return []RedactionPattern{
		{Name: "authorization-bearer", Regex: `(?i)(authorization:\s*bearer\s+)([^\s]+)`, SecretGroup: 2},
		{Name: "bare-bearer", Regex: `(?i)(bearer\s+)([^\s]+)`, SecretGroup: 2},
		{Name: "openai-api-key", Regex: `(?i)(openai_api_key=)([^\s]+)`, SecretGroup: 2},
		{Name: "generic-api-key", Regex: `(?i)([a-z0-9_]*api_key=)([^\s]+)`, SecretGroup: 2},
		{Name: "token-equals", Regex: `(?i)(token=)([^\s]+)`, SecretGroup: 2},
		{Name: "password-equals", Regex: `(?i)(password=)([^\s]+)`, SecretGroup: 2},
	}
}
