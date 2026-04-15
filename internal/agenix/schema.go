package agenix

import "fmt"

func ValidateManifest(manifest Manifest) error {
	if manifest.APIVersion == "" {
		return missingField("manifest", "apiVersion")
	}
	if manifest.Kind == "" {
		return missingField("manifest", "kind")
	}
	if manifest.Name == "" {
		return missingField("manifest", "name")
	}
	if manifest.Version == "" {
		return missingField("manifest", "version")
	}
	if manifest.Description == "" {
		return missingField("manifest", "description")
	}
	if len(manifest.Tools) == 0 {
		return missingField("manifest", "tools")
	}
	if len(manifest.Outputs.Required) == 0 {
		return missingField("manifest", "outputs.required")
	}
	if len(manifest.Verifiers) == 0 {
		return missingField("manifest", "verifiers")
	}
	for i, verifier := range manifest.Verifiers {
		if verifier.Type == "" {
			return missingField("manifest", fmt.Sprintf("verifiers[%d].type", i))
		}
		if verifier.Name == "" {
			return missingField("manifest", fmt.Sprintf("verifiers[%d].name", i))
		}
		if verifier.Type == "command" && verifier.Command == "" && len(verifier.Run) == 0 {
			return missingField("manifest", fmt.Sprintf("verifiers[%d].cmd", i))
		}
		if len(verifier.Run) > 0 {
			if verifier.Policy == nil {
				return missingField("manifest", fmt.Sprintf("verifiers[%d].policy", i))
			}
			if verifier.Policy.Executable == "" {
				return missingField("manifest", fmt.Sprintf("verifiers[%d].policy.executable", i))
			}
			if verifier.Policy.CWD == "" {
				return missingField("manifest", fmt.Sprintf("verifiers[%d].policy.cwd", i))
			}
			if verifier.Policy.TimeoutMS == 0 {
				return missingField("manifest", fmt.Sprintf("verifiers[%d].policy.timeout_ms", i))
			}
		}
	}
	return nil
}

func ValidateTrace(trace Trace) error {
	if trace.RunID == "" {
		return missingField("trace", "run_id")
	}
	if trace.Skill == "" {
		return missingField("trace", "skill")
	}
	if trace.ModelProfile == "" {
		return missingField("trace", "model_profile")
	}
	if trace.Final.Status == "" {
		return missingField("trace", "final.status")
	}
	for i, event := range trace.Events {
		if event.Type == "" {
			return missingField("trace", fmt.Sprintf("events[%d].type", i))
		}
		if event.Name == "" {
			return missingField("trace", fmt.Sprintf("events[%d].name", i))
		}
	}
	return nil
}

func missingField(scope, field string) error {
	return NewError(ErrInvalidInput, scope+" missing required field: "+field)
}
