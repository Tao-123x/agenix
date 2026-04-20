# Windows Symlink Test Compatibility Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make symlink-dependent tests skip on Windows sessions that cannot create symlinks because the required privilege is unavailable.

**Architecture:** Keep the fix local to test code by strengthening the helper that classifies symlink setup failures as unsupported-host conditions. Verify the new classification first with focused unit tests, then rerun the existing symlink-heavy tests that currently fail during setup.

**Tech Stack:** Go, Go test, Windows syscall error classification in test code.

---

## File Structure

**Modify:**

- `internal/agenix/policy_test.go`
- `internal/agenix/artifact_test.go`

**Why this structure:**

- `policy_test.go` already owns the shared helper and most of the symlink setup tests.
- `artifact_test.go` uses the same helper shape and should align with the stronger classification logic.

### Task 1: Add Failing Helper Tests

**Files:**
- Modify: `internal/agenix/policy_test.go`

- [ ] **Step 1: Add focused tests for the helper**

Add tests that assert:

- `isSymlinkUnsupported(os.ErrPermission)` returns true
- `isSymlinkUnsupported(windows privilege error)` returns true
- `isSymlinkUnsupported(os.ErrNotExist)` returns false

- [ ] **Step 2: Run the focused helper tests and verify the Windows privilege case fails**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run TestIsSymlinkUnsupported -count=1
```

Expected: the Windows privilege subtest fails before the helper change.

### Task 2: Implement the Minimal Helper Change

**Files:**
- Modify: `internal/agenix/policy_test.go`
- Modify: `internal/agenix/artifact_test.go`

- [ ] **Step 1: Extend the helper**

Update `isSymlinkUnsupported(err)` so it:

- returns true for `os.IsPermission(err)`
- returns true on Windows when the error unwraps to `syscall.Errno(1314)`
- returns false otherwise

- [ ] **Step 2: Re-run focused helper tests**

Run:

```powershell
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run TestIsSymlinkUnsupported -count=1
```

Expected:

- `ok   agenix/internal/agenix`

### Task 3: Verify Existing Failing Tests Now Skip Cleanly

**Files:**
- Verify: `internal/agenix/policy_test.go`
- Verify: `internal/agenix/artifact_test.go`

- [ ] **Step 1: Run the previously failing symlink tests**

Run:

```powershell
New-Item -ItemType Directory -Force .tmp-go | Out-Null
$env:GOTMPDIR=(Resolve-Path .tmp-go).Path
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -run 'TestMaterializeArtifactRejectsPreexistingWorkspaceSymlinkEscape|TestPolicyRejectsReadAndWriteThroughScopedSymlink|TestToolsFSWriteRejectsScopedSymlinkEscapeWithoutWritingOutside' -count=1
if (Test-Path .tmp-go) { cmd /C rmdir /S /Q .tmp-go }
```

Expected: package passes, with symlink tests skipped on this host instead of failing.

- [ ] **Step 2: Run broader package verification**

Run:

```powershell
New-Item -ItemType Directory -Force .tmp-go | Out-Null
$env:GOTMPDIR=(Resolve-Path .tmp-go).Path
$env:Path='C:\Program Files\Go\bin;' + [System.Environment]::GetEnvironmentVariable('Path','Machine') + ';' + [System.Environment]::GetEnvironmentVariable('Path','User')
go test ./internal/agenix -count=1
if (Test-Path .tmp-go) { cmd /C rmdir /S /Q .tmp-go }
```

Expected:

- `ok   agenix/internal/agenix`

- [ ] **Step 3: Commit**

```powershell
git add docs/superpowers/specs/2026-04-18-windows-symlink-test-compat-design.md docs/superpowers/plans/2026-04-18-windows-symlink-test-compat.md internal/agenix/policy_test.go internal/agenix/artifact_test.go
git commit -m "test: skip symlink cases without Windows privilege"
```
