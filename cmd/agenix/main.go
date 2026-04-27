package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"agenix/internal/agenix"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error=%s message=%s\n", agenix.ErrorClass(err), err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 1 {
		return usage()
	}
	switch args[0] {
	case "acceptance":
		if len(args) != 1 {
			return usage()
		}
		summary, err := agenix.RunV0AcceptanceSweep(agenix.AcceptanceOptions{})
		if err != nil {
			return err
		}
		fmt.Println(formatAcceptanceSummary(summary))
		return nil
	case "build":
		if len(args) != 4 || args[2] != "-o" {
			return usage()
		}
		result, err := agenix.BuildArtifact(agenix.BuildOptions{SkillDir: args[1], OutputPath: args[3]})
		if err != nil {
			return err
		}
		fmt.Println(formatArtifactSummary(result))
		return nil
	case "init":
		return runInit(args[1:])
	case "check":
		target, registryRoot, adapterName, err := parseRunArgs(args[1:])
		if err != nil {
			return err
		}
		adapter, err := agenix.ResolveBuiltinAdapter(adapterName)
		if err != nil {
			return err
		}
		result, err := agenix.CheckSkill(agenix.CheckOptions{Target: target, RegistryRoot: registryRoot, Adapter: adapter})
		if err != nil {
			if result.TracePath != "" {
				fmt.Printf("status=failed skill=%s artifact=%s run_id=%s trace=%s\n", result.Skill, result.ArtifactPath, result.RunID, result.TracePath)
			}
			return err
		}
		fmt.Println(formatCheckResult(result))
		return nil
	case "inspect":
		target, registryRoot, err := parseTargetWithOptionalRegistry(args[1:])
		if err != nil {
			return err
		}
		target, err = agenix.ResolveRegistryReference(target, registryRoot)
		if err != nil {
			return err
		}
		result, err := agenix.InspectArtifact(target)
		if err != nil {
			return err
		}
		fmt.Println(formatArtifactSummary(result))
		return nil
	case "run":
		target, registryRoot, adapterName, err := parseRunArgs(args[1:])
		if err != nil {
			return err
		}
		adapter, err := agenix.ResolveBuiltinAdapter(adapterName)
		if err != nil {
			return err
		}
		result, err := agenix.Run(agenix.RunOptions{ManifestPath: target, RegistryRoot: registryRoot, Adapter: adapter})
		if err != nil {
			if result.TracePath != "" {
				fmt.Printf("status=failed run_id=%s trace=%s\n", result.RunID, result.TracePath)
			}
			return err
		}
		fmt.Println(formatRunResult(result.Status, result.RunID, result.TracePath, result.ChangedFiles, result.VerifierSummary))
		return nil
	case "verify":
		if len(args) != 2 {
			return usage()
		}
		result, err := agenix.Verify(args[1])
		if err != nil {
			return err
		}
		fmt.Println(formatRunResult(result.Status, result.RunID, result.TracePath, result.ChangedFiles, result.VerifierSummary))
		return nil
	case "replay":
		if len(args) != 2 {
			return usage()
		}
		summary, err := agenix.Replay(args[1])
		if err != nil {
			return err
		}
		fmt.Println(formatReplaySummary(summary))
		for i, event := range summary.Events {
			fmt.Println(formatReplayEvent(i, event))
		}
		if summary.FinalOutput != nil {
			fmt.Printf("final_output=%s\n", mustJSON(summary.FinalOutput))
		}
		if summary.FinalError != "" {
			fmt.Printf("final_error=%s\n", summary.FinalError)
		}
		return nil
	case "validate":
		if len(args) != 2 {
			return usage()
		}
		kind, schemaPath, err := agenix.ValidateTarget(args[1])
		if err != nil {
			return err
		}
		fmt.Printf("status=valid kind=%s schema=%s path=%s\n", kind, schemaPath, args[1])
		return nil
	case "publish":
		artifactPath, registryRoot, err := parsePublishArgs(args[1:])
		if err != nil {
			return err
		}
		entry, err := agenix.PublishArtifact(agenix.PublishOptions{ArtifactPath: artifactPath, RegistryRoot: registryRoot})
		if err != nil {
			return err
		}
		fmt.Println(formatRegistryEntry(entry))
		return nil
	case "pull":
		ref, outputPath, registryRoot, err := parsePullArgs(args[1:])
		if err != nil {
			return err
		}
		summary, err := agenix.PullArtifact(agenix.PullOptions{Reference: ref, OutputPath: outputPath, RegistryRoot: registryRoot})
		if err != nil {
			return err
		}
		fmt.Println(formatArtifactSummary(summary))
		return nil
	case "registry":
		return runRegistry(args[1:])
	default:
		return usage()
	}
}

func usage() error {
	return agenix.NewError(agenix.ErrInvalidInput, "usage: agenix acceptance | init skill <name> --template python-pytest -o <dir> | check <skill-dir|manifest|artifact> [--registry <dir>] [--adapter <name>] | build <skill-dir> -o <artifact> | inspect <artifact> | run <manifest> [--registry <dir>] [--adapter <name>] | verify <trace> | replay <trace> | validate <manifest|trace> | publish <artifact> [--registry <dir>] | pull <skill@version|sha256:digest> -o <artifact> [--registry <dir>] | registry list [--registry <dir>] | registry show <skill> [--registry <dir>] | registry resolve <skill@version|sha256:digest> [--registry <dir>]")
}

func formatAcceptanceSummary(summary agenix.AcceptanceSummary) string {
	return fmt.Sprintf("status=%s skills=%d runs=%d", summary.Status, summary.SkillCount, summary.RunCount)
}

func formatInitSkillResult(result agenix.InitSkillResult) string {
	return fmt.Sprintf("status=created skill=%s template=%s path=%s", result.Name, result.Template, result.Path)
}

func formatCheckResult(result agenix.CheckResult) string {
	return fmt.Sprintf("status=%s skill=%s version=%s artifact=%s run_id=%s trace=%s changed_files=%s verifiers=%s events=%d", result.Status, result.Skill, result.Version, result.ArtifactPath, result.RunID, result.TracePath, strings.Join(result.ChangedFiles, ","), strings.Join(result.VerifierSummary, ","), result.EventCount)
}

func formatRunResult(status, runID, tracePath string, changedFiles, verifierSummary []string) string {
	return fmt.Sprintf("status=%s run_id=%s trace=%s changed_files=%s verifiers=%s", status, runID, tracePath, strings.Join(changedFiles, ","), strings.Join(verifierSummary, ","))
}

func formatArtifactSummary(summary agenix.ArtifactSummary) string {
	return fmt.Sprintf("skill=%s version=%s files=%d digest=%s artifact=%s created_at=%s built_by=%s build_host=%s source_commit=%s", summary.Skill, summary.Version, summary.FileCount, summary.Digest, summary.Path, summary.CreatedAt.Format(time.RFC3339), summary.BuiltBy, summary.BuildHost, summary.SourceCommit)
}

func formatRegistryEntry(entry agenix.RegistryEntry) string {
	return fmt.Sprintf("skill=%s version=%s digest=%s registry_artifact=%s published_at=%s published_by=%s source_commit=%s", entry.Skill, entry.Version, entry.Digest, entry.ArtifactPath, entry.PublishedAt.Format(time.RFC3339), entry.PublishedBy, entry.SourceCommit)
}

func formatRegistryEntries(entries []agenix.RegistryEntry) string {
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		lines = append(lines, formatRegistryEntry(entry))
	}
	return strings.Join(lines, "\n")
}

func formatReplaySummary(summary agenix.ReplaySummary) string {
	return fmt.Sprintf("run_id=%s skill=%s status=%s events=%d", summary.RunID, summary.Skill, summary.FinalStatus, summary.EventCount)
}

func formatReplayEvent(index int, event agenix.TraceEvent) string {
	parts := []string{
		fmt.Sprintf("event[%d]", index),
		"type=" + event.Type,
		"name=" + event.Name,
	}
	if event.Status != "" {
		parts = append(parts, "status="+event.Status)
	}
	if event.Type == "verifier" || event.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("exit_code=%d", event.ExitCode))
	}
	if event.DurationMS != 0 {
		parts = append(parts, fmt.Sprintf("duration_ms=%d", event.DurationMS))
	}
	if class := replayErrorClass(event.Error); class != "" {
		parts = append(parts, "error_class="+class)
	}
	return strings.Join(parts, " ")
}

func mustJSON(value interface{}) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return `"<unserializable>"`
	}
	return string(raw)
}

func replayErrorClass(value interface{}) string {
	if typed, ok := value.(map[string]interface{}); ok {
		if class, ok := typed["class"].(string); ok {
			return class
		}
	}
	return ""
}

func parsePublishArgs(args []string) (string, string, error) {
	if len(args) != 1 && len(args) != 3 {
		return "", "", usage()
	}
	artifactPath := args[0]
	if len(args) == 1 {
		return artifactPath, "", nil
	}
	if args[1] != "--registry" {
		return "", "", usage()
	}
	return artifactPath, args[2], nil
}

func parsePullArgs(args []string) (string, string, string, error) {
	if len(args) != 3 && len(args) != 5 {
		return "", "", "", usage()
	}
	if args[1] != "-o" {
		return "", "", "", usage()
	}
	ref := args[0]
	outputPath := args[2]
	if len(args) == 3 {
		return ref, outputPath, "", nil
	}
	if args[3] != "--registry" {
		return "", "", "", usage()
	}
	return ref, outputPath, args[4], nil
}

func parseTargetWithOptionalRegistry(args []string) (string, string, error) {
	if len(args) != 1 && len(args) != 3 {
		return "", "", usage()
	}
	target := args[0]
	if len(args) == 1 {
		return target, "", nil
	}
	if args[1] != "--registry" {
		return "", "", usage()
	}
	return target, args[2], nil
}

func parseRunArgs(args []string) (string, string, string, error) {
	if len(args) < 1 || len(args)%2 == 0 || len(args) > 5 {
		return "", "", "", usage()
	}
	target := args[0]
	registryRoot := ""
	adapterName := ""
	for i := 1; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return "", "", "", usage()
		}
		switch args[i] {
		case "--registry":
			registryRoot = args[i+1]
		case "--adapter":
			adapterName = args[i+1]
		default:
			return "", "", "", usage()
		}
	}
	return target, registryRoot, adapterName, nil
}

func runInit(args []string) error {
	options, err := parseInitSkillArgs(args)
	if err != nil {
		return err
	}
	result, err := agenix.InitSkill(options)
	if err != nil {
		return err
	}
	fmt.Println(formatInitSkillResult(result))
	return nil
}

func parseInitSkillArgs(args []string) (agenix.InitSkillOptions, error) {
	if len(args) < 2 || args[0] != "skill" {
		return agenix.InitSkillOptions{}, usage()
	}
	options := agenix.InitSkillOptions{Name: args[1], Template: agenix.PythonPytestTemplate}
	for i := 2; i < len(args); i += 2 {
		if i+1 >= len(args) {
			return agenix.InitSkillOptions{}, usage()
		}
		switch args[i] {
		case "--template":
			options.Template = args[i+1]
		case "-o":
			options.OutputDir = args[i+1]
		default:
			return agenix.InitSkillOptions{}, usage()
		}
	}
	if options.OutputDir == "" {
		return agenix.InitSkillOptions{}, usage()
	}
	return options, nil
}

func runRegistry(args []string) error {
	if len(args) < 1 {
		return usage()
	}
	switch args[0] {
	case "list":
		registryRoot, err := parseRegistryOnlyArgs(args[1:])
		if err != nil {
			return err
		}
		entries, err := agenix.ListRegistryEntries(registryRoot)
		if err != nil {
			return err
		}
		if len(entries) != 0 {
			fmt.Println(formatRegistryEntries(entries))
		}
		return nil
	case "show":
		skill, registryRoot, err := parseRegistryLookupArgs(args[1:])
		if err != nil {
			return err
		}
		entries, err := agenix.ShowRegistrySkill(skill, registryRoot)
		if err != nil {
			return err
		}
		fmt.Println(formatRegistryEntries(entries))
		return nil
	case "resolve":
		ref, registryRoot, err := parseRegistryLookupArgs(args[1:])
		if err != nil {
			return err
		}
		entry, err := agenix.ResolveRegistryEntry(ref, registryRoot)
		if err != nil {
			return err
		}
		fmt.Println(formatRegistryEntry(entry))
		return nil
	default:
		return usage()
	}
}

func parseRegistryOnlyArgs(args []string) (string, error) {
	if len(args) == 0 {
		return "", nil
	}
	if len(args) == 2 && args[0] == "--registry" {
		return args[1], nil
	}
	return "", usage()
}

func parseRegistryLookupArgs(args []string) (string, string, error) {
	if len(args) != 1 && len(args) != 3 {
		return "", "", usage()
	}
	target := args[0]
	if len(args) == 1 {
		return target, "", nil
	}
	if args[1] != "--registry" {
		return "", "", usage()
	}
	return target, args[2], nil
}
