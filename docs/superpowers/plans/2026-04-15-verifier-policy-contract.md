# Verifier Policy Contract Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the minimal P0 verifier policy contract for `command` verifiers by enforcing and tracing `executable`, `cwd`, and `timeout_ms` for `run: [...]` verifiers.

**Architecture:** Keep the current runtime shape and add a narrow verifier-specific contract instead of redesigning policy globally. Manifest parsing and validation should reject malformed `run` verifier policy upfront, while verifier execution should re-check the contract at runtime, record requested and resolved command context in trace, and preserve exact-before-resolution semantics that already exist for `shell.exec`.

**Tech Stack:** Go 1.26, ad-hoc YAML manifest parsing in `internal/agenix/manifest.go`, runtime validation in `internal/agenix/schema.go`, verifier execution in `internal/agenix/verifier.go`, JSON trace files in `internal/agenix/trace.go`, Go test suite via `go test`.

---

Implementation note: the current worktree already has an unrelated dirty fixture file at `examples/repo.fix_test_failure/fixture/mathlib.py` from a previous demo run. Do not revert or edit that file as part of this feature.

### Task 1: Parse And Validate Verifier Policy In Manifests

**Files:**
- Modify: `internal/agenix/manifest.go`
- Modify: `internal/agenix/schema.go`
- Test: `internal/agenix/manifest_test.go`
- Test: `internal/agenix/schema_test.go`

- [ ] **Step 1: Write the failing manifest parsing and validation tests**

```go
func TestLoadManifestParsesStructuredCommandVerifierPolicy(t *testing.T) {
	dir := t.TempDir()
	repo := filepath.Join(dir, "repo")
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
inputs:
  repo_path: ` + repo + `
outputs:
  required:
    - patch_summary
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: ${repo_path}
    policy:
      executable: python3
      cwd: ${repo_path}
      timeout_ms: 120000
    success:
      exit_code: 0
`
	if err := os.WriteFile(manifestPath, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("LoadManifest returned error: %v", err)
	}
	if got.Verifiers[0].Policy == nil {
		t.Fatal("expected verifier policy to be parsed")
	}
	if got.Verifiers[0].Policy.Executable != "python3" {
		t.Fatalf("executable = %q", got.Verifiers[0].Policy.Executable)
	}
	if got.Verifiers[0].Policy.CWD != repo {
		t.Fatalf("policy cwd = %q, want %q", got.Verifiers[0].Policy.CWD, repo)
	}
	if got.Verifiers[0].Policy.TimeoutMS != 120000 {
		t.Fatalf("timeout_ms = %d", got.Verifiers[0].Policy.TimeoutMS)
	}
}

func TestLoadManifestRejectsRunVerifierWithoutPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
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
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: fixture
    success:
      exit_code: 0
`
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
}

func TestLoadManifestRejectsRunVerifierWithNonPositivePolicyTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "manifest.yaml")
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
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: fixture
    policy:
      executable: python3
      cwd: fixture
      timeout_ms: 0
    success:
      exit_code: 0
`
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
}
```

- [ ] **Step 2: Run the focused manifest tests and verify they fail**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); go test ./internal/agenix -run "TestLoadManifestParsesStructuredCommandVerifierPolicy|TestLoadManifestRejectsRunVerifierWithoutPolicy|TestLoadManifestRejectsRunVerifierWithNonPositivePolicyTimeout" -count=1
```

Expected: `FAIL` because `Verifier` has no `Policy` field yet and validation does not enforce the new contract.

- [ ] **Step 3: Implement minimal manifest parsing and validation**

```go
type VerifierPolicy struct {
	Executable string `json:"executable,omitempty"`
	CWD        string `json:"cwd,omitempty"`
	TimeoutMS  int    `json:"timeout_ms,omitempty"`
}

type Verifier struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Command   string          `json:"cmd,omitempty"`
	Run       []string        `json:"run,omitempty"`
	CWD       string          `json:"cwd,omitempty"`
	Policy    *VerifierPolicy `json:"policy,omitempty"`
	SchemaRef string          `json:"schemaRef,omitempty"`
	Success   VerifierSuccess `json:"success,omitempty"`
}
```

```go
case "policy":
	currentVerifier.Policy = &VerifierPolicy{}
	sub = "verifier_policy"
	continue
```

```go
if sub == "verifier_policy" {
	key, value, ok := splitKeyValue(trimmed)
	if !ok {
		continue
	}
	switch key {
	case "executable":
		currentVerifier.Policy.Executable = cleanScalar(value)
	case "cwd":
		currentVerifier.Policy.CWD = cleanScalar(value)
	case "timeout_ms":
		timeoutMS, _ := strconv.Atoi(cleanScalar(value))
		currentVerifier.Policy.TimeoutMS = timeoutMS
	}
	continue
}
```

```go
if verifier.Type == "command" && len(verifier.Run) > 0 {
	if verifier.Policy == nil {
		return missingField("manifest", fmt.Sprintf("verifiers[%d].policy", i))
	}
	if verifier.Policy.Executable == "" {
		return missingField("manifest", fmt.Sprintf("verifiers[%d].policy.executable", i))
	}
	if verifier.Policy.Executable != verifier.Run[0] {
		return NewError(ErrInvalidInput, "manifest verifier policy executable must match run[0]")
	}
	if verifier.Policy.CWD == "" {
		return missingField("manifest", fmt.Sprintf("verifiers[%d].policy.cwd", i))
	}
	if verifier.Policy.CWD != verifier.CWD {
		return NewError(ErrInvalidInput, "manifest verifier policy cwd must match verifier cwd")
	}
	if verifier.Policy.TimeoutMS <= 0 {
		return NewError(ErrInvalidInput, "manifest verifier policy timeout_ms must be greater than zero")
	}
}
```

Also update `expandSubstitutions()` so `Policy.CWD` expands `${repo_path}` the same way `Verifier.CWD` already does.

- [ ] **Step 4: Re-run the focused manifest tests and verify they pass**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); go test ./internal/agenix -run "TestLoadManifestParsesStructuredCommandVerifierPolicy|TestLoadManifestRejectsRunVerifierWithoutPolicy|TestLoadManifestRejectsRunVerifierWithNonPositivePolicyTimeout" -count=1
```

Expected: `ok  	./internal/agenix`

- [ ] **Step 5: Commit the manifest contract changes**

```powershell
git add internal/agenix/manifest.go internal/agenix/schema.go internal/agenix/manifest_test.go internal/agenix/schema_test.go
git commit -m "feat: validate verifier policy manifests"
```

### Task 2: Enforce Verifier Policy At Runtime And Trace It

**Files:**
- Modify: `internal/agenix/verifier.go`
- Modify: `internal/agenix/trace.go`
- Test: `internal/agenix/verifier_test.go`
- Test: `internal/agenix/trace_test.go`

- [ ] **Step 1: Write the failing runtime and trace tests**

```go
func TestCommandVerifierRejectsExecutablePolicyMismatch(t *testing.T) {
	repo := t.TempDir()
	verifier := Verifier{
		Type: "command",
		Name: "run_tests",
		Run:  []string{"python3", "-c", "print(42)"},
		CWD:  repo,
		Policy: &VerifierPolicy{
			Executable: "python",
			CWD:        repo,
			TimeoutMS:  1000,
		},
		Success: VerifierSuccess{ExitCode: 0},
	}
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{})

	err := runCommandVerifier(verifier, trace)
	if err == nil {
		t.Fatal("expected PolicyViolation error")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
	if trace.Events[0].Error == nil {
		t.Fatalf("expected verifier trace error payload: %#v", trace.Events[0])
	}
}

func TestCommandVerifierUsesPolicyTimeout(t *testing.T) {
	repo := t.TempDir()
	verifier := Verifier{
		Type: "command",
		Name: "run_tests",
		Run:  []string{"python3", "-c", "import time; time.sleep(0.2)"},
		CWD:  repo,
		Policy: &VerifierPolicy{
			Executable: "python3",
			CWD:        repo,
			TimeoutMS:  10,
		},
		Success: VerifierSuccess{ExitCode: 0},
	}

	err := runCommandVerifier(verifier, NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{}))
	if err == nil {
		t.Fatal("expected Timeout error")
	}
	if !IsErrorClass(err, ErrTimeout) {
		t.Fatalf("expected Timeout, got %v", err)
	}
}

func TestCommandVerifierTraceRecordsRequestedAndResolvedCommand(t *testing.T) {
	repo := t.TempDir()
	verifier := Verifier{
		Type: "command",
		Name: "run_tests",
		Run:  []string{"python3", "-c", "print(42)"},
		CWD:  repo,
		Policy: &VerifierPolicy{
			Executable: "python3",
			CWD:        repo,
			TimeoutMS:  1000,
		},
		Success: VerifierSuccess{ExitCode: 0},
	}
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{})

	if err := runCommandVerifier(verifier, trace); err != nil {
		t.Fatalf("runCommandVerifier returned error: %v", err)
	}
	request := trace.Events[0].Request.(map[string]any)
	if request["cwd"] != repo {
		t.Fatalf("request cwd = %#v", request)
	}
	if request["timeout_ms"] != float64(1000) && request["timeout_ms"] != int64(1000) {
		t.Fatalf("request timeout = %#v", request)
	}
}
```

- [ ] **Step 2: Run the focused verifier tests and verify they fail**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); go test ./internal/agenix -run "TestCommandVerifierRejectsExecutablePolicyMismatch|TestCommandVerifierUsesPolicyTimeout|TestCommandVerifierTraceRecordsRequestedAndResolvedCommand" -count=1
```

Expected: `FAIL` because verifier runtime does not yet enforce `Policy`, trace verifier events do not carry request/error details, and timeouts are still wrapped as `VerificationFailed`.

- [ ] **Step 3: Implement minimal runtime enforcement and trace request shape**

```go
func runCommandVerifier(verifier Verifier, trace *Trace) error {
	requested := verifierArgs(verifier)
	timeout := time.Duration(verifier.Policy.TimeoutMS) * time.Millisecond
	resolved := normalizeCommandArgv(requested)
	request := map[string]any{
		"type":       verifier.Type,
		"cmd":        requested,
		"resolved_cmd": resolved,
		"cwd":        verifier.CWD,
		"timeout_ms": verifier.Policy.TimeoutMS,
	}
	if err := checkVerifierPolicy(verifier, requested); err != nil {
		trace.AddVerifierEvent(verifier.Name, verifier.Type, "failed", request, ShellResult{}, err)
		return err
	}
	result, err := runCommand(requested, verifier.CWD, timeout)
	status := "passed"
	if err != nil || result.ExitCode != verifier.Success.ExitCode {
		status = "failed"
	}
	trace.AddVerifierEvent(verifier.Name, verifier.Type, status, request, result, err)
	if err != nil {
		return err
	}
	if result.ExitCode != verifier.Success.ExitCode {
		return NewError(ErrVerificationFailed, verifier.Name+" failed")
	}
	return nil
}
```

```go
func checkVerifierPolicy(verifier Verifier, requested []string) error {
	if verifier.Policy == nil {
		return NewError(ErrPolicyViolation, "command verifier requires policy")
	}
	if len(requested) == 0 {
		return NewError(ErrInvalidInput, "empty command")
	}
	if requested[0] != verifier.Policy.Executable {
		return NewError(ErrPolicyViolation, "verifier executable does not match policy")
	}
	if verifier.CWD != verifier.Policy.CWD {
		return NewError(ErrPolicyViolation, "verifier cwd does not match policy")
	}
	if verifier.Policy.TimeoutMS <= 0 {
		return NewError(ErrPolicyViolation, "verifier timeout_ms must be greater than zero")
	}
	return nil
}
```

```go
func (t *Trace) AddVerifierEvent(name, verifierType, status string, request interface{}, result ShellResult, err error) {
	event := TraceEvent{
		Type:     "verifier",
		Name:     name,
		Request:  request,
		Status:   status,
		Stdout:   truncate(result.Stdout),
		Stderr:   truncate(result.Stderr),
		ExitCode: result.ExitCode,
	}
	if err != nil {
		event.Error = map[string]string{"class": ErrorClass(err), "message": err.Error()}
	}
	t.Events = append(t.Events, event)
}
```

Update existing schema-verifier call sites and `trace_test.go` for the new `AddVerifierEvent` signature by passing a minimal request map and `ShellResult{}` for non-command verifiers.

- [ ] **Step 4: Re-run the focused verifier tests and the trace writer test**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); go test ./internal/agenix -run "TestCommandVerifierRejectsExecutablePolicyMismatch|TestCommandVerifierUsesPolicyTimeout|TestCommandVerifierTraceRecordsRequestedAndResolvedCommand|TestTraceWriterPersistsRequiredShape" -count=1
```

Expected: `ok  	./internal/agenix`

- [ ] **Step 5: Commit the runtime verifier policy changes**

```powershell
git add internal/agenix/verifier.go internal/agenix/trace.go internal/agenix/verifier_test.go internal/agenix/trace_test.go
git commit -m "feat: enforce verifier policy contract"
```

### Task 3: Migrate Canonical Manifests And Keep Integration Tests Green

**Files:**
- Modify: `examples/repo.fix_test_failure/manifest.yaml`
- Modify: `examples/repo.analyze_test_failures/manifest.yaml`
- Modify: `examples/repo.apply_small_refactor/manifest.yaml`
- Modify: `internal/agenix/runtime_integration_test.go`
- Test: `cmd/agenix/main_test.go`

- [ ] **Step 1: Write the failing integration expectation for policy-bearing verifiers**

```go
func writeManifestAt(t *testing.T, path, repo string) {
	t.Helper()
	content := `apiVersion: agenix/v0.1
kind: Skill
name: repo.fix_test_failure
version: 0.1.0
description: Fix a failing pytest suite.
tools:
  - fs
  - shell
  - git
permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
  shell:
    allow:
      - run: ["python3", "-m", "pytest", "-q"]
      - run: ["git", "status", "--short"]
      - run: ["git", "diff", "--", "."]
inputs:
  repo_path: ` + repo + `
outputs:
  required:
    - patch_summary
    - changed_files
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
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
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
```

Also add one assertion in `TestRuntimeRunsCanonicalFixTestFailureSkill` that the verifier trace request includes `timeout_ms`.

- [ ] **Step 2: Run the canonical runtime and CLI tests and verify they fail or expose missing manifest updates**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); go test ./internal/agenix -run "TestRuntimeRunsCanonicalFixTestFailureSkill|TestRuntimeRunsReadOnlyAnalyzeTestFailuresSkill|TestRuntimeRunsSmallRefactorSkillWithConstrainedWrite" -count=1
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); go test ./cmd/agenix -count=1
```

Expected: failures until every canonical `run` verifier declares a matching `policy` block and the integration helper manifest is updated.

- [ ] **Step 3: Add policy blocks to canonical `run` verifiers and keep generated manifests aligned**

```yaml
verifiers:
  - type: command
    name: run_tests
    run: ["python3", "-m", "pytest", "-q"]
    cwd: ${repo_path}
    policy:
      executable: python3
      cwd: ${repo_path}
      timeout_ms: 120000
    success:
      exit_code: 0
```

Apply the same shape to:

- `examples/repo.fix_test_failure/manifest.yaml`
- `examples/repo.analyze_test_failures/manifest.yaml`
- `examples/repo.apply_small_refactor/manifest.yaml`
- the integration-test helper manifest built in `internal/agenix/runtime_integration_test.go`

- [ ] **Step 4: Re-run canonical runtime and CLI tests**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); go test ./internal/agenix -run "TestRuntimeRunsCanonicalFixTestFailureSkill|TestRuntimeRunsReadOnlyAnalyzeTestFailuresSkill|TestRuntimeRunsSmallRefactorSkillWithConstrainedWrite|TestRuntimeRunsSmallRefactorSkillWithCRLFSource" -count=1
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User'); go test ./cmd/agenix -count=1
```

Expected:

- `ok  	./internal/agenix`
- `ok  	./cmd/agenix`

- [ ] **Step 5: Commit the canonical manifest migration**

```powershell
git add examples/repo.fix_test_failure/manifest.yaml examples/repo.analyze_test_failures/manifest.yaml examples/repo.apply_small_refactor/manifest.yaml internal/agenix/runtime_integration_test.go cmd/agenix/main_test.go
git commit -m "chore: require policy on canonical verifiers"
```

### Task 4: Document The Contract And Run Full Verification

**Files:**
- Modify: `specs/skill-manifest.md`
- Modify: `specs/policy.md`
- Modify: `specs/tool-contract.md`
- Create: `docs/decisions/0003-verifier-policy-contract.md`
- Create: `docs/team/handoffs/2026-04-15-verifier-policy-contract.md`

- [ ] **Step 1: Update docs to match the implemented contract**

```markdown
- `run` command verifiers must declare `policy.executable`, `policy.cwd`, and `policy.timeout_ms`.
- Verifier policy comparison uses the requested executable before platform alias resolution.
- Verifier trace entries record `cmd`, `resolved_cmd`, `cwd`, and `timeout_ms`.
- Legacy `cmd` verifiers remain backward compatible but do not satisfy the new procurement-grade verifier policy contract.
```

Use that wording in:

- `specs/skill-manifest.md`
- `specs/policy.md`
- `specs/tool-contract.md`
- `docs/decisions/0003-verifier-policy-contract.md`
- `docs/team/handoffs/2026-04-15-verifier-policy-contract.md`

- [ ] **Step 2: Run the full suite for this slice**

Run:

```powershell
New-Item -ItemType Directory -Force .tmp-go | Out-Null
$env:GOTMPDIR=(Resolve-Path .tmp-go).Path
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./... -count=1
```

Expected: `ok` for all packages

- [ ] **Step 3: Clean the temporary Go directory and inspect the worktree**

Run:

```powershell
cmd /C rmdir /S /Q .tmp-go
git status --short --branch
```

Expected:

- `.tmp-go` is gone
- the only remaining changes are the verifier policy contract work plus the pre-existing dirty `examples/repo.fix_test_failure/fixture/mathlib.py`

- [ ] **Step 4: Commit docs and verification updates**

```powershell
git add specs/skill-manifest.md specs/policy.md specs/tool-contract.md docs/decisions/0003-verifier-policy-contract.md docs/team/handoffs/2026-04-15-verifier-policy-contract.md
git commit -m "docs: record verifier policy contract"
```

- [ ] **Step 5: Push the branch**

```powershell
git push
```

Expected: remote branch `codex/verifier-policy-contract` updates successfully.
