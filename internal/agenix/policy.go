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
}

func NewPolicy(permissions Permissions) (*Policy, error) {
	policy := &Policy{permissions: permissions}
	var err error
	policy.readScopes, err = normalizeScopes(permissions.Filesystem.Read)
	if err != nil {
		return nil, err
	}
	policy.writeScopes, err = normalizeScopes(permissions.Filesystem.Write)
	if err != nil {
		return nil, err
	}
	return policy, nil
}

func (p *Policy) CheckRead(path string) error {
	if !pathInScopes(path, p.readScopes) {
		return NewError(ErrPolicyViolation, "read outside declared scope: "+path)
	}
	return nil
}

func (p *Policy) CheckWrite(path string) error {
	if !pathInScopes(path, p.writeScopes) {
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

func normalizeScopes(scopes []string) ([]string, error) {
	normalized := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		abs, err := filepath.Abs(scope)
		if err != nil {
			return nil, WrapError(ErrInvalidInput, "normalize scope", err)
		}
		normalized = append(normalized, filepath.Clean(abs))
	}
	return normalized, nil
}

func pathInScopes(path string, scopes []string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	clean := filepath.Clean(abs)
	for _, scope := range scopes {
		if clean == scope {
			return true
		}
		rel, err := filepath.Rel(scope, clean)
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
