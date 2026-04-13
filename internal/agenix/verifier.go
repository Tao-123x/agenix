package agenix

import (
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
	argv := shellArgs(verifier.Command)
	result, err := runCommand(argv, verifier.CWD, 2*time.Minute)
	status := "passed"
	if err != nil || result.ExitCode != verifier.Success.ExitCode {
		status = "failed"
	}
	trace.AddVerifierEvent(verifier.Name, verifier.Type, status, result.Stdout, result.Stderr, result.ExitCode)
	if status == "failed" {
		if err != nil {
			return WrapError(ErrVerificationFailed, verifier.Name, err)
		}
		return NewError(ErrVerificationFailed, verifier.Name+" failed")
	}
	return nil
}

func runSchemaVerifier(manifest Manifest, verifier Verifier, output map[string]any, trace *Trace) error {
	for _, field := range manifest.Outputs.Required {
		if _, ok := output[field]; !ok {
			trace.AddVerifierEvent(verifier.Name, verifier.Type, "failed", "", "missing output field: "+field, 1)
			return NewError(ErrVerificationFailed, "missing output field: "+field)
		}
	}
	trace.AddVerifierEvent(verifier.Name, verifier.Type, "passed", "", "", 0)
	return nil
}

func shellArgs(command string) []string {
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C", command}
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
