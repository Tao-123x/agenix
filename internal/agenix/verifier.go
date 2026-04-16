package agenix

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func RunVerifiers(manifest Manifest, output map[string]any, trace *Trace) error {
	for _, verifier := range manifest.Verifiers {
		switch verifier.Type {
		case "command":
			if err := runCommandVerifier(manifest.Permissions, verifier, trace); err != nil {
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

func runCommandVerifier(permissions Permissions, verifier Verifier, trace *Trace) error {
	requested := verifierRequestedArgs(verifier)
	launchArgv := verifierLaunchArgv(verifier)
	timeout := verifierTimeout(verifier)
	request := verifierRequest(verifier, timeout)
	if err := checkVerifierPolicy(verifier, requested, timeout); err != nil {
		trace.AddVerifierEvent(verifier.Name, verifier.Type, "failed", request, ShellResult{}, err)
		return err
	}
	result, err := runCommand(launchArgv, verifier.CWD, timeout, permissions)
	status := "passed"
	if err != nil || result.ExitCode != verifier.Success.ExitCode {
		status = "failed"
	}
	trace.AddVerifierEvent(verifier.Name, verifier.Type, status, request, result, err)
	if status == "failed" {
		if err != nil {
			if IsErrorClass(err, ErrTimeout) || IsErrorClass(err, ErrPolicyViolation) {
				return err
			}
			return WrapError(ErrVerificationFailed, verifier.Name, err)
		}
		return NewError(ErrVerificationFailed, verifier.Name+" failed")
	}
	return nil
}

func verifierRequestedArgs(verifier Verifier) []string {
	return verifierRequestedArgsForOS(runtime.GOOS, verifier, exec.LookPath)
}

func verifierLaunchArgv(verifier Verifier) []string {
	return verifierLaunchArgvForOS(runtime.GOOS, verifier, exec.LookPath)
}

func verifierLaunchArgvForOS(goos string, verifier Verifier, lookPath lookPathFunc) []string {
	if len(verifier.Run) > 0 {
		return normalizeCommandArgvForOS(goos, verifier.Run, lookPath)
	}
	return shellArgsForOS(goos, verifier.Command, lookPath)
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

func verifierRequest(verifier Verifier, timeout time.Duration) map[string]any {
	return verifierRequestForOS(runtime.GOOS, verifier, timeout, exec.LookPath)
}

func verifierRequestForOS(goos string, verifier Verifier, timeout time.Duration, lookPath lookPathFunc) map[string]any {
	requested := verifierRequestedArgsForOS(goos, verifier, lookPath)
	request := commandRequestForOS(goos, requested, verifier.CWD, timeout, lookPath)
	request["type"] = verifier.Type
	return request
}

func verifierRequestedArgsForOS(goos string, verifier Verifier, lookPath lookPathFunc) []string {
	if len(verifier.Run) > 0 {
		return append([]string(nil), verifier.Run...)
	}
	return shellArgsForOS(goos, verifier.Command, lookPath)
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
	return shellArgsForOS(runtime.GOOS, command, exec.LookPath)
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
