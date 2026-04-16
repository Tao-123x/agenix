package agenix

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const networkDeniedMarker = "AGENIX_NETWORK_DISABLED"

type commandRunner func(argv []string, cwd string, timeout time.Duration, env []string) (ShellResult, error)

var execCommandRunner commandRunner = runExecCommand

func runCommand(argv []string, cwd string, timeout time.Duration, permissions Permissions) (ShellResult, error) {
	launch, err := prepareCommandLaunch(argv, permissions)
	if err != nil {
		return ShellResult{}, err
	}
	defer launch.Cleanup()

	result, err := execCommandRunner(launch.Argv, cwd, timeout, launch.Env)
	if err != nil && networkDeniedByRuntime(result) {
		return result, NewError(ErrPolicyViolation, "network disabled: subprocess attempted network access")
	}
	return result, err
}

type commandLaunch struct {
	Argv    []string
	Env     []string
	Cleanup func()
}

func prepareCommandLaunch(argv []string, permissions Permissions) (commandLaunch, error) {
	if len(argv) == 0 {
		return commandLaunch{}, NewError(ErrInvalidInput, "empty command")
	}
	launch := commandLaunch{
		Argv:    append([]string(nil), argv...),
		Cleanup: func() {},
	}
	if permissions.Network {
		return launch, nil
	}
	return prepareNetworkDeniedLaunch(launch.Argv)
}

func prepareNetworkDeniedLaunch(argv []string) (commandLaunch, error) {
	launch := commandLaunch{
		Argv:    append([]string(nil), argv...),
		Cleanup: func() {},
	}
	if len(argv) == 0 {
		return commandLaunch{}, NewError(ErrInvalidInput, "empty command")
	}
	if isPythonExecutable(argv[0]) {
		env, cleanup, err := pythonNetworkDeniedEnv()
		if err != nil {
			return commandLaunch{}, err
		}
		launch.Env = env
		launch.Cleanup = cleanup
		return launch, nil
	}
	if isOfflineSafeGitCommand(argv) {
		return launch, nil
	}
	return commandLaunch{}, NewError(ErrPolicyViolation, "network disabled: unsupported subprocess executable: "+argv[0])
}

func isPythonExecutable(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return base == "python" || base == "python.exe" || base == "python3" || base == "python3.exe"
}

func isOfflineSafeGitCommand(argv []string) bool {
	if len(argv) < 2 {
		return false
	}
	if strings.ToLower(filepath.Base(argv[0])) != "git" {
		return false
	}
	switch argv[1] {
	case "status", "diff", "apply":
		return true
	default:
		return false
	}
}

func pythonNetworkDeniedEnv() ([]string, func(), error) {
	dir, err := os.MkdirTemp("", "agenix-network-off-*")
	if err != nil {
		return nil, nil, WrapError(ErrDriverError, "create python network policy dir", err)
	}
	content := []byte(`import socket

MARKER = "` + networkDeniedMarker + `"

def _deny(*args, **kwargs):
    raise RuntimeError(MARKER)

class AgenixDeniedSocket(socket.socket):
    def connect(self, *args, **kwargs):
        _deny()

    def connect_ex(self, *args, **kwargs):
        _deny()

socket.socket = AgenixDeniedSocket
socket.create_connection = _deny
socket.getaddrinfo = _deny
`)
	path := filepath.Join(dir, "sitecustomize.py")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, nil, WrapError(ErrDriverError, "write python network policy", err)
	}

	env := append([]string(nil), os.Environ()...)
	separator := string(os.PathListSeparator)
	found := false
	for i, entry := range env {
		if strings.HasPrefix(entry, "PYTHONPATH=") {
			current := strings.TrimPrefix(entry, "PYTHONPATH=")
			if current == "" {
				env[i] = "PYTHONPATH=" + dir
			} else {
				env[i] = "PYTHONPATH=" + dir + separator + current
			}
			found = true
			break
		}
	}
	if !found {
		env = append(env, "PYTHONPATH="+dir)
	}
	cleanup := func() {
		_ = os.RemoveAll(dir)
	}
	return env, cleanup, nil
}

func networkDeniedByRuntime(result ShellResult) bool {
	return strings.Contains(result.Stdout, networkDeniedMarker) || strings.Contains(result.Stderr, networkDeniedMarker)
}

func runExecCommand(argv []string, cwd string, timeout time.Duration, env []string) (ShellResult, error) {
	if len(argv) == 0 {
		return ShellResult{}, NewError(ErrInvalidInput, "empty command")
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	if cwd != "" {
		abs, err := filepath.Abs(cwd)
		if err != nil {
			return ShellResult{}, WrapError(ErrInvalidInput, "normalize cwd", err)
		}
		cmd.Dir = abs
	}
	if len(env) > 0 {
		cmd.Env = env
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := ShellResult{Stdout: stdout.String(), Stderr: stderr.String(), ExitCode: cmd.ProcessState.ExitCode()}
	if ctx.Err() == context.DeadlineExceeded {
		return result, NewError(ErrTimeout, "command timed out")
	}
	if err != nil {
		return result, WrapError(ErrDriverError, "command failed", err)
	}
	return result, nil
}
