package agenix

import (
	"path/filepath"
	"testing"
)

func TestV0AcceptanceSweepForCanonicalSkills(t *testing.T) {
	tests := []struct {
		name        string
		skillDir    string
		skill       string
		adapter     Adapter
		assertRun   func(*testing.T, RunResult)
		assertTrace func(*testing.T, *Trace)
	}{
		{
			name:     "fix-test-failure",
			skillDir: filepath.Join("..", "..", "examples", "repo.fix_test_failure"),
			skill:    "repo.fix_test_failure",
			assertRun: func(t *testing.T, result RunResult) {
				t.Helper()
				if len(result.ChangedFiles) != 1 || filepath.Base(result.ChangedFiles[0]) != "mathlib.py" {
					t.Fatalf("expected changed_files to contain mathlib.py, got %#v", result.ChangedFiles)
				}
			},
			assertTrace: func(t *testing.T, trace *Trace) {
				t.Helper()
				if !traceHasEvent(*trace, "tool_call", "fs.write") {
					t.Fatalf("expected fs.write event in trace: %#v", trace.Events)
				}
			},
		},
		{
			name:     "analyze-test-failures",
			skillDir: filepath.Join("..", "..", "examples", "repo.analyze_test_failures"),
			skill:    "repo.analyze_test_failures",
			adapter:  HeuristicAnalyzeTestFailuresAdapter{},
			assertRun: func(t *testing.T, result RunResult) {
				t.Helper()
				if len(result.ChangedFiles) != 0 {
					t.Fatalf("expected no changed files, got %#v", result.ChangedFiles)
				}
			},
			assertTrace: func(t *testing.T, trace *Trace) {
				t.Helper()
				if !traceHasAdapterEvent(*trace, "execute", "ok") {
					t.Fatalf("expected successful adapter.execute event: %#v", trace.Events)
				}
				if traceHasEvent(*trace, "tool_call", "fs.write") {
					t.Fatalf("read-only skill should not emit fs.write: %#v", trace.Events)
				}
			},
		},
		{
			name:     "apply-small-refactor",
			skillDir: filepath.Join("..", "..", "examples", "repo.apply_small_refactor"),
			skill:    "repo.apply_small_refactor",
			assertRun: func(t *testing.T, result RunResult) {
				t.Helper()
				if len(result.ChangedFiles) != 1 || filepath.Base(result.ChangedFiles[0]) != "greeter.py" {
					t.Fatalf("expected changed_files to contain greeter.py, got %#v", result.ChangedFiles)
				}
			},
			assertTrace: func(t *testing.T, trace *Trace) {
				t.Helper()
				if !traceHasEvent(*trace, "tool_call", "fs.write") {
					t.Fatalf("expected fs.write event in trace: %#v", trace.Events)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			artifactPath := filepath.Join(root, tt.name+".agenix")
			registryRoot := filepath.Join(root, "registry")
			runDir := filepath.Join(root, ".agenix-runs")
			pulledPath := filepath.Join(root, "pulled.agenix")

			kind, _, err := ValidateTarget(filepath.Join(tt.skillDir, "manifest.yaml"))
			if err != nil {
				t.Fatalf("validate manifest: %v", err)
			}
			if kind != "manifest" {
				t.Fatalf("validate manifest kind = %q", kind)
			}

			buildSummary, err := BuildArtifact(BuildOptions{SkillDir: tt.skillDir, OutputPath: artifactPath})
			if err != nil {
				t.Fatalf("build artifact: %v", err)
			}
			if buildSummary.Skill != tt.skill {
				t.Fatalf("build summary skill = %q want %q", buildSummary.Skill, tt.skill)
			}

			inspectSummary, err := InspectArtifact(artifactPath)
			if err != nil {
				t.Fatalf("inspect artifact: %v", err)
			}
			if inspectSummary.Skill != tt.skill {
				t.Fatalf("inspect summary skill = %q want %q", inspectSummary.Skill, tt.skill)
			}

			runResult, trace := runAcceptanceTarget(t, artifactPath, runDir, "", tt.adapter, tt.assertRun, tt.assertTrace)
			if kind, _, err := ValidateTarget(runResult.TracePath); err != nil {
				t.Fatalf("validate trace: %v", err)
			} else if kind != "trace" {
				t.Fatalf("validate trace kind = %q", kind)
			}
			verifyAcceptanceTrace(t, runResult.TracePath)
			replayAcceptanceTrace(t, runResult.TracePath)

			entry, err := PublishArtifact(PublishOptions{ArtifactPath: artifactPath, RegistryRoot: registryRoot})
			if err != nil {
				t.Fatalf("publish artifact: %v", err)
			}
			if entry.Skill != tt.skill {
				t.Fatalf("registry entry skill = %q want %q", entry.Skill, tt.skill)
			}

			pulledSummary, err := PullArtifact(PullOptions{
				Reference:    tt.skill + "@0.1.0",
				OutputPath:   pulledPath,
				RegistryRoot: registryRoot,
			})
			if err != nil {
				t.Fatalf("pull artifact: %v", err)
			}
			if pulledSummary.Skill != tt.skill {
				t.Fatalf("pulled summary skill = %q want %q", pulledSummary.Skill, tt.skill)
			}
			if _, err := InspectArtifact(pulledPath); err != nil {
				t.Fatalf("inspect pulled artifact: %v", err)
			}

			registryRunResult, registryTrace := runAcceptanceTarget(t, tt.skill+"@0.1.0", filepath.Join(root, ".registry-runs"), registryRoot, tt.adapter, tt.assertRun, tt.assertTrace)
			verifyAcceptanceTrace(t, registryRunResult.TracePath)
			replayAcceptanceTrace(t, registryRunResult.TracePath)

			if trace.Final.Status != "passed" || registryTrace.Final.Status != "passed" {
				t.Fatalf("expected passed traces, got artifact=%q registry=%q", trace.Final.Status, registryTrace.Final.Status)
			}
		})
	}
}

func runAcceptanceTarget(t *testing.T, target, runDir, registryRoot string, adapter Adapter, assertRun func(*testing.T, RunResult), assertTrace func(*testing.T, *Trace)) (RunResult, *Trace) {
	t.Helper()
	result, err := Run(RunOptions{
		ManifestPath: target,
		RunDir:       runDir,
		RegistryRoot: registryRoot,
		Adapter:      adapter,
	})
	if err != nil {
		t.Fatalf("run %q: %v", target, err)
	}
	if result.Status != "passed" {
		t.Fatalf("run status = %q", result.Status)
	}
	if result.TracePath == "" {
		t.Fatal("missing trace path")
	}
	assertRun(t, result)

	trace, err := ReadTrace(result.TracePath)
	if err != nil {
		t.Fatalf("read trace: %v", err)
	}
	assertTrace(t, trace)
	return result, trace
}

func verifyAcceptanceTrace(t *testing.T, tracePath string) {
	t.Helper()
	result, err := Verify(tracePath)
	if err != nil {
		t.Fatalf("verify trace %q: %v", tracePath, err)
	}
	if result.Status != "passed" {
		t.Fatalf("verify status = %q", result.Status)
	}
}

func replayAcceptanceTrace(t *testing.T, tracePath string) {
	t.Helper()
	replay, err := Replay(tracePath)
	if err != nil {
		t.Fatalf("replay trace %q: %v", tracePath, err)
	}
	if replay.FinalStatus != "passed" {
		t.Fatalf("replay status = %q", replay.FinalStatus)
	}
	if len(replay.Events) == 0 {
		t.Fatal("expected replay events")
	}
}
