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

func TestV02AcceptanceSweepForSkillAuthoringRelease(t *testing.T) {
	summary, err := RunV02AcceptanceSweep(AcceptanceOptions{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatalf("v0.2 acceptance sweep failed: %v", err)
	}
	if summary.Status != "passed" {
		t.Fatalf("v0.2 acceptance status = %q", summary.Status)
	}
	if summary.TemplateCount != 2 {
		t.Fatalf("v0.2 template count = %d", summary.TemplateCount)
	}
	if summary.SkillCount != 2 {
		t.Fatalf("v0.2 generated skill count = %d", summary.SkillCount)
	}
	if summary.CheckCount != 3 {
		t.Fatalf("v0.2 check count = %d", summary.CheckCount)
	}
	if summary.RunCount != 5 {
		t.Fatalf("v0.2 run count = %d", summary.RunCount)
	}
	if summary.FailureReportCount != 1 {
		t.Fatalf("v0.2 failure report count = %d", summary.FailureReportCount)
	}
}
