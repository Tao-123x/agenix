package agenix

import (
	"os/exec"
	"path"
	"runtime"
	"strings"
)

type lookPathFunc func(string) (string, error)

func normalizeCommandArgv(argv []string) []string {
	return normalizeCommandArgvForOS(runtime.GOOS, argv, exec.LookPath)
}

func normalizeCommandArgvForOS(goos string, argv []string, lookPath lookPathFunc) []string {
	if len(argv) == 0 {
		return nil
	}
	out := append([]string(nil), argv...)
	out[0] = resolveExecutableAliasForOS(goos, out[0], lookPath)
	return out
}

func normalizeShellCommand(command string) string {
	return normalizeShellCommandForOS(runtime.GOOS, command, exec.LookPath)
}

func normalizeShellCommandForOS(goos string, command string, lookPath lookPathFunc) string {
	if goos != "windows" {
		return command
	}
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return command
	}
	resolved := resolveExecutableAliasForOS(goos, fields[0], lookPath)
	if resolved == fields[0] {
		return command
	}
	start := strings.Index(command, fields[0])
	if start < 0 {
		return command
	}
	return command[:start] + resolved + command[start+len(fields[0]):]
}

func resolveExecutableAlias(name string) string {
	return resolveExecutableAliasForOS(runtime.GOOS, name, exec.LookPath)
}

func resolveExecutableAliasForOS(goos string, name string, lookPath lookPathFunc) string {
	if goos != "windows" || name != "python3" {
		return name
	}
	if python3Path, err := lookPath("python3"); err == nil && !isWindowsStoreShimPath(python3Path) {
		return name
	}
	if _, err := lookPath("python"); err == nil {
		return "python"
	}
	return name
}

func isWindowsStoreShimPath(rawPath string) bool {
	clean := path.Clean(strings.ReplaceAll(strings.ToLower(rawPath), `\`, `/`))
	base := path.Base(clean)
	return (base == "python.exe" || base == "python3.exe") &&
		strings.Contains(clean, "microsoft/windowsapps/")
}
