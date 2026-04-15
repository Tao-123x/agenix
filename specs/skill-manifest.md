# Skill Manifest (v0.1 Draft)

## Purpose

Describe a reusable agent capability as a portable, verifiable package.

## Required fields

- `name` (string)
- `version` (semver)
- `description` (string)
- `capabilities` (capability requirements)
- `tools` (required tool namespaces)
- `permissions` (network/filesystem/tool scopes)
- `inputs` (JSON Schema)
- `outputs` (JSON Schema)
- `verifiers` (list)
- `recovery` (checkpoint strategy)

## Example

```yaml
apiVersion: agenix/v0.1
kind: Skill

name: repo.fix_test_failure
version: 0.1.0
description: Locate failing tests, patch code, and verify via test runner.

capabilities:
  requires:
    tool_calling: true
    structured_output: true
    max_context_tokens: 32000
    reasoning_level: medium

tools:
  - fs
  - shell
  - git

permissions:
  network: false
  filesystem:
    read:
      - ${repo_path}
    write:
      - ${repo_path}
  shell:
    allow:
      - run: ["pytest", "-q"]
      - run: ["python", "-m", "pip", "--version"]

inputs:
  type: object
  required: [repo_path]
  properties:
    repo_path:
      type: string
      description: Absolute or repo‑relative path in the runtime workspace.

outputs:
  type: object
  required: [patch_summary, changed_files]
  properties:
    patch_summary:
      type: string
    changed_files:
      type: array
      items:
        type: string

verifiers:
  - type: command
    name: run_tests
    cmd: "cd ${repo_path} && pytest -q"
    success:
      exit_code: 0
    artifacts:
      logs: true

  - type: schema
    name: output_schema_check
    schemaRef: "outputs"

recovery:
  strategy: checkpoint
  intervals: 5
```

## Notes

- `${repo_path}` is a runtime substitution.
- Verifiers are not optional: “agent said done” is not a verifier.
- Permissions must be explicit.

## Implemented minimum validation

The current skeleton implements a lightweight contract check, not full JSON Schema
validation. `LoadManifest` returns `InvalidInput` when these fields are missing:

- `apiVersion`
- `kind`
- `name`
- `version`
- `description`
- `tools`
- `outputs.required`
- `verifiers`
- each verifier's `type`
- each verifier's `name`

The parser now also understands this subset of `capabilities.requires`:

- `tool_calling`
- `structured_output`
- `max_context_tokens`
- `reasoning_level`

The validator intentionally does not yet validate semver format, permission
scope completeness, input/output property schemas, verifier type-specific
fields, or recovery settings.
