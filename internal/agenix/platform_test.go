package agenix

import (
	"errors"
	"os/exec"
	"reflect"
	"testing"
)

func TestWindowsStoreShimPathRecognitionIsHostIndependent(t *testing.T) {
	cases := []string{
		`C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`,
		`C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python.exe`,
		`C:/Users/Administrator/AppData/Local/Microsoft/WindowsApps/python3.exe`,
	}
	for _, path := range cases {
		if !isWindowsStoreShimPath(path) {
			t.Fatalf("expected %q to be recognized as a Windows Store Python shim", path)
		}
	}

	if isWindowsStoreShimPath(`C:\Users\Administrator\AppData\Local\Programs\Python\Python311\python.exe`) {
		t.Fatal("did not expect a real Python install to be treated as a Windows Store shim")
	}
}

func TestResolveExecutableAliasFallsBackFromPython3ShimOnWindows(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "python3":
			return `C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`, nil
		case "python":
			return `C:\Python311\python.exe`, nil
		default:
			return "", exec.ErrNotFound
		}
	}

	got := resolveExecutableAliasForOS("windows", "python3", lookPath)
	if got != "python" {
		t.Fatalf("expected python3 shim to fall back to python, got %q", got)
	}
}

func TestResolveExecutableAliasKeepsRealPython3OnWindows(t *testing.T) {
	lookPath := func(name string) (string, error) {
		if name == "python3" {
			return `C:\Python311\python3.exe`, nil
		}
		return "", exec.ErrNotFound
	}

	got := resolveExecutableAliasForOS("windows", "python3", lookPath)
	if got != "python3" {
		t.Fatalf("expected real python3 executable to stay python3, got %q", got)
	}
}

func TestNormalizeCommandArgvForOSIsExactExceptExecutableAlias(t *testing.T) {
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

	got := normalizeCommandArgvForOS("windows", []string{"python3", "-m", "pytest", "-q"}, lookPath)
	want := []string{"python", "-m", "pytest", "-q"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected normalized argv: got %#v want %#v", got, want)
	}

	posix := normalizeCommandArgvForOS("linux", []string{"python3", "-m", "pytest", "-q"}, lookPath)
	wantPosix := []string{"python3", "-m", "pytest", "-q"}
	if !reflect.DeepEqual(posix, wantPosix) {
		t.Fatalf("expected POSIX argv to remain exact: got %#v want %#v", posix, wantPosix)
	}
}

func TestNormalizeShellCommandForOSOnlyRewritesExecutable(t *testing.T) {
	lookPath := func(name string) (string, error) {
		switch name {
		case "python3":
			return `C:\Users\Administrator\AppData\Local\Microsoft\WindowsApps\python3.exe`, nil
		case "python":
			return `C:\Python311\python.exe`, nil
		default:
			return "", exec.ErrNotFound
		}
	}

	got := normalizeShellCommandForOS("windows", `python3 -m pytest -q`, lookPath)
	if got != `python -m pytest -q` {
		t.Fatalf("unexpected normalized command: %q", got)
	}

	posix := normalizeShellCommandForOS("darwin", `python3 -m pytest -q`, lookPath)
	if posix != `python3 -m pytest -q` {
		t.Fatalf("expected POSIX shell command to remain exact, got %q", posix)
	}
}
