package agenix

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestCommandRequestForOSRecordsRequestedAndResolvedCommands(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "python3":
			return `C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`, nil
		case "python":
			return `C:\Python311\python.exe`, nil
		default:
			return "", errors.New("not found")
		}
	}

	requested := []string{"python3", "-m", "pytest", "-q"}
	timeout := 1500 * time.Millisecond

	windows := commandRequestForOS("windows", requested, "fixture", timeout, lookPath)
	if !reflect.DeepEqual(windows["cmd"], requested) {
		t.Fatalf("windows cmd = %#v want %#v", windows["cmd"], requested)
	}
	wantWindowsResolved := []string{"python", "-m", "pytest", "-q"}
	if !reflect.DeepEqual(windows["resolved_cmd"], wantWindowsResolved) {
		t.Fatalf("windows resolved_cmd = %#v want %#v", windows["resolved_cmd"], wantWindowsResolved)
	}
	if windows["cwd"] != "fixture" || windows["timeout_ms"] != int(timeout.Milliseconds()) {
		t.Fatalf("unexpected windows request payload: %#v", windows)
	}

	linux := commandRequestForOS("linux", requested, "fixture", timeout, lookPath)
	if !reflect.DeepEqual(linux["cmd"], requested) {
		t.Fatalf("linux cmd = %#v want %#v", linux["cmd"], requested)
	}
	if !reflect.DeepEqual(linux["resolved_cmd"], requested) {
		t.Fatalf("linux resolved_cmd = %#v want %#v", linux["resolved_cmd"], requested)
	}
}

func TestShellArgsForOSUsesPlatformShellContract(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "python3":
			return `C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`, nil
		case "python":
			return `C:\Python311\python.exe`, nil
		default:
			return "", errors.New("not found")
		}
	}

	windows := shellArgsForOS("windows", "python3 -m pytest -q", lookPath)
	wantWindows := []string{"cmd", "/C", "python -m pytest -q"}
	if !reflect.DeepEqual(windows, wantWindows) {
		t.Fatalf("windows shell args = %#v want %#v", windows, wantWindows)
	}

	linux := shellArgsForOS("linux", "python3 -m pytest -q", lookPath)
	wantLinux := []string{"sh", "-c", "python3 -m pytest -q"}
	if !reflect.DeepEqual(linux, wantLinux) {
		t.Fatalf("linux shell args = %#v want %#v", linux, wantLinux)
	}
}

func TestCrossPlatformNetworkDeniedLaunchContract(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "python3":
			return `C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`, nil
		case "python":
			return `C:\Python311\python.exe`, nil
		default:
			return "", errors.New("not found")
		}
	}

	tests := []struct {
		name         string
		goos         string
		requested    []string
		wantResolved []string
		wantEnv      bool
		wantErrClass string
	}{
		{
			name:         "windows python shim resolves then launches",
			goos:         "windows",
			requested:    []string{"python3", "-m", "pytest", "-q"},
			wantResolved: []string{"python", "-m", "pytest", "-q"},
			wantEnv:      true,
		},
		{
			name:         "linux python stays exact then launches",
			goos:         "linux",
			requested:    []string{"python3", "-m", "pytest", "-q"},
			wantResolved: []string{"python3", "-m", "pytest", "-q"},
			wantEnv:      true,
		},
		{
			name:         "offline git remains allowed",
			goos:         "darwin",
			requested:    []string{"git", "status", "--short"},
			wantResolved: []string{"git", "status", "--short"},
			wantEnv:      false,
		},
		{
			name:         "unsupported executable fails closed",
			goos:         "linux",
			requested:    []string{"curl", "https://example.com"},
			wantResolved: []string{"curl", "https://example.com"},
			wantErrClass: ErrPolicyViolation,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			resolved := normalizeCommandArgvForOS(tc.goos, tc.requested, lookPath)
			if !reflect.DeepEqual(resolved, tc.wantResolved) {
				t.Fatalf("resolved = %#v want %#v", resolved, tc.wantResolved)
			}

			launch, err := prepareCommandLaunch(resolved, Permissions{Network: false})
			if tc.wantErrClass != "" {
				if err == nil {
					t.Fatalf("expected %s", tc.wantErrClass)
				}
				if !IsErrorClass(err, tc.wantErrClass) {
					t.Fatalf("expected %s, got %v", tc.wantErrClass, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("prepareCommandLaunch returned error: %v", err)
			}
			if !reflect.DeepEqual(launch.Argv, tc.wantResolved) {
				t.Fatalf("launch argv = %#v want %#v", launch.Argv, tc.wantResolved)
			}
			if gotEnv := len(launch.Env) > 0; gotEnv != tc.wantEnv {
				t.Fatalf("launch env presence = %v want %v", gotEnv, tc.wantEnv)
			}
			launch.Cleanup()
		})
	}
}

func TestOfflineSafeGitCommandTable(t *testing.T) {
	tests := []struct {
		argv []string
		want bool
	}{
		{argv: []string{"git", "status", "--short"}, want: true},
		{argv: []string{"git", "diff", "--", "."}, want: true},
		{argv: []string{"git", "apply", "patch.diff"}, want: true},
		{argv: []string{"git", "fetch", "origin"}, want: false},
		{argv: []string{"git", "clone", "https://example.com/repo.git"}, want: false},
		{argv: []string{"python3", "-m", "pytest", "-q"}, want: false},
	}

	for _, tc := range tests {
		if got := isOfflineSafeGitCommand(tc.argv); got != tc.want {
			t.Fatalf("isOfflineSafeGitCommand(%#v) = %v want %v", tc.argv, got, tc.want)
		}
	}
}

func TestVerifierRequestForOSRecordsRequestedAndResolvedCommands(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "python3":
			return `C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`, nil
		case "python":
			return `C:\Python311\python.exe`, nil
		default:
			return "", errors.New("not found")
		}
	}

	verifier := Verifier{
		Type: "command",
		Name: "run_tests",
		Run:  []string{"python3", "-m", "pytest", "-q"},
		CWD:  "fixture",
		Policy: &VerifierPolicy{
			Executable: "python3",
			CWD:        "fixture",
			TimeoutMS:  120000,
		},
	}

	windows := verifierRequestForOS("windows", verifier, 1500*time.Millisecond, lookPath)
	wantCmd := []string{"python3", "-m", "pytest", "-q"}
	wantResolved := []string{"python", "-m", "pytest", "-q"}
	if !reflect.DeepEqual(windows["cmd"], wantCmd) {
		t.Fatalf("windows cmd = %#v want %#v", windows["cmd"], wantCmd)
	}
	if !reflect.DeepEqual(windows["resolved_cmd"], wantResolved) {
		t.Fatalf("windows resolved_cmd = %#v want %#v", windows["resolved_cmd"], wantResolved)
	}

	linux := verifierRequestForOS("linux", verifier, 1500*time.Millisecond, lookPath)
	if !reflect.DeepEqual(linux["cmd"], wantCmd) || !reflect.DeepEqual(linux["resolved_cmd"], wantCmd) {
		t.Fatalf("linux verifier request = %#v", linux)
	}
}

func TestVerifierLaunchArgvForOSUsesResolvedExecutable(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "python3":
			return `C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`, nil
		case "python":
			return `C:\Python311\python.exe`, nil
		default:
			return "", errors.New("not found")
		}
	}

	verifier := Verifier{
		Type: "command",
		Name: "run_tests",
		Run:  []string{"python3", "-m", "pytest", "-q"},
		CWD:  "fixture",
	}

	got := verifierLaunchArgvForOS("windows", verifier, lookPath)
	want := []string{"python", "-m", "pytest", "-q"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("windows launch argv = %#v want %#v", got, want)
	}

	posix := verifierLaunchArgvForOS("linux", verifier, lookPath)
	wantPosix := []string{"python3", "-m", "pytest", "-q"}
	if !reflect.DeepEqual(posix, wantPosix) {
		t.Fatalf("linux launch argv = %#v want %#v", posix, wantPosix)
	}
}

func TestLegacyCommandVerifierRequestForOSUsesPlatformShellWrapper(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "python3":
			return `C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`, nil
		case "python":
			return `C:\Python311\python.exe`, nil
		default:
			return "", errors.New("not found")
		}
	}

	verifier := Verifier{
		Type:    "command",
		Name:    "run_tests",
		Command: "python3 -m pytest -q",
		CWD:     "fixture",
	}

	windows := verifierRequestForOS("windows", verifier, 1500*time.Millisecond, lookPath)
	wantWindows := []string{"cmd", "/C", "python -m pytest -q"}
	if !reflect.DeepEqual(windows["cmd"], wantWindows) || !reflect.DeepEqual(windows["resolved_cmd"], wantWindows) {
		t.Fatalf("windows legacy verifier request = %#v", windows)
	}

	linux := verifierRequestForOS("linux", verifier, 1500*time.Millisecond, lookPath)
	wantLinux := []string{"sh", "-c", "python3 -m pytest -q"}
	if !reflect.DeepEqual(linux["cmd"], wantLinux) || !reflect.DeepEqual(linux["resolved_cmd"], wantLinux) {
		t.Fatalf("linux legacy verifier request = %#v", linux)
	}
}

func TestTraceRoundTripPreservesDistinctVerifierCmdAndResolvedCmd(t *testing.T) {
	trace := NewTrace("repo.fix_test_failure", "fake-scripted", Permissions{Network: false})
	trace.Events = append(trace.Events, TraceEvent{
		Type:   "verifier",
		Name:   "run_tests",
		Status: "failed",
		Request: map[string]any{
			"type":         "command",
			"cmd":          []string{"python3", "-m", "pytest", "-q"},
			"resolved_cmd": []string{"python", "-m", "pytest", "-q"},
			"cwd":          "fixture",
			"timeout_ms":   120000,
		},
	})
	trace.SetFinal("failed", nil, "x")

	path := t.TempDir() + "/trace.json"
	if err := WriteTrace(path, trace); err != nil {
		t.Fatal(err)
	}
	decoded, err := ReadTrace(path)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := json.Marshal(decoded.Events[0].Request)
	if err != nil {
		t.Fatal(err)
	}
	var request map[string]any
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	wantCmd := []any{"python3", "-m", "pytest", "-q"}
	wantResolved := []any{"python", "-m", "pytest", "-q"}
	if !reflect.DeepEqual(request["cmd"], wantCmd) {
		t.Fatalf("trace cmd = %#v want %#v", request["cmd"], wantCmd)
	}
	if !reflect.DeepEqual(request["resolved_cmd"], wantResolved) {
		t.Fatalf("trace resolved_cmd = %#v want %#v", request["resolved_cmd"], wantResolved)
	}

	replay, err := Replay(path)
	if err != nil {
		t.Fatal(err)
	}
	raw, err = json.Marshal(replay.Events[0].Request)
	if err != nil {
		t.Fatal(err)
	}
	request = map[string]any{}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(request["cmd"], wantCmd) || !reflect.DeepEqual(request["resolved_cmd"], wantResolved) {
		t.Fatalf("replay request = %#v", request)
	}
}
