# Provider-Backed Adapter Spike Design

## Status

Approved for implementation planning.

## Context

The Agenix reference runtime v0 is complete and acceptance-tested. Post-v0 work
should sharpen adapter realism without weakening the existing runtime contract.

The current runtime already proves:

- local fake and builtin adapters run through the same runtime loop
- capability preflight is explicit
- adapter execution, verifier execution, and policy failures are distinct

The remaining gap is a minimal provider-backed adapter path that can make a
real remote model call when credentials are present, while preserving these
constraints:

- default repository tests remain fully offline
- existing v0 acceptance criteria do not change
- `permissions.network` remains a real runtime boundary rather than a hint

## Goal

Add one minimal provider-backed read-only adapter that can execute a single
skill through a real remote model API when credentials are present, without
changing the current v0 loop or broadening the adapter surface area.

## Non-Goals

This spike does not include:

- multi-provider abstraction or plugin registration
- provider-side tool calling
- provider streaming output
- retry, backoff, or cost controls beyond basic error handling
- adding provider-backed runs to the v0 acceptance sweep
- changing current canonical skill semantics
- weakening `permissions.network`

## Recommended Approach

Use an explicit remote-backed skill variant plus a single provider adapter.

This keeps current canonical skills unchanged and keeps offline behavior
stable. Remote execution becomes opt-in and visible in the manifest rather than
being smuggled through the current `network: false` contracts.

## Chosen Runtime Boundary

### Adapter transport

Extend `AdapterMetadata` with:

- `transport: local | remote`

Current adapters become:

- `fake-scripted` -> `local`
- `heuristic-analyze` -> `local`

New adapter:

- `openai-analyze` -> `remote`

### Preflight sequence

Runtime preflight becomes four ordered stages:

1. `adapter.selection`
2. `adapter.capability_check`
3. `adapter.policy_check`
4. `adapter.execute`

### Remote adapter policy rule

If an adapter reports `transport=remote` and the manifest declares
`permissions.network: false`, runtime must fail before execution with
`PolicyViolation`.

This failure must happen before any provider request is made and before
`adapter.execute`.

### Skill scope

Do not modify the current canonical skills. Add a new manifest-backed example:

- `repo.analyze_test_failures.remote`

This skill:

- remains read-only
- keeps the same output schema as `repo.analyze_test_failures`
- sets `permissions.network: true`
- exists only to make remote execution explicit

## Adapter and Provider Contract

### Adapter interface

Keep the current runtime adapter interface unchanged:

- `Metadata() AdapterMetadata`
- `Execute(manifest Manifest, tools *Tools) (map[string]any, error)`

This spike must not introduce a second runtime adapter interface.

### Provider client

Add one narrow provider client used only by `openai-analyze`.

The provider client accepts a minimal adapter-owned request:

- skill name
- gathered read-only repository context
- expected structured output fields

The provider client returns only the structured output fields required by the
skill:

- `analysis_summary`
- `failing_tests`
- `likely_root_cause`
- `changed_files`

### File and tool access

The provider client must not access the filesystem or runtime tools directly.

The adapter remains responsible for:

- reading files with `fs.read`
- listing files with `fs.list`
- choosing what context to send

This keeps all repository side effects and access under existing runtime tool
policy.

### Credentials

Credentials are environment-variable based for this spike.

If the selected provider adapter has no usable API key:

- adapter selection succeeds
- capability and policy preflight still run normally
- execution fails as `DriverError`

Missing credentials are not `UnsupportedAdapter`, because the adapter contract
is valid even when the backend is unavailable.

## CLI Surface

Do not add new commands. Use the existing interface:

```bash
agenix run <manifest> --adapter openai-analyze
```

This keeps build, inspect, run, verify, and replay behavior consistent.

## Trace Contract

Keep the existing adapter lifecycle events:

- `adapter.selection`
- `adapter.capability_check`
- `adapter.policy_check`
- `adapter.execute`

For provider-backed execution, `adapter.execute.request` should include only the
minimum provider metadata required for audit:

- `adapter`
- `transport`
- `provider`
- `model`

Do not persist:

- API keys
- raw provider request bodies
- raw provider response bodies

Trace should continue to record the final structured output and the stable error
class if execution fails.

## Error Taxonomy

This spike must preserve the current top-level failure classes:

- `UnsupportedAdapter`
  - unknown adapter name
  - adapter does not support the requested skill
  - adapter capability preflight mismatch
- `PolicyViolation`
  - remote adapter selected while manifest declares `permissions.network: false`
- `DriverError`
  - missing credentials
  - HTTP failure
  - provider response decode failure
  - provider response cannot be mapped to required structured output
- `VerificationFailed`
  - schema verifier or command verifier fails after adapter execution

## Testing Strategy

### Default tests

All default repository tests remain offline.

Use `httptest.Server` or an equivalent local stub to test provider behavior:

- remote adapter rejected by `network: false`
- remote manifest plus fake provider response passes
- missing API key returns `DriverError`
- malformed provider response returns `DriverError`
- trace includes provider metadata without leaking secrets

### Manual smoke

Add one opt-in manual smoke path for real provider calls:

```bash
OPENAI_API_KEY=... agenix run examples/repo.analyze_test_failures.remote/manifest.yaml --adapter openai-analyze
```

This smoke path is documented but does not run in default CI.

## Acceptance Criteria

Implementation is acceptable when all of the following are true:

1. `go test ./... -count=1` passes without any real provider credentials
2. current v0 acceptance sweep remains unchanged and passing
3. remote adapter with `permissions.network: false` fails as `PolicyViolation`
4. remote read-only manifest with a stub provider can pass verifiers
5. missing provider credentials fail as `DriverError`
6. trace contains remote adapter metadata without secret leakage
7. manual real-provider smoke is possible when credentials are present

## Implementation Outline

The work should be split into these bounded slices:

1. extend `AdapterMetadata` with transport and add `adapter.policy_check`
2. add remote-network policy enforcement in runtime
3. add a narrow provider client and `openai-analyze` adapter
4. add explicit remote read-only example manifest
5. add offline tests and trace assertions
6. add documentation for opt-in manual smoke

## Risks

- provider response shape may drift from the expected structured output
- the first provider integration may tempt broader abstraction too early
- example naming can become confusing if the remote variant looks canonical when
  it is really a post-v0 spike

## Mitigations

- keep the provider client adapter-specific and narrow
- keep the remote example separate from the current canonical v0 examples
- gate real network execution on both adapter transport and manifest policy
- preserve the existing offline acceptance path as the main regression signal
