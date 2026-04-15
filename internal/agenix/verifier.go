package agenix

import (
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func RunVerifiers(manifest Manifest, output map[string]any, trace *Trace) error {
	for _, verifier := range manifest.Verifiers {
		switch verifier.Type {
		case "command":
			if err := runCommandVerifier(verifier, trace); err != nil {
				return err
			}
		case "schema":
			if err := runSchemaVerifier(manifest, verifier, output, trace); err != nil {
				return err
			}
		default:
			return NewError(ErrInvalidInput, "unknown verifier type: "+verifier.Type)
		}
	}
	return nil
}

func runCommandVerifier(verifier Verifier, trace *Trace) error {
	requested := verifierArgs(verifier)
	timeout := verifierTimeout(verifier)
	request := verifierRequest(verifier, requested, timeout)
	if err := checkVerifierPolicy(verifier, requested, timeout); err != nil {
		trace.AddVerifierEvent(verifier.Name, verifier.Type, "failed", request, ShellResult{}, err)
		return err
	}
	result, err := runCommand(requested, verifier.CWD, timeout)
	status := "passed"
	if err != nil || result.ExitCode != verifier.Success.ExitCode {
		status = "failed"
	}
	trace.AddVerifierEvent(verifier.Name, verifier.Type, status, request, result, err)
	if status == "failed" {
		if err != nil {
			if IsErrorClass(err, ErrTimeout) {
				return err
			}
			return WrapError(ErrVerificationFailed, verifier.Name, err)
		}
		return NewError(ErrVerificationFailed, verifier.Name+" failed")
	}
	return nil
}

func verifierArgs(verifier Verifier) []string {
	if len(verifier.Run) > 0 {
		return append([]string(nil), verifier.Run...)
	}
	return shellArgs(verifier.Command)
}

func runSchemaVerifier(manifest Manifest, verifier Verifier, output map[string]any, trace *Trace) error {
	for _, field := range manifest.Outputs.Required {
		if _, ok := output[field]; !ok {
			trace.AddVerifierEvent(verifier.Name, verifier.Type, "failed", map[string]string{"type": verifier.Type}, ShellResult{Stderr: "missing output field: " + field, ExitCode: 1}, nil)
			return NewError(ErrVerificationFailed, "missing output field: "+field)
		}
	}
	trace.AddVerifierEvent(verifier.Name, verifier.Type, "passed", map[string]string{"type": verifier.Type}, ShellResult{}, nil)
	return nil
}

func verifierTimeout(verifier Verifier) time.Duration {
	if verifier.Policy != nil && verifier.Policy.TimeoutMS > 0 {
		return time.Duration(verifier.Policy.TimeoutMS) * time.Millisecond
	}
	return 2 * time.Minute
}

func verifierRequest(verifier Verifier, requested []string, timeout time.Duration) map[string]any {
	return map[string]any{
		"type":         verifier.Type,
		"cmd":          append([]string(nil), requested...),
		"resolved_cmd": normalizeCommandArgv(requested),
		"cwd":          verifier.CWD,
		"timeout_ms":   int(timeout.Milliseconds()),
	}
}

func checkVerifierPolicy(verifier Verifier, requested []string, timeout time.Duration) error {
	if len(verifier.Run) == 0 {
		return nil
	}
	if verifier.Policy == nil {
		return NewError(ErrPolicyViolation, "command verifier requires policy")
	}
	if len(requested) == 0 {
		return NewError(ErrInvalidInput, "empty command")
	}
	if requested[0] != verifier.Policy.Executable {
		return NewError(ErrPolicyViolation, "verifier executable does not match policy")
	}
	if filepath.Clean(verifier.CWD) != filepath.Clean(verifier.Policy.CWD) {
		return NewError(ErrPolicyViolation, "verifier cwd does not match policy")
	}
	if timeout.Milliseconds() != int64(verifier.Policy.TimeoutMS) {
		return NewError(ErrPolicyViolation, "verifier timeout_ms does not match policy")
	}
	return nil
}

func shellArgs(command string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C", normalizeShellCommand(command)}
	}
	return []string{"sh", "-c", command}
}

func outputStrings(output map[string]any, key string) []string {
	value, ok := output[key]
	if !ok {
		return nil
	}
	switch typed := value.(type) {
	case []string:
		return typed
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, strings.TrimSpace(item.(string)))
		}
		return out
	default:
		return nil
	}
}
