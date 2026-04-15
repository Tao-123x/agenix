# Minimum Trace Redaction Design

## Status

Approved approach: minimum viable trace redaction for Maya Chen's P0.

## Problem

Agenix now records enough runtime and verifier context to support a controlled
technical trial, but trace files still store raw values from tool requests,
tool results, verifier stdout and stderr, final output, and final error
messages. That leaves a direct path for secrets such as bearer tokens, API
keys, passwords, and session tokens to land in on-disk traces.

Maya Chen has already called this out as a procurement blocker:

- secrets can land in trace stdout and stderr without redaction
- redaction and secret handling are not implemented

The current repository also has no runtime redaction layer at all:

- `internal/agenix/trace.go` writes the trace object directly to disk
- `specs/trace.md` says secrets must not appear in trace, but does not define
  the mechanism
- existing tests validate trace shape, not trace sanitization

## Goals

- Add a minimum viable redaction layer before trace data is written to disk.
- Keep trace structure usable for `verify`, `replay`, debugging, and customer
  review.
- Use precise value redaction instead of replacing whole fields whenever
  possible.
- Always apply runtime default redaction rules, even if the manifest does not
  declare anything.
- Allow a skill manifest to append additional redaction rules for
  customer-specific secrets.
- Fail closed if trace redaction cannot be applied safely.

## Non-Goals

- No full secret-detection platform or exhaustive scanner.
- No redaction policy UI or CLI surface in this slice.
- No removal of all sensitive context from trace; paths, command structure, and
  status metadata should remain when they are not themselves secrets.
- No verifier env or network redaction contract in this slice.
- No artifact-level encryption or trace access-control design in this slice.
- No per-team registry of reusable redaction packs in this slice.

## Approaches Considered

### Option 1: Redact once when writing the trace

Build a sanitized copy of the trace immediately before `WriteTrace` encodes it.

Pros:

- smallest runtime change
- keeps event collection flow stable
- guarantees the on-disk trace is redacted
- aligns with P0's minimum viable scope

Cons:

- the in-memory trace still contains raw values until write time

### Option 2: Redact at every trace append site

Apply redaction inside `AddToolEvent`, `AddVerifierEvent`, and `SetFinal`.

Pros:

- the trace object becomes safe earlier

Cons:

- more invasive
- easier to miss call sites
- higher risk of interface churn during a small P0 slice

### Option 3: Write both raw and redacted traces

Keep a private raw trace plus a redacted public trace.

Pros:

- most debugging flexibility

Cons:

- directly conflicts with Maya's blocker because raw secrets still land on disk
- adds too much surface area for a minimum slice

## Decision

Choose Option 1.

The runtime will keep collecting normal trace events in memory, then create a
sanitized copy before persisting a trace file. Only the sanitized copy will be
written to disk and later consumed by `ReadTrace`, `Verify`, and `Replay`.

## Manifest Contract

Add a new top-level manifest block:

```yaml
redaction:
  keys:
    - session_token
    - customer_api_key
  patterns:
    - name: customer-bearer
      regex: '(?i)(x-customer-token:\s*)([^\s]+)'
      secret_group: 2
```

### Semantics

- `redaction.keys`
  - append-only list of additional sensitive field names
  - compared case-insensitively after key normalization
  - applies to structured trace values such as request maps, result maps, and
    final output objects

- `redaction.patterns`
  - append-only list of additional text redaction rules
  - each rule defines:
    - `name`
    - `regex`
    - `secret_group`
  - the runtime replaces only the matching capture group with `[REDACTED]`
  - surrounding context remains visible

### Validation

Manifest validation must reject:

- missing `name` on a custom pattern
- missing `regex` on a custom pattern
- invalid regular expressions
- `secret_group <= 0`
- `secret_group` greater than the number of capture groups in the regex

These failures should return stable `InvalidInput`.

## Runtime Default Rules

The runtime always applies built-in rules, even when the manifest provides no
`redaction` block.

### Default sensitive keys

Initial built-in key set:

- `authorization`
- `api_key`
- `access_token`
- `refresh_token`
- `token`
- `secret`
- `password`

Key matching should normalize:

- case
- `_` versus `-`
- surrounding whitespace

Examples that should all match:

- `Authorization`
- `authorization`
- `api-key`
- `API_KEY`

CamelCase aliases such as `refreshToken` are outside this initial slice unless
they are added explicitly later. Minimum scope stays narrow and predictable.

### Default text patterns

Initial built-in text patterns should cover the most common high-risk cases:

- `Authorization: Bearer <secret>`
- `Bearer <secret>`
- `OPENAI_API_KEY=<secret>`
- `<something>_API_KEY=<secret>`
- `token=<secret>`
- `password=<secret>`

Each built-in pattern should preserve non-secret context and replace only the
secret value with `[REDACTED]`.

## Runtime Behavior

Redaction happens in `WriteTrace`.

### Execution flow

1. Runtime builds the normal in-memory `Trace`.
2. After manifest load, runtime builds the effective redaction config:
   - built-in defaults
   - manifest-provided appended keys and patterns
3. The effective config is attached to the in-memory trace as runtime-only
   context and is not persisted as part of the JSON schema.
4. `WriteTrace(path, trace)` uses that effective config to create a sanitized
   copy of the trace.
5. The sanitized copy is encoded and written to disk.
6. `ReadTrace`, `Verify`, and `Replay` continue to operate on the persisted,
   already-redacted trace.

### Config plumbing

The persisted trace schema does not need to expose redaction rules. The minimum
implementation may keep the effective redaction config as an in-memory-only
field on `Trace`, or pass it through `WriteTrace` via a small internal helper.
The important contract is:

- manifest rules are validated at manifest load time
- runtime default rules always apply
- only the redacted trace is serialized to disk

### Fields to redact

Redaction should recursively scan only payload-bearing fields:

- `events[].request`
- `events[].result`
- `events[].error`
  - especially `message`
- `events[].stdout`
- `events[].stderr`
- `final.output`
- `final.error`

### Fields to preserve

These fields remain unchanged unless their values themselves match a text
pattern:

- `run_id`
- `skill`
- `manifest_path`
- `model_profile`
- `started_at`
- `events[].type`
- `events[].name`
- `events[].status`
- `events[].exit_code`
- `events[].duration_ms`

This preserves audit usefulness while still masking secrets embedded in larger
strings.

## Recursive Redaction Rules

### Structured values

For maps:

- normalize each key
- if the key is sensitive and the value is string-like, replace only the value
  with `[REDACTED]`
- if the value is nested, continue recursion

For arrays:

- recurse element-by-element

For strings:

- apply built-in and manifest-provided regex patterns
- replace only the configured capture group with `[REDACTED]`

For non-string scalars:

- preserve as-is

## Failure Semantics

- invalid manifest redaction config: `InvalidInput`
- runtime failure while compiling or applying redaction during trace write:
  `DriverError`

This slice should fail closed:

- do not silently write an unredacted trace if redaction processing fails
- return the trace write failure instead

## Compatibility Requirements

Minimum trace redaction must not break:

- `ReadTrace` minimum validation
- `Verify` consuming the persisted trace
- `Replay` summarizing the persisted trace
- existing policy-verification assertions that depend on event names, statuses,
  paths, commands, and timeout metadata

Redaction should be narrow enough that these values remain available unless the
value itself is the secret.

## Testing Strategy

Add regression coverage for:

1. built-in key redaction in structured request or result payloads
2. built-in text redaction in verifier stdout and stderr
3. built-in text redaction in final error strings
4. manifest-provided `redaction.keys` appending to the default key set
5. manifest-provided `redaction.patterns` appending to the default pattern set
6. non-sensitive audit fields remain intact
7. invalid manifest redaction regex or `secret_group` fails as `InvalidInput`
8. redacted traces still pass `ReadTrace`, `Verify`, and `Replay` for supported
   scenarios

## Files Expected To Change

- `internal/agenix/manifest.go`
- `internal/agenix/schema.go`
- `internal/agenix/trace.go`
- `internal/agenix/runtime.go`
- `internal/agenix/manifest_test.go`
- `internal/agenix/schema_test.go`
- `internal/agenix/trace_test.go`
- `internal/agenix/runtime_integration_test.go`
- `specs/trace.md`
- `specs/skill-manifest.md`
- one decision record
- one handoff note

## Risks

- Built-in patterns that are too broad could hide useful debugging context.
- Built-in patterns that are too narrow could miss some secrets.
- Redacting too much of stdout or stderr could weaken customer trust in trace
  usefulness.
- Manifest-added regex rules add flexibility, but incorrect custom rules could
  produce surprising masking unless validation is strict.

## Follow-Up

- expand key normalization and alias coverage after customer feedback
- define verifier env redaction and capture contract
- design network redaction once network denial exists
- decide whether raw in-memory traces should also become redacted by default
