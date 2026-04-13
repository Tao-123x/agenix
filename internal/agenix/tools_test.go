package agenix

import (
	"testing"
	"time"
)

func TestShellExecReportsStartFailureWithoutPanic(t *testing.T) {
	policy, err := NewPolicy(Permissions{
		Shell: ShellPermissions{Allow: []ShellCommand{{Run: []string{"agenix-missing-command"}}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	tools := NewTools(policy, NewTrace("test", "fake-scripted", Permissions{}))

	_, err = tools.ShellExec([]string{"agenix-missing-command"}, "", time.Second)
	if err == nil {
		t.Fatal("expected command start failure")
	}
	if !IsErrorClass(err, ErrDriverError) {
		t.Fatalf("expected DriverError, got %v", err)
	}
}
