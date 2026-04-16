# 0009: Semver-Aware Registry Ordering

## Status

Accepted

## Context

Registry discovery commands now expose local registry contents, but entry
ordering was still plain string ordering. That makes versions like `0.10.0`
sort before `0.2.0`, which is inconsistent with the manifest contract that
describes `version` as semver.

At the same time, the current manifest validator still does not reject
non-semver version strings, so ordering cannot assume every published entry is
strictly valid semver.

## Decision

Keep exact reference matching unchanged and tighten only discovery ordering:

- `registry list` and `registry show` keep sorting by `skill` first
- within a skill, valid semver versions sort in ascending semantic order
- valid semver versions sort before invalid/non-semver strings
- invalid/non-semver strings keep deterministic string ordering
- `digest` remains the final tie-breaker

This change does not add `latest`, implicit version selection, or stronger
manifest validation.

## Consequences

- `0.2.0` now sorts before `0.10.0`
- discovery output is stable and closer to user expectations
- future `latest` or range resolution can build on the same comparator without
  changing exact reference semantics
- manifest parsing still accepts non-semver versions for now; they simply sort
  after valid semver values in discovery output
