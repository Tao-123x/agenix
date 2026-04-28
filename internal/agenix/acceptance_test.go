package agenix

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

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

func TestV03AcceptanceSweepForAdapterReadinessRelease(t *testing.T) {
	summary, err := RunV03AcceptanceSweep(AcceptanceOptions{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatalf("v0.3 acceptance sweep failed: %v", err)
	}
	if summary.Status != "passed" {
		t.Fatalf("v0.3 acceptance status = %q", summary.Status)
	}
	if summary.AdapterCount != 5 {
		t.Fatalf("v0.3 adapter count = %d", summary.AdapterCount)
	}
	if summary.CompatibilityReportCount != 3 {
		t.Fatalf("v0.3 compatibility report count = %d", summary.CompatibilityReportCount)
	}
	if summary.SchemaCount != 3 {
		t.Fatalf("v0.3 schema count = %d", summary.SchemaCount)
	}
	if summary.ProviderSmokeStatus != "skipped_offline" {
		t.Fatalf("v0.3 provider smoke status = %q", summary.ProviderSmokeStatus)
	}
}

func TestV03AcceptanceSweepRecordsSkippedProviderSmokeWithoutCredentials(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "")
	t.Setenv("AGENIX_OPENAI_BASE_URL", "")

	summary, err := RunV03AcceptanceSweep(AcceptanceOptions{WorkDir: t.TempDir(), ProviderSmoke: true})
	if err != nil {
		t.Fatalf("v0.3 acceptance sweep failed: %v", err)
	}
	if summary.Status != "passed" {
		t.Fatalf("v0.3 acceptance status = %q", summary.Status)
	}
	if summary.ProviderSmokeStatus != "skipped_no_credentials" {
		t.Fatalf("v0.3 provider smoke status = %q", summary.ProviderSmokeStatus)
	}
	if summary.ProviderSmokeTracePath != "" {
		t.Fatalf("provider smoke trace should be empty when skipped, got %q", summary.ProviderSmokeTracePath)
	}
}

func TestV03AcceptanceSweepRunsProviderSmokeWithStubProvider(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "output": [
    {
      "type": "message",
      "content": [
        {
          "type": "output_text",
          "text": "{\"analysis_summary\":\"fixture fails\",\"failing_tests\":[\"test_mathlib.py::test_adds_numbers\"],\"likely_root_cause\":\"mathlib.add subtracts instead of adding\",\"changed_files\":[]}"
        }
      ]
    }
  ]
}`))
	}))
	defer server.Close()

	t.Setenv("OPENAI_API_KEY", "test-key")
	t.Setenv("AGENIX_OPENAI_BASE_URL", server.URL)

	summary, err := RunV03AcceptanceSweep(AcceptanceOptions{WorkDir: t.TempDir(), ProviderSmoke: true})
	if err != nil {
		t.Fatalf("v0.3 acceptance sweep failed: %v", err)
	}
	if atomic.LoadInt32(&callCount) == 0 {
		t.Fatal("stub provider server was not called")
	}
	if summary.ProviderSmokeStatus != "passed" {
		t.Fatalf("v0.3 provider smoke status = %q", summary.ProviderSmokeStatus)
	}
	if summary.ProviderSmokeTracePath == "" {
		t.Fatal("provider smoke trace path is empty")
	}
	trace, err := ReadTrace(summary.ProviderSmokeTracePath)
	if err != nil {
		t.Fatalf("read provider smoke trace: %v", err)
	}
	if trace.Final.Status != "passed" {
		t.Fatalf("provider smoke trace status = %q", trace.Final.Status)
	}
}
