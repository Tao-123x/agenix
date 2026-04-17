# Provider-Backed Adapter Spike Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add one provider-backed read-only adapter that can call a real remote model when credentials are present, while keeping default tests offline and enforcing `permissions.network` as a hard runtime boundary.

**Architecture:** Extend adapter metadata with an explicit `transport` field and insert a runtime `adapter.policy_check` stage before execution. Implement a narrow OpenAI-specific provider client plus a single `openai-analyze` adapter for a remote read-only manifest variant, then verify the new path with offline stub tests and opt-in manual smoke docs.

**Tech Stack:** Go 1.22+, stdlib `net/http`, `httptest`, existing Agenix runtime/trace/policy/test helpers

---

### Task 1: Add transport metadata and remote policy preflight

**Files:**
- Modify: `internal/agenix/runtime.go`
- Modify: `internal/agenix/adapter_builtin.go`
- Modify: `internal/agenix/runtime_integration_test.go`
- Test: `internal/agenix/runtime_integration_test.go`

- [ ] **Step 1: Write the failing runtime tests**

Add these tests to `internal/agenix/runtime_integration_test.go`:

```go
func TestRuntimeRejectsRemoteAdapterWhenManifestDisablesNetwork(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, true)
	manifestPath := filepath.Join(root, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(`apiVersion: agenix/v0.1
kind: Skill
name: repo.analyze_test_failures.remote
version: 0.1.0
description: Remote analyze fixture.
capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 32000
    reasoning_level: medium
tools:
  - fs
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
  shell:
    allow:
inputs:
  repo_path: `+repo+`
outputs:
  required:
    - analysis_summary
    - failing_tests
    - likely_root_cause
    - changed_files
verifiers:
  - type: command
    name: fixture_still_fails
    run: ["python3", "verify_failing.py"]
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
recovery:
  strategy: checkpoint
  intervals: 5
`), 0o600); err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(root, ".agenix-runs")

	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter: remoteMetadataOnlyAdapter{
			metadata: AdapterMetadata{
				Name:            "openai-analyze",
				ModelProfile:    "openai:gpt-5.4-mini",
				Transport:       "remote",
				SupportedSkills: []string{"repo.analyze_test_failures.remote"},
				Capabilities: CapabilitySet{
					ToolCalling:      true,
					StructuredOutput: true,
					MaxContextTokens: 128000,
					ReasoningLevel:   "medium",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected remote policy failure")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	trace, readErr := ReadTrace(result.TracePath)
	if readErr != nil {
		t.Fatalf("ReadTrace returned error: %v", readErr)
	}
	if !traceHasAdapterEvent(*trace, "policy_check", "failed") {
		t.Fatalf("trace does not contain failed adapter policy_check event: %#v", trace.Events)
	}
	if traceHasAdapterEvent(*trace, "execute", "ok") || traceHasAdapterEvent(*trace, "execute", "failed") {
		t.Fatalf("remote policy failure should happen before execute: %#v", trace.Events)
	}
}

func TestRuntimeRecordsSuccessfulRemotePolicyCheckBeforeExecute(t *testing.T) {
	root := t.TempDir()
	repo := writePythonFixture(t, root, true)
	manifestPath := filepath.Join(root, "manifest.yaml")
	if err := os.WriteFile(manifestPath, []byte(`apiVersion: agenix/v0.1
kind: Skill
name: repo.analyze_test_failures.remote
version: 0.1.0
description: Remote analyze fixture.
capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 32000
    reasoning_level: medium
tools:
  - fs
permissions:
  network: true
  filesystem:
    read:
      - ${repo_path}
    write:
  shell:
    allow:
inputs:
  repo_path: `+repo+`
outputs:
  required:
    - analysis_summary
    - failing_tests
    - likely_root_cause
    - changed_files
verifiers:
  - type: command
    name: fixture_still_fails
    run: ["python3", "verify_failing.py"]
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
recovery:
  strategy: checkpoint
  intervals: 5
`), 0o600); err != nil {
		t.Fatal(err)
	}
	runDir := filepath.Join(root, ".agenix-runs")
	called := false

	_, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter: remoteMetadataOnlyAdapter{
			called: &called,
			metadata: AdapterMetadata{
				Name:            "openai-analyze",
				ModelProfile:    "openai:gpt-5.4-mini",
				Transport:       "remote",
				SupportedSkills: []string{"repo.analyze_test_failures.remote"},
				Capabilities: CapabilitySet{
					ToolCalling:      true,
					StructuredOutput: true,
					MaxContextTokens: 128000,
					ReasoningLevel:   "medium",
				},
			},
		},
	})
	if err == nil {
		t.Fatal("expected downstream failure from metadata-only remote adapter")
	}
	if !called {
		t.Fatal("expected Execute to run after successful remote policy check")
	}
}
```

- [ ] **Step 2: Run the runtime tests to verify they fail**

Run:

```bash
go test ./internal/agenix -run 'TestRuntimeRejectsRemoteAdapterWhenManifestDisablesNetwork|TestRuntimeRecordsSuccessfulRemotePolicyCheckBeforeExecute' -count=1
```

Expected: FAIL with compile errors for missing `Transport`, missing `policy_check`, or missing `remoteMetadataOnlyAdapter`.

- [ ] **Step 3: Add transport metadata and policy preflight**

Update `internal/agenix/runtime.go` and `internal/agenix/adapter_builtin.go` with the minimal implementation:

```go
type AdapterMetadata struct {
	Name            string        `json:"name"`
	ModelProfile    string        `json:"model_profile"`
	Transport       string        `json:"transport,omitempty"`
	SupportedSkills []string      `json:"supported_skills,omitempty"`
	Capabilities    CapabilitySet `json:"capabilities"`
}

func normalizeTransport(value string) string {
	switch strings.ToLower(value) {
	case "", "local":
		return "local"
	case "remote":
		return "remote"
	default:
		return value
	}
}

func validateAdapterPolicy(manifest Manifest, metadata AdapterMetadata) error {
	if normalizeTransport(metadata.Transport) == "remote" && !manifest.Permissions.Network {
		return NewError(ErrPolicyViolation, "remote adapter requires permissions.network=true")
	}
	return nil
}
```

Wire it into `Run(...)` immediately after capability preflight:

```go
	metadata.Transport = normalizeTransport(metadata.Transport)
	trace.AddAdapterEvent("selection", "ok", map[string]string{"skill": manifest.Name}, metadata, nil)
	if err := validateAdapter(manifest, metadata); err != nil {
		trace.AddAdapterEvent("capability_check", "failed", manifest.Capabilities.Requires, metadata, err)
		trace.SetFinal("failed", nil, err.Error())
		_ = WriteTrace(tracePath, trace)
		result.Status = "failed"
		return result, err
	}
	trace.AddAdapterEvent("capability_check", "ok", manifest.Capabilities.Requires, metadata.Capabilities, nil)
	if err := validateAdapterPolicy(manifest, metadata); err != nil {
		trace.AddAdapterEvent("policy_check", "failed", map[string]any{
			"transport": metadata.Transport,
			"network":   manifest.Permissions.Network,
		}, metadata, err)
		trace.SetFinal("failed", nil, err.Error())
		_ = WriteTrace(tracePath, trace)
		result.Status = "failed"
		return result, err
	}
	trace.AddAdapterEvent("policy_check", "ok", map[string]any{
		"transport": metadata.Transport,
		"network":   manifest.Permissions.Network,
	}, map[string]string{"status": "allowed"}, nil)
```

Set explicit transport in builtin adapters:

```go
func (FakeFixTestFailureAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "fake-scripted",
		ModelProfile:    fakeModelProfile,
		Transport:       "local",
		SupportedSkills: []string{"repo.fix_test_failure", "repo.analyze_test_failures", "repo.apply_small_refactor"},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 32000,
			ReasoningLevel:   "medium",
		},
	}
}

func (HeuristicAnalyzeTestFailuresAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "heuristic-analyze",
		ModelProfile:    "heuristic-analyze",
		Transport:       "local",
		SupportedSkills: []string{"repo.analyze_test_failures"},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 32000,
			ReasoningLevel:   "medium",
		},
	}
}
```

Add the test helper adapter near the bottom of `internal/agenix/runtime_integration_test.go`:

```go
type remoteMetadataOnlyAdapter struct {
	metadata AdapterMetadata
	called   *bool
}

func (a remoteMetadataOnlyAdapter) Metadata() AdapterMetadata { return a.metadata }

func (a remoteMetadataOnlyAdapter) Execute(_ Manifest, _ *Tools) (map[string]any, error) {
	if a.called != nil {
		*a.called = true
	}
	return nil, NewError(ErrDriverError, "stub remote adapter")
}
```

- [ ] **Step 4: Run the runtime tests to verify they pass**

Run:

```bash
go test ./internal/agenix -run 'TestRuntimeRejectsRemoteAdapterWhenManifestDisablesNetwork|TestRuntimeRecordsSuccessfulRemotePolicyCheckBeforeExecute' -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agenix/runtime.go internal/agenix/adapter_builtin.go internal/agenix/runtime_integration_test.go
git commit -m "feat: add remote adapter policy preflight"
```

### Task 2: Add the OpenAI provider client with offline tests

**Files:**
- Create: `internal/agenix/provider_openai.go`
- Create: `internal/agenix/provider_openai_test.go`
- Test: `internal/agenix/provider_openai_test.go`

- [ ] **Step 1: Write the failing provider tests**

Create `internal/agenix/provider_openai_test.go` with:

```go
package agenix

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIClientAnalyzeReturnsStructuredOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("authorization = %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "resp_123",
			"output": [{
				"content": [{
					"type": "output_text",
					"text": "{\"analysis_summary\":\"fixture fails\",\"failing_tests\":[\"test_mathlib.py::test_adds_numbers\"],\"likely_root_cause\":\"mathlib.add subtracts instead of adding\",\"changed_files\":[]}"
				}]
			}]
		}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4-mini",
	}
	result, err := client.Analyze(OpenAIAnalyzeRequest{Skill: "repo.analyze_test_failures.remote", Context: "fixture context"})
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}
	if result.AnalysisSummary == "" || len(result.FailingTests) != 1 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestOpenAIClientAnalyzeFailsWithoutAPIKey(t *testing.T) {
	client := OpenAIAnalyzeClient{BaseURL: "http://example.invalid", Model: "gpt-5.4-mini"}
	_, err := client.Analyze(OpenAIAnalyzeRequest{Skill: "repo.analyze_test_failures.remote", Context: "fixture context"})
	if err == nil {
		t.Fatal("expected missing key error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
}

func TestOpenAIClientAnalyzeRejectsMalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"output":[{"content":[{"type":"output_text","text":"not-json"}]}]}`))
	}))
	defer server.Close()

	client := OpenAIAnalyzeClient{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "gpt-5.4-mini",
	}
	_, err := client.Analyze(OpenAIAnalyzeRequest{Skill: "repo.analyze_test_failures.remote", Context: "fixture context"})
	if err == nil {
		t.Fatal("expected malformed response error")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
}
```

- [ ] **Step 2: Run the provider tests to verify they fail**

Run:

```bash
go test ./internal/agenix -run 'TestOpenAIClientAnalyzeReturnsStructuredOutput|TestOpenAIClientAnalyzeFailsWithoutAPIKey|TestOpenAIClientAnalyzeRejectsMalformedResponse' -count=1
```

Expected: FAIL with missing `OpenAIAnalyzeClient`, missing request/response types, or missing `Analyze(...)`.

- [ ] **Step 3: Write the minimal provider client**

Create `internal/agenix/provider_openai.go` with:

```go
package agenix

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
)

type OpenAIAnalyzeRequest struct {
	Skill   string
	Context string
}

type OpenAIAnalyzeResult struct {
	AnalysisSummary string   `json:"analysis_summary"`
	FailingTests    []string `json:"failing_tests"`
	LikelyRootCause string   `json:"likely_root_cause"`
	ChangedFiles    []string `json:"changed_files"`
}

type OpenAIAnalyzeClient struct {
	BaseURL string
	APIKey  string
	Model   string
	Client  *http.Client
}

func (c OpenAIAnalyzeClient) Analyze(request OpenAIAnalyzeRequest) (OpenAIAnalyzeResult, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return OpenAIAnalyzeResult{}, NewError(ErrDriverError, "openai api key is not configured")
	}
	baseURL := strings.TrimRight(c.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	body := map[string]any{
		"model": c.Model,
		"input": "Return JSON with keys analysis_summary, failing_tests, likely_root_cause, changed_files.\n\nSkill: " + request.Skill + "\n\nContext:\n" + request.Context,
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "encode openai request", err)
	}
	httpClient := c.Client
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/responses", bytes.NewReader(rawBody))
	if err != nil {
		return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "create openai request", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "openai request failed", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Output []struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "decode openai response", err)
	}
	for _, item := range payload.Output {
		for _, content := range item.Content {
			if content.Type != "output_text" {
				continue
			}
			var result OpenAIAnalyzeResult
			if err := json.Unmarshal([]byte(content.Text), &result); err != nil {
				return OpenAIAnalyzeResult{}, WrapError(ErrDriverError, "decode openai structured output", err)
			}
			return result, nil
		}
	}
	return OpenAIAnalyzeResult{}, NewError(ErrDriverError, "openai response missing structured output")
}
```

- [ ] **Step 4: Run the provider tests to verify they pass**

Run:

```bash
go test ./internal/agenix -run 'TestOpenAIClientAnalyzeReturnsStructuredOutput|TestOpenAIClientAnalyzeFailsWithoutAPIKey|TestOpenAIClientAnalyzeRejectsMalformedResponse' -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agenix/provider_openai.go internal/agenix/provider_openai_test.go
git commit -m "feat: add openai analyze provider client"
```

### Task 3: Add the `openai-analyze` adapter and remote example manifest

**Files:**
- Modify: `internal/agenix/adapter_builtin.go`
- Modify: `internal/agenix/runtime_integration_test.go`
- Modify: `cmd/agenix/main_test.go`
- Create: `examples/repo.analyze_test_failures.remote/manifest.yaml`
- Create: `examples/repo.analyze_test_failures.remote/fixture/mathlib.py`
- Create: `examples/repo.analyze_test_failures.remote/fixture/test_mathlib.py`
- Create: `examples/repo.analyze_test_failures.remote/fixture/verify_failing.py`
- Test: `internal/agenix/runtime_integration_test.go`
- Test: `cmd/agenix/main_test.go`

- [ ] **Step 1: Write the failing integration tests**

Add to `internal/agenix/runtime_integration_test.go`:

```go
func TestRuntimeRunsRemoteAnalyzeSkillWithStubProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [{
				"content": [{
					"type": "output_text",
					"text": "{\"analysis_summary\":\"fixture fails\",\"failing_tests\":[\"test_mathlib.py::test_adds_numbers\"],\"likely_root_cause\":\"mathlib.add subtracts instead of adding\",\"changed_files\":[]}"
				}]
			}]
		}`))
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("AGENIX_OPENAI_BASE_URL", server.URL)

	manifestPath := filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote", "manifest.yaml")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")
	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter:      mustResolveAdapter(t, "openai-analyze"),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != "passed" {
		t.Fatalf("status = %q", result.Status)
	}
	trace, readErr := ReadTrace(result.TracePath)
	if readErr != nil {
		t.Fatalf("ReadTrace returned error: %v", readErr)
	}
	if !traceHasAdapterEvent(*trace, "policy_check", "ok") {
		t.Fatalf("trace does not contain adapter policy_check event: %#v", trace.Events)
	}
	if !traceHasAdapterRequestField(*trace, "execute", "provider") {
		t.Fatalf("trace does not contain provider metadata: %#v", trace.Events)
	}
}

func TestRuntimeRemoteAnalyzeFailsWithoutAPIKey(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	manifestPath := filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote", "manifest.yaml")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")

	_, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter:      mustResolveAdapter(t, "openai-analyze"),
	})
	if err == nil {
		t.Fatal("expected missing key failure")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
}
```

Add to `cmd/agenix/main_test.go`:

```go
func TestCLIRunRemoteAnalyzeArtifactWithStubProvider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [{
				"content": [{
					"type": "output_text",
					"text": "{\"analysis_summary\":\"fixture fails\",\"failing_tests\":[\"test_mathlib.py::test_adds_numbers\"],\"likely_root_cause\":\"mathlib.add subtracts instead of adding\",\"changed_files\":[]}"
				}]
			}]
		}`))
	}))
	defer server.Close()

	root := t.TempDir()
	skillDir := filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote")
	artifact := filepath.Join(root, "analyze-remote.agenix")
	buildOut, err := exec.Command("go", "run", ".", "build", skillDir, "-o", artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, buildOut)
	}

	cmd := exec.Command("go", "run", ".", "run", artifact, "--adapter", "openai-analyze")
	cmd.Env = append(os.Environ(), "OPENAI_API_KEY=test-key", "AGENIX_OPENAI_BASE_URL="+server.URL)
	runOut, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run remote artifact failed: %v\n%s", err, runOut)
	}
	if !strings.Contains(string(runOut), "status=passed") {
		t.Fatalf("unexpected run output: %s", runOut)
	}
}
```

- [ ] **Step 2: Run the integration tests to verify they fail**

Run:

```bash
go test ./internal/agenix -run 'TestRuntimeRunsRemoteAnalyzeSkillWithStubProvider|TestRuntimeRemoteAnalyzeFailsWithoutAPIKey' -count=1
go test ./cmd/agenix -run 'TestCLIRunRemoteAnalyzeArtifactWithStubProvider' -count=1
```

Expected: FAIL with missing adapter, missing example manifest, or missing trace helper.

- [ ] **Step 3: Implement the remote adapter and example**

Update `internal/agenix/adapter_builtin.go`:

```go
type OpenAIAnalyzeAdapter struct{}

func (OpenAIAnalyzeAdapter) Metadata() AdapterMetadata {
	return AdapterMetadata{
		Name:            "openai-analyze",
		ModelProfile:    "openai:gpt-5.4-mini",
		Transport:       "remote",
		SupportedSkills: []string{"repo.analyze_test_failures.remote"},
		Capabilities: CapabilitySet{
			ToolCalling:      true,
			StructuredOutput: true,
			MaxContextTokens: 128000,
			ReasoningLevel:   "medium",
		},
	}
}

func (OpenAIAnalyzeAdapter) Execute(manifest Manifest, tools *Tools) (map[string]any, error) {
	repoPath := manifest.Inputs["repo_path"]
	if _, err := tools.FSList(repoPath); err != nil {
		return nil, err
	}
	source, err := tools.FSRead(filepath.Join(repoPath, "mathlib.py"))
	if err != nil {
		return nil, err
	}
	testContent, err := tools.FSRead(filepath.Join(repoPath, "test_mathlib.py"))
	if err != nil {
		return nil, err
	}
	client := OpenAIAnalyzeClient{
		BaseURL: os.Getenv("AGENIX_OPENAI_BASE_URL"),
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-5.4-mini",
	}
	result, err := client.Analyze(OpenAIAnalyzeRequest{
		Skill: manifest.Name,
		Context: "mathlib.py:\n" + source + "\n\n" +
			"test_mathlib.py:\n" + testContent,
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"analysis_summary":  result.AnalysisSummary,
		"failing_tests":     result.FailingTests,
		"likely_root_cause": result.LikelyRootCause,
		"changed_files":     result.ChangedFiles,
	}, nil
}

func ResolveBuiltinAdapter(name string) (Adapter, error) {
	switch name {
	case "", "fake-scripted":
		return FakeFixTestFailureAdapter{}, nil
	case "heuristic-analyze":
		return HeuristicAnalyzeTestFailuresAdapter{}, nil
	case "openai-analyze":
		return OpenAIAnalyzeAdapter{}, nil
	default:
		return nil, NewError(ErrUnsupportedAdapter, "unknown adapter: "+name)
	}
}
```

Create `examples/repo.analyze_test_failures.remote/manifest.yaml`:

```yaml
apiVersion: agenix/v0.1
kind: Skill

name: repo.analyze_test_failures.remote
version: 0.1.0
description: Analyze a failing pytest suite through a remote provider-backed adapter.

capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 32000
    reasoning_level: medium

tools:
  - fs

permissions:
  network: true
  filesystem:
    read:
      - ${repo_path}
    write:
  shell:
    allow:

inputs:
  repo_path: fixture

outputs:
  required:
    - analysis_summary
    - failing_tests
    - likely_root_cause
    - changed_files

verifiers:
  - type: command
    name: fixture_still_fails
    run: ["python3", "verify_failing.py"]
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

recovery:
  strategy: checkpoint
  intervals: 5
```

Create the remote fixture files with the same failing content as the local
read-only example:

```python
# examples/repo.analyze_test_failures.remote/fixture/mathlib.py
def add(a, b):
    return a - b
```

```python
# examples/repo.analyze_test_failures.remote/fixture/test_mathlib.py
from mathlib import add


def test_adds_numbers():
    assert add(2, 3) == 5
```

```python
# examples/repo.analyze_test_failures.remote/fixture/verify_failing.py
import subprocess
import sys


def main() -> int:
    completed = subprocess.run([sys.executable, "-m", "pytest", "-q"], check=False)
    return 0 if completed.returncode != 0 else 1


if __name__ == "__main__":
    raise SystemExit(main())
```

Add the test helpers:

```go
func mustResolveAdapter(t *testing.T, name string) Adapter {
	t.Helper()
	adapter, err := ResolveBuiltinAdapter(name)
	if err != nil {
		t.Fatalf("ResolveBuiltinAdapter(%q) returned error: %v", name, err)
	}
	return adapter
}

func traceHasAdapterRequestField(trace Trace, name, field string) bool {
	for _, event := range trace.Events {
		if event.Type != "adapter" || event.Name != name {
			continue
		}
		raw, _ := json.Marshal(event.Request)
		var request map[string]any
		if err := json.Unmarshal(raw, &request); err != nil {
			return false
		}
		_, ok := request[field]
		return ok
	}
	return false
}
```

- [ ] **Step 4: Run the integration tests to verify they pass**

Run:

```bash
go test ./internal/agenix -run 'TestRuntimeRunsRemoteAnalyzeSkillWithStubProvider|TestRuntimeRemoteAnalyzeFailsWithoutAPIKey' -count=1
go test ./cmd/agenix -run 'TestCLIRunRemoteAnalyzeArtifactWithStubProvider' -count=1
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/agenix/adapter_builtin.go internal/agenix/runtime_integration_test.go cmd/agenix/main_test.go examples/repo.analyze_test_failures.remote
git commit -m "feat: add remote openai analyze adapter"
```

### Task 4: Document the remote adapter path and run full verification

**Files:**
- Modify: `README.md`
- Modify: `specs/policy.md`
- Modify: `specs/trace.md`
- Modify: `specs/capability.md`
- Modify: `docs/roadmap/2026-04-14-agenix-roadmap.md`
- Modify: `docs/team/handoffs/2026-04-17-adapter-failure-taxonomy.md`
- Test: `internal/agenix/runtime_integration_test.go`

- [ ] **Step 1: Write the failing doc/trace assertions**

Add one trace assertion test to `internal/agenix/runtime_integration_test.go`:

```go
func TestRemoteAdapterTraceRedactsSecretsAndKeepsProviderMetadata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"output": [{
				"content": [{
					"type": "output_text",
					"text": "{\"analysis_summary\":\"fixture fails\",\"failing_tests\":[\"test_mathlib.py::test_adds_numbers\"],\"likely_root_cause\":\"mathlib.add subtracts instead of adding\",\"changed_files\":[]}"
				}]
			}]
		}`))
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "super-secret-key")
	t.Setenv("AGENIX_OPENAI_BASE_URL", server.URL)

	manifestPath := filepath.Join("..", "..", "examples", "repo.analyze_test_failures.remote", "manifest.yaml")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")
	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter:      mustResolveAdapter(t, "openai-analyze"),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	raw, readErr := os.ReadFile(result.TracePath)
	if readErr != nil {
		t.Fatalf("ReadFile returned error: %v", readErr)
	}
	if strings.Contains(string(raw), "super-secret-key") {
		t.Fatalf("trace leaked api key: %s", raw)
	}
	if !strings.Contains(string(raw), "\"provider\":\"openai\"") {
		t.Fatalf("trace missing provider metadata: %s", raw)
	}
}
```

- [ ] **Step 2: Run the trace assertion to verify it fails**

Run:

```bash
go test ./internal/agenix -run 'TestRemoteAdapterTraceRedactsSecretsAndKeepsProviderMetadata' -count=1
```

Expected: FAIL because remote adapter trace metadata is missing, or because trace request payload is too thin.

- [ ] **Step 3: Update trace payload and documentation**

Extend the `executeRequest` map in `internal/agenix/runtime.go`:

```go
	executeRequest := map[string]string{
		"skill":     manifest.Name,
		"adapter":   metadata.Name,
		"transport": metadata.Transport,
	}
	if metadata.Transport == "remote" {
		executeRequest["provider"] = "openai"
		executeRequest["model"] = metadata.ModelProfile
	}
```

Document the manual smoke path in `README.md`:

```md
Run the provider-backed read-only smoke path:

```bash
OPENAI_API_KEY="$OPENAI_API_KEY" go run ./cmd/agenix run examples/repo.analyze_test_failures.remote/manifest.yaml --adapter openai-analyze
```

This path is opt-in, uses `permissions.network: true`, and is not part of the
default offline CI suite.
```

Update `specs/policy.md`:

```md
- Remote adapters are subject to manifest `permissions.network`.
- A `transport=remote` adapter selected against `permissions.network=false`
  must fail closed as `PolicyViolation` before adapter execution.
```

Update `specs/trace.md`:

```md
- Provider-backed `adapter.execute` events may record `transport`, `provider`,
  and `model` request metadata, but must not persist credentials or raw
  provider payloads.
```

Update `specs/capability.md`:

```md
- Capability preflight and remote-network policy preflight are separate stages.
- A provider-backed adapter still uses the same capability contract as local
  adapters.
```

Update `docs/roadmap/2026-04-14-agenix-roadmap.md`:

```md
- Post-v0 provider-backed adapter work now has an explicit read-only spike plan.
- The provider-backed path remains outside the v0 acceptance sweep.
```

Update `docs/team/handoffs/2026-04-17-adapter-failure-taxonomy.md`:

```md
## Next Handoff

The next agent should:

- implement the provider-backed read-only adapter spike behind the new remote
  policy boundary
- keep the v0 acceptance sweep unchanged while the remote path is still opt-in
```

- [ ] **Step 4: Run full verification**

Run:

```bash
go test ./internal/agenix -run 'TestRemoteAdapterTraceRedactsSecretsAndKeepsProviderMetadata' -count=1
go test -count=1 ./...
go vet ./...
go build ./cmd/agenix
git diff --check
```

Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add README.md specs/policy.md specs/trace.md specs/capability.md docs/roadmap/2026-04-14-agenix-roadmap.md docs/team/handoffs/2026-04-17-adapter-failure-taxonomy.md internal/agenix/runtime.go internal/agenix/runtime_integration_test.go
git commit -m "docs: document provider-backed adapter spike"
```
