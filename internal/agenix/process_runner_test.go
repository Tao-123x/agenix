package agenix

import (
	"reflect"
	"testing"
)

func TestPrepareNetworkDeniedLaunchAllowsPythonAndOfflineGit(t *testing.T) {
	t.Run("python", func(t *testing.T) {
		launch, err := prepareCommandLaunch([]string{"python3", "check.py"}, Permissions{Network: false})
		if err != nil {
			t.Fatalf("prepareCommandLaunch returned error: %v", err)
		}
		if !reflect.DeepEqual(launch.Argv, []string{"python3", "check.py"}) {
			t.Fatalf("unexpected argv: %#v", launch.Argv)
		}
		if len(launch.Env) == 0 {
			t.Fatal("expected injected environment for python network denial")
		}
		launch.Cleanup()
	})

	t.Run("git", func(t *testing.T) {
		launch, err := prepareCommandLaunch([]string{"git", "status", "--short"}, Permissions{Network: false})
		if err != nil {
			t.Fatalf("prepareCommandLaunch returned error: %v", err)
		}
		if !reflect.DeepEqual(launch.Argv, []string{"git", "status", "--short"}) {
			t.Fatalf("unexpected argv: %#v", launch.Argv)
		}
		if len(launch.Env) != 0 {
			t.Fatalf("did not expect extra env for offline git, got %#v", launch.Env)
		}
	})
}

func TestPrepareNetworkDeniedLaunchRejectsUnsupportedExecutable(t *testing.T) {
	_, err := prepareCommandLaunch([]string{"curl", "https://example.com"}, Permissions{Network: false})
	if err == nil {
		t.Fatal("expected PolicyViolation")
	}
	if !IsErrorClass(err, ErrPolicyViolation) {
		t.Fatalf("expected PolicyViolation, got %v", err)
	}
}
