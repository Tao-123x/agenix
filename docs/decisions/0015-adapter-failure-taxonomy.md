# Decision 0015: Adapter Failure Taxonomy

## Status

Accepted

## Context

The reference runtime already had a real adapter boundary in behavior:

- adapter selection
- capability preflight
- adapter execution
- verifier execution

But the CLI and runtime error surface still merged adapter-selection and
capability-preflight failures into `InvalidInput`. That made three different
classes look the same:

- malformed user input
- unknown or unsupported adapter
- adapter capability mismatch

The roadmap already required these states to be distinguished explicitly.

## Decision

The runtime adds a new stable error class:

- `UnsupportedAdapter`

This class is used for:

- unknown builtin adapter names
- adapter does not support the requested skill
- adapter capability preflight failures against `capabilities.requires`

The runtime keeps:

- `InvalidInput` for malformed manifest, trace, and CLI usage errors
- `DriverError` for adapter execution failures after preflight succeeds
- `VerificationFailed` for verifier failures after adapter execution

## Consequences

Positive:

- adapter selection and capability mismatch are now diagnosable without reading
  trace internals
- CLI error output now matches the runtime contract more closely
- post-v0 provider-backed adapter work has a cleaner failure surface to build on

Tradeoffs:

- the taxonomy is still intentionally narrow; it does not add provider-specific
  subcodes or retry semantics
- builtin adapters and provider-backed adapters still share the same top-level
  `UnsupportedAdapter` class

## Follow-up

- keep `UnsupportedAdapter` in the published tool and capability contract docs
- extend the same taxonomy to provider-backed adapters if and when they land
