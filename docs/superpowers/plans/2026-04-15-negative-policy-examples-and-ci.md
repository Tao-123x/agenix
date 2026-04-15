# Negative Policy Examples And CI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add test-only negative policy examples plus minimum GitHub Actions coverage so Agenix continuously proves that unsafe writes, shell allowlist mismatches, and verifier policy mismatches are rejected.

**Architecture:** Keep the public `examples/` directory positive and add intentionally failing policy scenarios under `internal/agenix/testdata/policy_negative`. Exercise those scenarios through integration-style Go tests so the default `go test ./...` path covers the same runtime behavior Maya cares about, then run that same test path on Linux and Windows in GitHub Actions.

**Tech Stack:** Go 1.22, Go test integration tests in `internal/agenix`, YAML manifests under testdata, GitHub Actions workflow YAML in `.github/workflows`.

---

## File Structure

**Create:**

- `internal/agenix/testdata/policy_negative/write_scope_escape/manifest.yaml`
- `internal/agenix/testdata/policy_negative/write_scope_escape/fixture/README.txt`
- `internal/agenix/testdata/policy_negative/shell_allowlist_mismatch/manifest.yaml`
- `internal/agenix/testdata/policy_negative/verifier_policy_reject/manifest.yaml`
- `internal/agenix/testdata/policy_negative/verifier_policy_reject/fixture/verify_ok.py`
- `internal/agenix/policy_negative_integration_test.go`
- `.github/workflows/policy-negative.yml`

**Modify:**

- `docs/team/handoffs/2026-04-15-verifier-policy-contract.md` only if the final handoff needs a follow-up note

**Why this structure:**

- `internal/agenix/testdata/policy_negative/...` holds intentionally failing skills without polluting the public success-path demos.
- `internal/agenix/policy_negative_integration_test.go` keeps all negative policy runtime scenarios in one focused test file instead of bloating `runtime_integration_test.go`.
- `.github/workflows/policy-negative.yml` is a minimal workflow that reuses the existing default test command rather than inventing a second CI path.

### Task 1: Add Negative Policy Testdata Scenarios

**Files:**
- Create: `internal/agenix/testdata/policy_negative/write_scope_escape/manifest.yaml`
- Create: `internal/agenix/testdata/policy_negative/write_scope_escape/fixture/README.txt`
- Create: `internal/agenix/testdata/policy_negative/shell_allowlist_mismatch/manifest.yaml`
- Create: `internal/agenix/testdata/policy_negative/verifier_policy_reject/manifest.yaml`
- Create: `internal/agenix/testdata/policy_negative/verifier_policy_reject/fixture/verify_ok.py`

- [ ] **Step 1: Create the write-scope-escape manifest and fixture**

```yaml
apiVersion: agenix/v0.1
kind: Skill
name: policy_negative.write_scope_escape
version: 0.1.0
description: Attempt a filesystem write outside declared write scope.
tools:
  - fs
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
inputs:
  repo_path: fixture
outputs:
  required:
    - patch_summary
    - changed_files
verifiers:
  - type: schema
    name: output_schema_check
    schemaRef: outputs
recovery:
  strategy: checkpoint
  intervals: 5
```

```text
policy-negative fixture root for write-scope escape tests
```

- [ ] **Step 2: Create the shell-allowlist-mismatch manifest**

```yaml
apiVersion: agenix/v0.1
kind: Skill
name: policy_negative.shell_allowlist_mismatch
version: 0.1.0
description: Attempt a shell command that is related to, but not equal to, the allowlisted argv.
tools:
  - shell
permissions:
  network: false
  filesystem:
    read:
    write:
  shell:
    allow:
      - run: ["python3", "-m", "pytest", "-q"]
outputs:
  required:
    - patch_summary
    - changed_files
verifiers:
  - type: schema
    name: output_schema_check
    schemaRef: outputs
recovery:
  strategy: checkpoint
  intervals: 5
```

- [ ] **Step 3: Create the verifier-policy-reject manifest and verifier script**

```yaml
apiVersion: agenix/v0.1
kind: Skill
name: policy_negative.verifier_policy_reject
version: 0.1.0
description: Fail verifier execution before command launch because verifier policy mismatches the requested command.
tools:
  - fs
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
inputs:
  repo_path: fixture
outputs:
  required:
    - patch_summary
    - changed_files
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "verify_ok.py"]
    cwd: ${repo_path}
    policy:
      executable: python
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

```python
print("ok")
```

- [ ] **Step 4: Verify the files exist before writing tests**

Run:

```powershell
Get-ChildItem -Recurse internal\agenix\testdata\policy_negative
```

Expected: all three scenario directories appear, and `verify_ok.py` is present under `verifier_policy_reject/fixture`.

- [ ] **Step 5: Commit the testdata fixtures**

```powershell
git add internal/agenix/testdata/policy_negative
git commit -m "test: add negative policy scenario fixtures"
```

### Task 2: Add Runtime-Level Negative Policy Integration Tests

**Files:**
- Create: `internal/agenix/policy_negative_integration_test.go`

- [ ] **Step 1: Write the failing integration test file with test adapters and helpers**

```go
package agenix

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"
)

type shellMismatchAdapter struct{}

func (shellMismatchAdapter) Execute(_ Manifest, tools *Tools) (map[string]any, error) {
	_, err := tools.ShellExec([]string{"python3", "-m", "pip", "install", "pytest"}, "", 5*time.Second)
	return map[string]any{
		"patch_summary": "attempted disallowed shell command",
		"changed_files": []string{},
	}, err
}

type staticOutputAdapter struct {
	output map[string]any
}

func (adapter staticOutputAdapter) Execute(_ Manifest, _ *Tools) (map[string]any, error) {
	return adapter.output, nil
}

func materializePolicyScenario(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("testdata", "policy_negative", name)
	dst := filepath.Join(t.TempDir(), name)
	copyDir(t, src, dst)
	return filepath.Join(dst, "manifest.yaml")
}

func readPolicyTrace(t *testing.T, path string) *Trace {
	t.Helper()
	trace, err := ReadTrace(path)
	if err != nil {
		t.Fatal(err)
	}
	return trace
}

func traceEventNamed(t *testing.T, trace *Trace, eventType, name string) TraceEvent {
	t.Helper()
	for _, event := range trace.Events {
		if event.Type == eventType && event.Name == name {
			return event
		}
	}
	t.Fatalf("missing %s event %q in trace: %#v", eventType, name, trace.Events)
	return TraceEvent{}
}

func TestPolicyNegativeWriteScopeEscape(t *testing.T) {
	manifestPath := materializePolicyScenario(t, "write_scope_escape")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")
	outsidePath := filepath.Join(t.TempDir(), "outside.txt")

	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter:      EscapeAdapter{Path: outsidePath},
	})
	if err == nil {
		t.Fatal("expected policy violation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	trace := readPolicyTrace(t, result.TracePath)
	event := traceEventNamed(t, trace, "tool_call", "fs.write")
	if eventErrorClass(event.Error) != ErrPolicyViolation {
		t.Fatalf("expected fs.write policy violation, got %#v", event)
	}
}

func TestPolicyNegativeShellAllowlistMismatch(t *testing.T) {
	manifestPath := materializePolicyScenario(t, "shell_allowlist_mismatch")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")

	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter:      shellMismatchAdapter{},
	})
	if err == nil {
		t.Fatal("expected policy violation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	trace := readPolicyTrace(t, result.TracePath)
	event := traceEventNamed(t, trace, "tool_call", "shell.exec")
	if eventErrorClass(event.Error) != ErrPolicyViolation {
		t.Fatalf("expected shell.exec policy violation, got %#v", event)
	}
}

func TestPolicyNegativeVerifierPolicyReject(t *testing.T) {
	manifestPath := materializePolicyScenario(t, "verifier_policy_reject")
	runDir := filepath.Join(t.TempDir(), ".agenix-runs")

	result, err := Run(RunOptions{
		ManifestPath: manifestPath,
		RunDir:       runDir,
		Adapter: staticOutputAdapter{output: map[string]any{
			"patch_summary": "noop",
			"changed_files": []string{},
		}},
	})
	if err == nil {
		t.Fatal("expected verifier policy violation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	trace := readPolicyTrace(t, result.TracePath)
	event := traceEventNamed(t, trace, "verifier", "run_tests")
	if event.Status != "failed" {
		t.Fatalf("expected failed verifier event, got %#v", event)
	}
	if eventErrorClass(event.Error) != ErrPolicyViolation {
		t.Fatalf("expected verifier policy violation, got %#v", event)
	}

	raw, err := json.Marshal(event.Request)
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"cmd", "resolved_cmd", "cwd", "timeout_ms"} {
		if request[key] == nil {
			t.Fatalf("missing verifier request field %q in %#v", key, request)
		}
	}
}
```

- [ ] **Step 2: Run the focused policy-negative tests and verify they fail**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run "TestPolicyNegativeWriteScopeEscape|TestPolicyNegativeShellAllowlistMismatch|TestPolicyNegativeVerifierPolicyReject" -count=1
```

Expected: `FAIL` because the new test file does not exist yet, or the first draft fails to compile until imports/helpers line up.

- [ ] **Step 3: Make the tests compile and pass without changing production logic**

Use these exact fixes if needed while keeping all behavior test-only:

```go
func (shellMismatchAdapter) Execute(_ Manifest, tools *Tools) (map[string]any, error) {
	_, err := tools.ShellExec([]string{"python3", "-m", "pip", "install", "pytest"}, "", 5*time.Second)
	return map[string]any{
		"patch_summary": "attempted disallowed shell command",
		"changed_files": []string{},
	}, err
}
```

```go
func materializePolicyScenario(t *testing.T, name string) string {
	t.Helper()
	src := filepath.Join("testdata", "policy_negative", name)
	dst := filepath.Join(t.TempDir(), name)
	copyDir(t, src, dst)
	return filepath.Join(dst, "manifest.yaml")
}
```

Do not add new production adapters in `runtime.go` for these tests. Keep the
helpers and adapters in the `_test.go` file.

- [ ] **Step 4: Re-run the focused policy-negative tests and the full internal package**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run "TestPolicyNegativeWriteScopeEscape|TestPolicyNegativeShellAllowlistMismatch|TestPolicyNegativeVerifierPolicyReject" -count=1
go test ./internal/agenix -count=1
```

Expected:

- `ok   agenix/internal/agenix`
- `ok   agenix/internal/agenix`

- [ ] **Step 5: Commit the negative policy integration tests**

```powershell
git add internal/agenix/policy_negative_integration_test.go
git commit -m "test: add negative policy integration coverage"
```

### Task 3: Add Minimum GitHub Actions Coverage

**Files:**
- Create: `.github/workflows/policy-negative.yml`

- [ ] **Step 1: Write the failing workflow file**

```yaml
name: policy-negative

on:
  push:
    branches:
      - main
      - 'codex/**'
  pull_request:

jobs:
  go-test:
    strategy:
      fail-fast: false
      matrix:
        os: [ubuntu-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - uses: actions/setup-python@v5
        with:
          python-version: '3.12'
      - name: Run Go test suite
        run: go test ./... -count=1
```

- [ ] **Step 2: Validate the workflow file locally with a structure check**

Run:

```powershell
Get-Content .github\workflows\policy-negative.yml
```

Expected: the workflow contains `ubuntu-latest`, `windows-latest`,
`actions/setup-go@v5`, `actions/setup-python@v5`, and `go test ./... -count=1`.

- [ ] **Step 3: Run the same full suite the workflow will run**

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

- [ ] **Step 4: Remove the temporary Go directory and inspect the worktree**

Run:

```powershell
if (Test-Path .tmp-go) { cmd /C rmdir /S /Q .tmp-go }
git status --short --branch
```

Expected: only the negative policy scenario fixtures, integration test, and CI
workflow remain modified or added.

- [ ] **Step 5: Commit the workflow**

```powershell
git add .github/workflows/policy-negative.yml
git commit -m "ci: add policy negative workflow"
```

### Task 4: Final Documentation And Push

**Files:**
- Modify: `docs/team/handoffs/2026-04-15-verifier-policy-contract.md` only if the final handoff needs a follow-up bullet

- [ ] **Step 1: Update the handoff note if needed**

Append only this sentence if there is no equivalent follow-up already present:

```markdown
- negative policy scenarios now live under `internal/agenix/testdata/policy_negative` and run in GitHub Actions on Linux and Windows
```

- [ ] **Step 2: Run the final branch verification**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./... -count=1
git status --short --branch
```

Expected:

- `go test ./... -count=1` passes
- worktree is clean except for intentional changes that are ready to commit

- [ ] **Step 3: Commit any handoff update**

```powershell
git add docs/team/handoffs/2026-04-15-verifier-policy-contract.md
git commit -m "docs: note negative policy coverage"
```

Skip this commit if no handoff change was needed.

- [ ] **Step 4: Push the branch**

```powershell
git push
```

Expected: remote branch `codex/negative-policy-examples` updates successfully.
