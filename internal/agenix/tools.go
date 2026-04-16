package agenix

import (
	"os"
	"time"
)

type Tools struct {
	policy *Policy
	trace  *Trace
}

type ShellResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}

func NewTools(policy *Policy, trace *Trace) *Tools {
	return &Tools{policy: policy, trace: trace}
}

func (t *Tools) FSRead(path string) (string, error) {
	start := time.Now()
	request := map[string]string{"path": path}
	if err := t.policy.CheckRead(path); err != nil {
		t.trace.AddToolEvent("fs.read", request, nil, err, time.Since(start).Milliseconds())
		return "", err
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		wrapped := WrapError(ErrDriverError, "fs.read", err)
		t.trace.AddToolEvent("fs.read", request, nil, wrapped, time.Since(start).Milliseconds())
		return "", wrapped
	}
	result := map[string]string{"content": string(raw), "encoding": "utf-8"}
	t.trace.AddToolEvent("fs.read", request, result, nil, time.Since(start).Milliseconds())
	return string(raw), nil
}

func (t *Tools) FSWrite(path, content string, overwrite bool) error {
	start := time.Now()
	request := map[string]interface{}{"path": path, "overwrite": overwrite}
	if err := t.policy.CheckWrite(path); err != nil {
		t.trace.AddToolEvent("fs.write", request, nil, err, time.Since(start).Milliseconds())
		return err
	}
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			wrapped := NewError(ErrInvalidInput, "fs.write target exists and overwrite=false")
			t.trace.AddToolEvent("fs.write", request, nil, wrapped, time.Since(start).Milliseconds())
			return wrapped
		}
	}
	if err := ensureParent(path); err != nil {
		wrapped := WrapError(ErrDriverError, "create parent directory", err)
		t.trace.AddToolEvent("fs.write", request, nil, wrapped, time.Since(start).Milliseconds())
		return wrapped
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		wrapped := WrapError(ErrDriverError, "fs.write", err)
		t.trace.AddToolEvent("fs.write", request, nil, wrapped, time.Since(start).Milliseconds())
		return wrapped
	}
	t.trace.AddToolEvent("fs.write", request, map[string]bool{"written": true}, nil, time.Since(start).Milliseconds())
	return nil
}

func (t *Tools) FSList(path string) ([]map[string]string, error) {
	start := time.Now()
	request := map[string]string{"path": path}
	if err := t.policy.CheckRead(path); err != nil {
		t.trace.AddToolEvent("fs.list", request, nil, err, time.Since(start).Milliseconds())
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		wrapped := WrapError(ErrDriverError, "fs.list", err)
		t.trace.AddToolEvent("fs.list", request, nil, wrapped, time.Since(start).Milliseconds())
		return nil, wrapped
	}
	result := make([]map[string]string, 0, len(entries))
	for _, entry := range entries {
		kind := "file"
		if entry.IsDir() {
			kind = "dir"
		}
		result = append(result, map[string]string{"name": entry.Name(), "type": kind})
	}
	t.trace.AddToolEvent("fs.list", request, result, nil, time.Since(start).Milliseconds())
	return result, nil
}

func (t *Tools) ShellExec(argv []string, cwd string, timeout time.Duration) (ShellResult, error) {
	start := time.Now()
	resolved := normalizeCommandArgv(argv)
	request := map[string]interface{}{"cmd": argv, "resolved_cmd": resolved, "cwd": cwd, "timeout_ms": timeout.Milliseconds()}
	if err := t.policy.CheckShell(argv); err != nil {
		t.trace.AddToolEvent("shell.exec", request, nil, err, time.Since(start).Milliseconds())
		return ShellResult{}, err
	}
	result, err := runCommand(resolved, cwd, timeout, t.policy.permissions)
	if err != nil {
		t.trace.AddToolEvent("shell.exec", request, result, err, time.Since(start).Milliseconds())
		return result, err
	}
	t.trace.AddToolEvent("shell.exec", request, result, nil, time.Since(start).Milliseconds())
	return result, nil
}

func (t *Tools) GitStatus(repoPath string) (ShellResult, error) {
	return t.ShellExec([]string{"git", "status", "--short"}, repoPath, 30*time.Second)
}

func (t *Tools) GitDiff(repoPath string) (ShellResult, error) {
	return t.ShellExec([]string{"git", "diff", "--", "."}, repoPath, 30*time.Second)
}
