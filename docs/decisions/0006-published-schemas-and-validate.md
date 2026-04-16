# 0006: Published Schemas and Validate Command

## Status

Accepted

## Context

Agenix already had runtime-level shape checks for manifests and traces, but it
did not publish machine-readable schema files or provide a standalone contract
validation command.

## Decision

Add two published schema files:

- `specs/manifest.schema.json`
- `specs/trace.schema.json`

Add a CLI command:

- `agenix validate <manifest|trace>`

Validation behavior for v0:

- `LoadManifest` and `ReadTrace` remain the authoritative runtime parsers and
  minimum semantic validators.
- `agenix validate` applies the published schema-backed document check after the
  runtime parser succeeds.
- Validation target detection is content-based: JSON-like content is treated as
  trace, otherwise manifest.

## Consequences

- External tooling now has stable schema artifacts to consume.
- The CLI can check contracts without executing a run.
- Schema validation stays aligned with current runtime behavior instead of
  replacing it with a stricter, independent parser.
