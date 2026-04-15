package agenix

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Manifest struct {
	Path        string                 `json:"path"`
	APIVersion  string                 `json:"apiVersion"`
	Kind        string                 `json:"kind"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Tools       []string               `json:"tools"`
	Permissions Permissions            `json:"permissions"`
	Inputs      map[string]string      `json:"inputs"`
	Outputs     OutputSchema           `json:"outputs"`
	Verifiers   []Verifier             `json:"verifiers"`
	Redaction   RedactionConfig        `json:"redaction,omitempty"`
	Recovery    map[string]interface{} `json:"recovery,omitempty"`
}

type Permissions struct {
	Network    bool                  `json:"network"`
	Filesystem FilesystemPermissions `json:"filesystem"`
	Shell      ShellPermissions      `json:"shell"`
}

type FilesystemPermissions struct {
	Read  []string `json:"read"`
	Write []string `json:"write"`
}

type ShellPermissions struct {
	Allow []ShellCommand `json:"allow"`
}

type ShellCommand struct {
	Run []string `json:"run"`
}

type OutputSchema struct {
	Required []string `json:"required"`
}

type RedactionConfig struct {
	Keys     []string           `json:"keys,omitempty"`
	Patterns []RedactionPattern `json:"patterns,omitempty"`
}

type RedactionPattern struct {
	Name        string `json:"name"`
	Regex       string `json:"regex"`
	SecretGroup int    `json:"secret_group"`
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

type VerifierPolicy struct {
	Executable string `json:"executable,omitempty"`
	CWD        string `json:"cwd,omitempty"`
	TimeoutMS  int    `json:"timeout_ms,omitempty"`
}

type VerifierSuccess struct {
	ExitCode int `json:"exit_code"`
}

func LoadManifest(path string) (Manifest, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, WrapError(ErrNotFound, "read manifest", err)
	}
	lines := strings.Split(string(raw), "\n")
	manifest := Manifest{Path: path, Inputs: map[string]string{}}

	current := ""
	sub := ""
	var currentVerifier *Verifier
	var currentPattern *RedactionPattern
	for _, line := range lines {
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		trimmed := strings.TrimSpace(line)

		if indent == 0 && !strings.HasPrefix(trimmed, "- ") {
			key, value, ok := splitKeyValue(trimmed)
			if !ok {
				continue
			}
			current, sub = key, ""
			currentVerifier = nil
			currentPattern = nil
			switch key {
			case "apiVersion":
				manifest.APIVersion = cleanScalar(value)
			case "kind":
				manifest.Kind = cleanScalar(value)
			case "name":
				manifest.Name = cleanScalar(value)
			case "version":
				manifest.Version = cleanScalar(value)
			case "description":
				manifest.Description = cleanScalar(value)
			}
			continue
		}

		switch current {
		case "tools":
			if strings.HasPrefix(trimmed, "- ") {
				manifest.Tools = append(manifest.Tools, cleanScalar(strings.TrimPrefix(trimmed, "- ")))
			}
		case "inputs":
			if key, value, ok := splitKeyValue(trimmed); ok {
				manifest.Inputs[key] = cleanScalar(value)
			}
		case "outputs":
			if indent == 2 {
				key, _, _ := splitKeyValue(trimmed)
				sub = key
				continue
			}
			if sub == "required" && strings.HasPrefix(trimmed, "- ") {
				manifest.Outputs.Required = append(manifest.Outputs.Required, cleanScalar(strings.TrimPrefix(trimmed, "- ")))
			}
		case "permissions":
			parsePermissionsLine(trimmed, indent, &sub, &manifest.Permissions)
		case "verifiers":
			if strings.HasPrefix(trimmed, "- type:") {
				verifier := Verifier{Type: cleanScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- type:")))}
				manifest.Verifiers = append(manifest.Verifiers, verifier)
				currentVerifier = &manifest.Verifiers[len(manifest.Verifiers)-1]
				sub = ""
				continue
			}
			if currentVerifier == nil {
				continue
			}
			if indent == 4 {
				key, value, ok := splitKeyValue(trimmed)
				if !ok {
					continue
				}
				switch key {
				case "name":
					currentVerifier.Name = cleanScalar(value)
				case "cmd":
					currentVerifier.Command = cleanScalar(value)
				case "run":
					currentVerifier.Run = parseInlineArray(value)
				case "cwd":
					currentVerifier.CWD = cleanScalar(value)
				case "policy":
					currentVerifier.Policy = &VerifierPolicy{}
					sub = "verifier_policy"
				case "schemaRef":
					currentVerifier.SchemaRef = cleanScalar(value)
				case "success":
					sub = "verifier_success"
				}
				continue
			}
			if sub == "verifier_policy" && indent == 6 && currentVerifier.Policy != nil {
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
			if sub == "verifier_success" && strings.Contains(trimmed, "exit_code:") {
				_, value, _ := splitKeyValue(trimmed)
				exitCode, _ := strconv.Atoi(cleanScalar(value))
				currentVerifier.Success.ExitCode = exitCode
			}
		case "redaction":
			if indent == 2 && !strings.HasPrefix(trimmed, "- ") {
				key, _, ok := splitKeyValue(trimmed)
				if !ok {
					continue
				}
				sub = key
				currentPattern = nil
				continue
			}
			if sub == "keys" && strings.HasPrefix(trimmed, "- ") {
				manifest.Redaction.Keys = append(manifest.Redaction.Keys, cleanScalar(strings.TrimPrefix(trimmed, "- ")))
				continue
			}
			if sub == "patterns" && indent == 4 && strings.HasPrefix(trimmed, "- name:") {
				pattern := RedactionPattern{Name: cleanScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "- name:")))}
				manifest.Redaction.Patterns = append(manifest.Redaction.Patterns, pattern)
				currentPattern = &manifest.Redaction.Patterns[len(manifest.Redaction.Patterns)-1]
				continue
			}
			if sub == "patterns" && currentPattern != nil && indent == 6 {
				key, value, ok := splitKeyValue(trimmed)
				if !ok {
					continue
				}
				switch key {
				case "regex":
					currentPattern.Regex = cleanScalar(value)
				case "secret_group":
					secretGroup, _ := strconv.Atoi(cleanScalar(value))
					currentPattern.SecretGroup = secretGroup
				}
				continue
			}
		}
	}

	manifest.expandSubstitutions()
	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func parsePermissionsLine(line string, indent int, sub *string, permissions *Permissions) {
	if indent == 2 {
		key, value, ok := splitKeyValue(line)
		if !ok {
			return
		}
		*sub = key
		if key == "network" {
			permissions.Network = cleanScalar(value) == "true"
		}
		return
	}
	if indent == 4 {
		key, _, ok := splitKeyValue(line)
		if ok {
			root := *sub
			if dot := strings.Index(root, "."); dot >= 0 {
				root = root[:dot]
			}
			*sub = root + "." + key
		}
		return
	}
	if strings.HasPrefix(line, "- ") {
		value := strings.TrimPrefix(line, "- ")
		switch *sub {
		case "filesystem.read":
			permissions.Filesystem.Read = append(permissions.Filesystem.Read, cleanScalar(value))
		case "filesystem.write":
			permissions.Filesystem.Write = append(permissions.Filesystem.Write, cleanScalar(value))
		case "shell.allow":
			if strings.HasPrefix(value, "run:") {
				permissions.Shell.Allow = append(permissions.Shell.Allow, ShellCommand{Run: parseInlineArray(strings.TrimSpace(strings.TrimPrefix(value, "run:")))})
			}
		}
	}
}

func (m *Manifest) expandSubstitutions() {
	repoPath := m.Inputs["repo_path"]
	if repoPath != "" && !filepath.IsAbs(repoPath) {
		base := filepath.Dir(m.Path)
		if absBase, err := filepath.Abs(base); err == nil {
			repoPath = filepath.Clean(filepath.Join(absBase, repoPath))
			m.Inputs["repo_path"] = repoPath
		}
	}
	expand := func(value string) string {
		return strings.ReplaceAll(value, "${repo_path}", repoPath)
	}
	for i := range m.Permissions.Filesystem.Read {
		m.Permissions.Filesystem.Read[i] = expand(m.Permissions.Filesystem.Read[i])
	}
	for i := range m.Permissions.Filesystem.Write {
		m.Permissions.Filesystem.Write[i] = expand(m.Permissions.Filesystem.Write[i])
	}
	for i := range m.Verifiers {
		m.Verifiers[i].CWD = expand(m.Verifiers[i].CWD)
		m.Verifiers[i].Command = expand(m.Verifiers[i].Command)
		if m.Verifiers[i].Policy != nil {
			m.Verifiers[i].Policy.CWD = expand(m.Verifiers[i].Policy.CWD)
		}
		for j := range m.Verifiers[i].Run {
			m.Verifiers[i].Run[j] = expand(m.Verifiers[i].Run[j])
		}
	}
}

func splitKeyValue(line string) (string, string, bool) {
	idx := strings.Index(line, ":")
	if idx < 0 {
		return "", "", false
	}
	return strings.TrimSpace(line[:idx]), strings.TrimSpace(line[idx+1:]), true
}

func cleanScalar(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	value = strings.Trim(value, `'`)
	return value
}

func parseInlineArray(value string) []string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "[")
	value = strings.TrimSuffix(value, "]")
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, cleanScalar(part))
	}
	return out
}
