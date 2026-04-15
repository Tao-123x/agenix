# Minimum Trace Redaction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add minimum viable trace redaction so Agenix masks common secrets before trace files are written to disk, while keeping `verify` and `replay` usable.

**Architecture:** Extend the manifest with a top-level `redaction` block, compile an effective redaction configuration from built-in defaults plus manifest additions, and sanitize a copy of the trace in `WriteTrace` before JSON serialization. Keep the event-collection APIs stable by introducing a focused redaction helper file and a runtime-only config hook on `Trace`.

**Tech Stack:** Go 1.22+, standard-library regex and JSON handling, existing hand-written manifest parser, Go test suite under `internal/agenix`.

---

## File Structure

**Create:**

- `internal/agenix/redaction.go`
- `internal/agenix/redaction_test.go`
- `docs/decisions/0004-minimum-trace-redaction.md`

**Modify:**

- `internal/agenix/manifest.go`
- `internal/agenix/schema.go`
- `internal/agenix/manifest_test.go`
- `internal/agenix/schema_test.go`
- `internal/agenix/trace.go`
- `internal/agenix/trace_test.go`
- `internal/agenix/runtime.go`
- `internal/agenix/runtime_integration_test.go`
- `specs/trace.md`
- `specs/skill-manifest.md`
- `docs/team/handoffs/2026-04-15-verifier-policy-contract.md`

**Why this structure:**

- `redaction.go` keeps rule compilation, key normalization, and recursive payload sanitization out of `trace.go`.
- `manifest.go` and `schema.go` remain the only places that define and validate manifest contract shape.
- `trace.go` only needs the plumbing required to attach runtime redaction config and sanitize before writing.
- `runtime.go` owns fail-closed behavior when a trace cannot be redacted and written safely.
- Separate focused tests keep manifest parsing, redaction engine behavior, and runtime integration independently understandable.

### Task 1: Add Manifest Redaction Contract And Validation

**Files:**
- Modify: `internal/agenix/manifest.go`
- Modify: `internal/agenix/schema.go`
- Modify: `internal/agenix/manifest_test.go`
- Modify: `internal/agenix/schema_test.go`

- [ ] **Step 1: Write the failing manifest parsing test for a valid top-level redaction block**

Add this test to `internal/agenix/manifest_test.go`:

```go
func TestLoadManifestParsesTopLevelRedactionBlock(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.yaml")
	raw := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
permissions:
  network: false
outputs:
  required:
    - patch_summary
verifiers:
  - type: schema
    name: output_schema_check
    schemaRef: outputs
redaction:
  keys:
    - session_token
  patterns:
    - name: customer-bearer
      regex: '(?i)(x-customer-token:\\s*)([^\\s]+)'
      secret_group: 2
`
	if err := os.WriteFile(manifestPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if len(manifest.Redaction.Keys) != 1 || manifest.Redaction.Keys[0] != "session_token" {
		t.Fatalf("redaction keys = %#v", manifest.Redaction.Keys)
	}
	if len(manifest.Redaction.Patterns) != 1 {
		t.Fatalf("redaction patterns = %#v", manifest.Redaction.Patterns)
	}
	if manifest.Redaction.Patterns[0].Name != "customer-bearer" {
		t.Fatalf("pattern name = %#v", manifest.Redaction.Patterns[0])
	}
}
```

- [ ] **Step 2: Run the manifest parsing test and verify it fails**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run TestLoadManifestParsesTopLevelRedactionBlock -count=1
```

Expected: `FAIL` because `Manifest` does not yet define `Redaction` and the parser ignores the block.

- [ ] **Step 3: Add the manifest redaction structs and parser support**

Update `internal/agenix/manifest.go` with these additions:

```go
type Manifest struct {
	Path        string                 `json:"path"`
	APIVersion  string                 `json:"apiVersion"`
	Kind        string                 `json:"kind"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Tools       []string               `json:"tools"`
	Permissions Permissions            `json:"permissions"`
	Inputs      map[string]string      `json:"inputs"`
	Outputs     OutputSchema           `json:"outputs"`
	Verifiers   []Verifier             `json:"verifiers"`
	Redaction   RedactionConfig        `json:"redaction,omitempty"`
	Recovery    map[string]interface{} `json:"recovery,omitempty"`
}

type RedactionConfig struct {
	Keys     []string             `json:"keys,omitempty"`
	Patterns []RedactionPattern   `json:"patterns,omitempty"`
}

type RedactionPattern struct {
	Name        string `json:"name"`
	Regex       string `json:"regex"`
	SecretGroup int    `json:"secret_group"`
}
```

Add a new top-level parse branch inside `LoadManifest`:

```go
		case "redaction":
			if indent == 2 {
				key, _, _ := splitKeyValue(trimmed)
				sub = key
				continue
			}
			if sub == "keys" && strings.HasPrefix(trimmed, "- ") {
				manifest.Redaction.Keys = append(manifest.Redaction.Keys, cleanScalar(strings.TrimPrefix(trimmed, "- ")))
				continue
			}
			if sub == "patterns" && strings.HasPrefix(trimmed, "- name:") {
				pattern := RedactionPattern{Name: cleanScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))}
				manifest.Redaction.Patterns = append(manifest.Redaction.Patterns, pattern)
				continue
			}
```

Then finish the parser with a `currentPattern` pointer, mirroring the existing verifier parsing style:

```go
	var currentPattern *RedactionPattern
```

and:

```go
			if sub == "patterns" && strings.HasPrefix(trimmed, "- name:") {
				pattern := RedactionPattern{Name: cleanScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))}
				manifest.Redaction.Patterns = append(manifest.Redaction.Patterns, pattern)
				currentPattern = &manifest.Redaction.Patterns[len(manifest.Redaction.Patterns)-1]
				continue
			}
			if sub == "patterns" && currentPattern != nil && indent == 4 {
				key, value, ok := splitKeyValue(trimmed)
				if !ok {
					continue
				}
				switch key {
				case "regex":
					currentPattern.Regex = cleanScalar(value)
				case "secret_group":
					secretGroup, _ := strconv.Atoi(cleanScalar(value))
					currentPattern.SecretGroup = secretGroup
				}
			}
```

- [ ] **Step 4: Add validation tests for invalid redaction configuration**

Append this table-driven test to `internal/agenix/schema_test.go`:

```go
func TestLoadManifestRejectsInvalidRedactionPatterns(t *testing.T) {
	valid := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
permissions:
  network: false
outputs:
  required:
    - patch_summary
verifiers:
  - type: schema
    name: output_schema_check
    schemaRef: outputs
redaction:
  patterns:
    - name: customer-bearer
      regex: '(?i)(x-customer-token:\\s*)([^\\s]+)'
      secret_group: 2
`
	tests := []struct {
		name string
		old  string
		new  string
	}{
		{name: "missing name", old: "    - name: customer-bearer\n", new: "    - name:\n"},
		{name: "missing regex", old: "      regex: '(?i)(x-customer-token:\\\\s*)([^\\\\s]+)'\n", new: "      regex:\n"},
		{name: "invalid regex", old: "      regex: '(?i)(x-customer-token:\\\\s*)([^\\\\s]+)'\n", new: "      regex: '('\n"},
		{name: "nonpositive secret_group", old: "      secret_group: 2\n", new: "      secret_group: 0\n"},
		{name: "secret_group out of range", old: "      secret_group: 2\n", new: "      secret_group: 3\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "manifest.yaml")
			raw := strings.Replace(valid, tt.old, tt.new, 1)
			if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
				t.Fatal(err)
			}

			_, err := LoadManifest(path)
			if err == nil {
				t.Fatal("expected InvalidInput error")
			}
			if !IsErrorClass(err, ErrInvalidInput) {
				t.Fatalf("expected InvalidInput, got %v", err)
			}
		})
	}
}
```

- [ ] **Step 5: Implement validation for `manifest.redaction`**

Extend `internal/agenix/schema.go` with:

```go
import (
	"fmt"
	"regexp"
)
```

and add this helper:

```go
func validateRedactionConfig(config RedactionConfig) error {
	for i, pattern := range config.Patterns {
		if pattern.Name == "" {
			return missingField("manifest", fmt.Sprintf("redaction.patterns[%d].name", i))
		}
		if pattern.Regex == "" {
			return missingField("manifest", fmt.Sprintf("redaction.patterns[%d].regex", i))
		}
		compiled, err := regexp.Compile(pattern.Regex)
		if err != nil {
			return WrapError(ErrInvalidInput, "manifest redaction pattern regex", err)
		}
		if pattern.SecretGroup <= 0 {
			return NewError(ErrInvalidInput, "manifest redaction secret_group must be greater than zero")
		}
		if pattern.SecretGroup > compiled.NumSubexp() {
			return NewError(ErrInvalidInput, "manifest redaction secret_group exceeds regex capture groups")
		}
	}
	return nil
}
```

Call it from `ValidateManifest` immediately after the required-field checks:

```go
	if err := validateRedactionConfig(manifest.Redaction); err != nil {
		return err
	}
```

- [ ] **Step 6: Re-run the focused manifest tests and verify they pass**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run "TestLoadManifestParsesTopLevelRedactionBlock|TestLoadManifestRejectsInvalidRedactionPatterns" -count=1
```

Expected:

- `ok   agenix/internal/agenix`

- [ ] **Step 7: Commit the manifest contract work**

```powershell
git add internal/agenix/manifest.go internal/agenix/schema.go internal/agenix/manifest_test.go internal/agenix/schema_test.go
git commit -m "feat: add manifest redaction contract"
```

### Task 2: Build The Redaction Engine In A Focused Helper

**Files:**
- Create: `internal/agenix/redaction.go`
- Create: `internal/agenix/redaction_test.go`

- [ ] **Step 1: Write focused failing tests for key matching and text masking**

Create `internal/agenix/redaction_test.go` with:

```go
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
```

- [ ] **Step 2: Run the focused redaction tests and verify they fail**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run "TestRedactValueMasksSensitiveKeysAndPreservesShape|TestRedactTextMasksBuiltInAndCustomPatternsPrecisely" -count=1
```

Expected: `FAIL` because `compileRedactionConfig`, `redactValue`, and `redactText` do not exist yet.

- [ ] **Step 3: Implement the redaction engine**

Create `internal/agenix/redaction.go` with this structure:

```go
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
			return compiledRedactionConfig{}, WrapError(ErrDriverError, "compile redaction regex", err)
		}
		config.patterns = append(config.patterns, compiledPattern{
			name:        pattern.Name,
			regex:       compiled,
			secretGroup: pattern.SecretGroup,
		})
	}
	return config, nil
}
```

Then add the recursive helpers:

```go
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
```

And the text rule application:

```go
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
```

Finish with:

```go
func normalizeRedactionKey(key string) string {
	trimmed := strings.TrimSpace(strings.ToLower(key))
	trimmed = strings.ReplaceAll(trimmed, "-", "_")
	return trimmed
}
```

and built-in rules:

```go
func defaultRedactionKeys() []string {
	return []string{
		"authorization",
		"api_key",
		"access_token",
		"refresh_token",
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
```

- [ ] **Step 4: Re-run the focused redaction tests and verify they pass**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run "TestRedactValueMasksSensitiveKeysAndPreservesShape|TestRedactTextMasksBuiltInAndCustomPatternsPrecisely" -count=1
```

Expected:

- `ok   agenix/internal/agenix`

- [ ] **Step 5: Commit the redaction engine**

```powershell
git add internal/agenix/redaction.go internal/agenix/redaction_test.go
git commit -m "feat: add trace redaction engine"
```

### Task 3: Wire Redaction Into Trace Writing And Runtime Fail-Closed Behavior

**Files:**
- Modify: `internal/agenix/trace.go`
- Modify: `internal/agenix/runtime.go`
- Modify: `internal/agenix/trace_test.go`
- Modify: `internal/agenix/runtime_integration_test.go`

- [ ] **Step 1: Write the failing trace tests for built-in and manifest-driven redaction**

Append these tests to `internal/agenix/trace_test.go`:

```go
func TestWriteTraceRedactsSensitiveValuesBeforePersisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{Network: false})
	trace.AddToolEvent("shell.exec", map[string]any{
		"Authorization": "Bearer topsecret",
		"cmd":           []string{"python3", "-m", "pytest", "-q"},
	}, map[string]any{
		"api_key": "sk-live",
		"path":    "repo/demo.py",
	}, nil, 12)
	trace.AddVerifierEvent("run_tests", "command", "failed", map[string]any{"type": "command"}, ShellResult{
		Stdout: "Authorization: Bearer topsecret",
		Stderr: "OPENAI_API_KEY=sk-test",
	}, nil)
	trace.SetFinal("failed", map[string]any{
		"session_token": "secret-token",
		"changed_files": []string{"repo/demo.py"},
	}, "password=hunter2")

	if err := WriteTrace(path, trace); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, forbidden := range []string{"topsecret", "sk-live", "sk-test", "secret-token", "hunter2"} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("trace leaked %q: %s", forbidden, text)
		}
	}
	for _, wanted := range []string{"repo/demo.py", "\"changed_files\"", "\"cmd\""} {
		if !strings.Contains(text, wanted) {
			t.Fatalf("trace lost %q: %s", wanted, text)
		}
	}
}

func TestWriteTraceUsesManifestAddedRedactionRules(t *testing.T) {
	path := filepath.Join(t.TempDir(), "trace.json")
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{})
	trace.SetRedaction(RedactionConfig{
		Keys: []string{"session_token"},
		Patterns: []RedactionPattern{
			{
				Name:        "customer-bearer",
				Regex:       `(?i)(x-customer-token:\\s*)([^\\s]+)`,
				SecretGroup: 2,
			},
		},
	})
	trace.AddToolEvent("shell.exec", map[string]any{
		"session_token": "value-123",
	}, nil, nil, 5)
	trace.SetFinal("failed", map[string]any{}, "x-customer-token: hello")

	if err := WriteTrace(path, trace); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "value-123") || strings.Contains(text, "hello") {
		t.Fatalf("trace leaked manifest-defined secret: %s", text)
	}
}
```

- [ ] **Step 2: Run the focused trace redaction tests and verify they fail**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run "TestWriteTraceRedactsSensitiveValuesBeforePersisting|TestWriteTraceUsesManifestAddedRedactionRules" -count=1
```

Expected: `FAIL` because `Trace` does not yet carry redaction config and `WriteTrace` still writes raw values.

- [ ] **Step 3: Add runtime-only redaction plumbing to `Trace` and sanitize on write**

Update `internal/agenix/trace.go`:

```go
type Trace struct {
	RunID        string       `json:"run_id"`
	Skill        string       `json:"skill"`
	ManifestPath string       `json:"manifest_path,omitempty"`
	ModelProfile string       `json:"model_profile"`
	StartedAt    time.Time    `json:"started_at"`
	Policy       Permissions  `json:"policy"`
	Events       []TraceEvent `json:"events"`
	Final        TraceFinal   `json:"final"`
	redaction    RedactionConfig
}

func (t *Trace) SetRedaction(config RedactionConfig) {
	t.redaction = config
}
```

Add a helper to produce the persisted copy:

```go
func sanitizeTrace(trace *Trace) (*Trace, error) {
	config, err := compileRedactionConfig(trace.redaction)
	if err != nil {
		return nil, err
	}
	sanitized := *trace
	sanitized.Events = make([]TraceEvent, 0, len(trace.Events))
	for _, event := range trace.Events {
		copyEvent := event
		copyEvent.Request = redactValue(event.Request, config)
		copyEvent.Result = redactValue(event.Result, config)
		copyEvent.Error = redactValue(event.Error, config)
		copyEvent.Stdout = redactText(event.Stdout, config)
		copyEvent.Stderr = redactText(event.Stderr, config)
		sanitized.Events = append(sanitized.Events, copyEvent)
	}
	sanitized.Final = TraceFinal{
		Status: trace.Final.Status,
		Output: redactValue(trace.Final.Output, config),
		Error:  redactText(trace.Final.Error, config),
	}
	sanitized.redaction = RedactionConfig{}
	return &sanitized, nil
}
```

Then change `WriteTrace`:

```go
func WriteTrace(path string, trace *Trace) error {
	if err := ensureParent(path); err != nil {
		return WrapError(ErrDriverError, "create trace directory", err)
	}
	sanitized, err := sanitizeTrace(trace)
	if err != nil {
		return WrapError(ErrDriverError, "sanitize trace", err)
	}
	raw, err := json.MarshalIndent(sanitized, "", "  ")
	if err != nil {
		return WrapError(ErrDriverError, "encode trace", err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		return WrapError(ErrDriverError, "write trace", err)
	}
	return nil
}
```

- [ ] **Step 4: Attach manifest redaction config in runtime and fail closed on trace write errors**

Update `internal/agenix/runtime.go` after `NewTrace(...)`:

```go
	trace := NewTrace(manifest.Name, fakeModelProfile, manifest.Permissions)
	trace.RunID = runID
	trace.ManifestPath = manifestPath
	trace.SetRedaction(manifest.Redaction)
```

Then replace the ignored write errors:

```go
		if writeErr := WriteTrace(tracePath, trace); writeErr != nil {
			return result, writeErr
		}
		return result, err
```

Apply the same pattern to:

- policy creation failure branch
- adapter failure branch
- verifier failure branch

The passing branch already returns the write error; leave that shape in place.

- [ ] **Step 5: Add an integration test that proves `verify` still works on a redacted trace**

Append this to `internal/agenix/runtime_integration_test.go`:

```go
func TestVerifyAcceptsRedactedTraceThatKeepsAuditFields(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, false)
	manifestPath := filepath.Join(root, "manifest.yaml")
	content := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
  - shell
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
  shell:
    allow:
      - run: ["python3", "-c", "print('Authorization: Bearer topsecret')"]
inputs:
  repo_path: ` + repo + `
outputs:
  required:
    - patch_summary
    - changed_files
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-c", "print('OPENAI_API_KEY=sk-test')"]
    cwd: ${repo_path}
    policy:
      executable: python3
      cwd: ${repo_path}
      timeout_ms: 120000
    success:
      exit_code: 0
  - type: schema
    name: output_schema_check
    schemaRef: outputs
redaction:
  keys:
    - session_token
recovery:
  strategy: checkpoint
  intervals: 5
`
	if err := os.WriteFile(manifestPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{})
	trace.ManifestPath = manifestPath
	trace.SetRedaction(RedactionConfig{Keys: []string{"session_token"}})
	trace.AddToolEvent("shell.exec", map[string]any{"cmd": []string{"python3"}, "session_token": "abc"}, map[string]any{"path": filepath.Join(repo, "mathlib.py")}, nil, 5)
	trace.AddVerifierEvent("run_tests", "command", "passed", map[string]any{"cmd": []string{"python3"}}, ShellResult{Stdout: "OPENAI_API_KEY=sk-test", ExitCode: 0}, nil)
	trace.SetFinal("passed", map[string]any{"patch_summary": "ok", "changed_files": []string{filepath.Join(repo, "mathlib.py")}}, "")
	tracePath := filepath.Join(root, "trace.json")
	if err := WriteTrace(tracePath, trace); err != nil {
		t.Fatal(err)
	}

	raw, err := os.ReadFile(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "sk-test") || strings.Contains(string(raw), "\"session_token\":\"abc\"") {
		t.Fatalf("trace leaked secret: %s", raw)
	}

	result, err := Verify(tracePath)
	if err != nil {
		t.Fatalf("Verify returned error: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("verify status = %q", result.Status)
	}
}
```

- [ ] **Step 6: Re-run focused trace tests and the full internal package**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run "TestWriteTraceRedactsSensitiveValuesBeforePersisting|TestWriteTraceUsesManifestAddedRedactionRules|TestVerifyAcceptsRedactedTraceThatKeepsAuditFields" -count=1
go test ./internal/agenix -count=1
```

Expected:

- `ok   agenix/internal/agenix`
- `ok   agenix/internal/agenix`

- [ ] **Step 7: Commit the trace and runtime wiring**

```powershell
git add internal/agenix/trace.go internal/agenix/runtime.go internal/agenix/trace_test.go internal/agenix/runtime_integration_test.go
git commit -m "feat: redact trace output before persistence"
```

### Task 4: Update Contract Docs And Record The Decision

**Files:**
- Modify: `specs/trace.md`
- Modify: `specs/skill-manifest.md`
- Create: `docs/decisions/0004-minimum-trace-redaction.md`
- Modify: `docs/team/handoffs/2026-04-15-verifier-policy-contract.md`

- [ ] **Step 1: Update the public trace spec**

Replace the redaction section in `specs/trace.md` with:

```md
## Redaction

- No secrets in persisted trace files.
- Runtime applies built-in redaction rules for common secret-bearing keys and
  text patterns before writing trace JSON.
- Skills may append additional redaction rules through a top-level
  `redaction.keys` and `redaction.patterns` manifest block.
- Redaction should preserve surrounding audit context and replace only the
  secret value with `[REDACTED]` when possible.
- If trace redaction fails, the runtime must fail closed and refuse to write the
  trace.
```

- [ ] **Step 2: Document the new manifest contract**

Append this to `specs/skill-manifest.md` under the verifier notes:

```md
- Skills may declare a top-level `redaction` block.
- `redaction.keys` appends structured sensitive field names to the runtime
  default set.
- `redaction.patterns` appends text masking rules using `name`, `regex`, and
  `secret_group`.
- Invalid redaction patterns must fail manifest load as `InvalidInput`.
```

- [ ] **Step 3: Record the decision**

Create `docs/decisions/0004-minimum-trace-redaction.md`:

```md
# Decision Record: Minimum Trace Redaction

## Status

`accepted`

## Context

Maya Chen's P0 trial criteria require minimum trace redaction because secrets
can currently land in verifier stdout, stderr, tool payloads, and final output.

## Decision

Persist only redacted traces.

- Runtime applies built-in redaction rules for common secret-bearing keys and
  text patterns.
- Skills may append additional `redaction.keys` and `redaction.patterns`.
- Redaction runs in `WriteTrace` against a sanitized copy of the trace.
- If redaction fails, trace persistence fails closed.

## Consequences

- `verify` and `replay` consume already-redacted trace files.
- Audit context such as paths, commands, statuses, and timing remains visible
  unless the value itself is a secret.
- In-memory traces may still contain raw values until write time in this slice.
```

- [ ] **Step 4: Add a handoff note**

Append this bullet to `docs/team/handoffs/2026-04-15-verifier-policy-contract.md`:

```md
- minimum trace redaction now masks built-in secret patterns plus manifest-added redaction rules before trace files are written
```

- [ ] **Step 5: Commit the docs**

```powershell
git add specs/trace.md specs/skill-manifest.md docs/decisions/0004-minimum-trace-redaction.md docs/team/handoffs/2026-04-15-verifier-policy-contract.md
git commit -m "docs: record minimum trace redaction"
```

### Task 5: Final Verification And Push

**Files:**
- Verify all files changed in Tasks 1-4

- [ ] **Step 1: Run the full branch verification**

Run:

```powershell
New-Item -ItemType Directory -Force .tmp-go | Out-Null
$env:GOTMPDIR=(Resolve-Path .tmp-go).Path
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./cmd/agenix -count=1
go test ./internal/agenix -count=1
go test ./... -count=1
```

Expected:

- `ok   agenix/cmd/agenix`
- `ok   agenix/internal/agenix`
- `ok` for all packages under `./...`

- [ ] **Step 2: Clean temporary files and inspect the worktree**

Run:

```powershell
if (Test-Path .tmp-go) { cmd /C rmdir /S /Q .tmp-go }
git status --short --branch
```

Expected: the worktree is clean after the planned commits.

- [ ] **Step 3: Push the branch**

```powershell
git push -u origin codex/minimum-trace-redaction
```

Expected: remote branch `codex/minimum-trace-redaction` updates successfully.
