package main

import (
	"fmt"
	"os"
	"strings"

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
	case "inspect":
		if len(args) != 2 {
			return usage()
		}
		result, err := agenix.InspectArtifact(args[1])
		if err != nil {
			return err
		}
		fmt.Println(formatArtifactSummary(result))
		return nil
	case "run":
		if len(args) != 2 {
			return usage()
		}
		result, err := agenix.Run(agenix.RunOptions{ManifestPath: args[1]})
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
		fmt.Printf("run_id=%s skill=%s status=%s events=%d\n", summary.RunID, summary.Skill, summary.FinalStatus, summary.EventCount)
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
	default:
		return usage()
	}
}

func usage() error {
	return agenix.NewError(agenix.ErrInvalidInput, "usage: agenix build <skill-dir> -o <artifact> | inspect <artifact> | run <manifest> | verify <trace> | replay <trace> | publish <artifact> [--registry <dir>] | pull <skill@version|sha256:digest> -o <artifact> [--registry <dir>]")
}

func formatRunResult(status, runID, tracePath string, changedFiles, verifierSummary []string) string {
	return fmt.Sprintf("status=%s run_id=%s trace=%s changed_files=%s verifiers=%s", status, runID, tracePath, strings.Join(changedFiles, ","), strings.Join(verifierSummary, ","))
}

func formatArtifactSummary(summary agenix.ArtifactSummary) string {
	return fmt.Sprintf("skill=%s version=%s files=%d digest=%s artifact=%s", summary.Skill, summary.Version, summary.FileCount, summary.Digest, summary.Path)
}

func formatRegistryEntry(entry agenix.RegistryEntry) string {
	return fmt.Sprintf("skill=%s version=%s digest=%s registry_artifact=%s", entry.Skill, entry.Version, entry.Digest, entry.ArtifactPath)
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
