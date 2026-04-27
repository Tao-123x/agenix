package agenix

import "testing"

func TestV0AcceptanceSweepForCanonicalSkills(t *testing.T) {
	summary, err := RunV0AcceptanceSweep(AcceptanceOptions{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatalf("acceptance sweep failed: %v", err)
	}
	if summary.Status != "passed" {
		t.Fatalf("acceptance status = %q", summary.Status)
	}
	if summary.SkillCount != 3 {
		t.Fatalf("acceptance skill count = %d", summary.SkillCount)
	}
	if summary.RunCount != 6 {
		t.Fatalf("acceptance run count = %d", summary.RunCount)
	}
}
