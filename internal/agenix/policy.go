package agenix

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
)

type Policy struct {
	permissions Permissions
	readScopes  []string
	writeScopes []string
	baseDir     string
}

func NewPolicy(permissions Permissions) (*Policy, error) {
	return NewPolicyWithBase(permissions, "")
}

func NewPolicyWithBase(permissions Permissions, baseDir string) (*Policy, error) {
	resolvedBaseDir, err := resolvePolicyBaseDir(baseDir)
	if err != nil {
		return nil, err
	}
	policy := &Policy{permissions: permissions, baseDir: resolvedBaseDir}
	policy.readScopes, err = normalizeScopes(permissions.Filesystem.Read, policy.baseDir)
	if err != nil {
		return nil, err
	}
	policy.writeScopes, err = normalizeScopes(permissions.Filesystem.Write, policy.baseDir)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func (p *Policy) CheckRead(path string) error {
	if !pathInScopes(path, p.readScopes, p.baseDir) {
		return NewError(ErrPolicyViolation, "read outside declared scope: "+path)
	}
	return nil
}

func (p *Policy) CheckWrite(path string) error {
	if !pathInScopes(path, p.writeScopes, p.baseDir) {
		return NewError(ErrPolicyViolation, "write outside declared scope: "+path)
	}
	return nil
}

func (p *Policy) CheckShell(argv []string) error {
	for _, allowed := range p.permissions.Shell.Allow {
		if reflect.DeepEqual(argv, allowed.Run) {
			return nil
		}
	}
	return NewError(ErrPolicyViolation, "shell command is not allowlisted: "+strings.Join(argv, " "))
}

func normalizeScopes(scopes []string, baseDir string) ([]string, error) {
	normalized := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		resolved, err := resolvePolicyPathWithBase(scope, baseDir)
		if err != nil {
			return nil, WrapError(ErrInvalidInput, "normalize scope", err)
		}
		normalized = append(normalized, resolved)
	}
	return normalized, nil
}

func pathInScopes(path string, scopes []string, baseDir string) bool {
	resolved, err := resolvePolicyPathWithBase(path, baseDir)
	if err != nil {
		return false
	}
	for _, scope := range scopes {
		if resolved == scope {
			return true
		}
		rel, err := filepath.Rel(scope, resolved)
		if err != nil {
			continue
		}
		if rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel)) {
			return true
		}
	}
	return false
}

func ensureParent(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func resolvePolicyPath(path string) (string, error) {
	return resolvePolicyPathWithBase(path, "")
}

func resolvePolicyBaseDir(baseDir string) (string, error) {
	if baseDir == "" {
		return os.Getwd()
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return "", WrapError(ErrInvalidInput, "normalize base dir", err)
	}
	return filepath.Clean(abs), nil
}

func resolvePolicyPathWithBase(path, baseDir string) (string, error) {
	abs, err := absolutizePolicyPath(path, baseDir)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	missing := make([]string, 0, 4)
	current := clean
	for {
		resolved, err := filepath.EvalSymlinks(current)
		if err == nil {
			resolved = filepath.Clean(resolved)
			for i := len(missing) - 1; i >= 0; i-- {
				resolved = filepath.Join(resolved, missing[i])
			}
			return resolved, nil
		}
		if !os.IsNotExist(err) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return clean, nil
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

func absolutizePolicyPath(path, baseDir string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Abs(path)
	}
	resolvedBaseDir, err := resolvePolicyBaseDir(baseDir)
	if err != nil {
		return "", err
	}
	return filepath.Abs(filepath.Join(resolvedBaseDir, path))
}
