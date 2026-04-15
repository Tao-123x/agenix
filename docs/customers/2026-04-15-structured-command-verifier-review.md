# Customer Review

## Reviewer

Maya Chen

## Change Under Review

`084775c feat: add structured command verifiers` on `codex/structured-command-verifier`

## Trial Verdict

`conditional approve`

## Procurement Verdict

`reject`

## Why This Matters

- Structured verifier argv closes one of the biggest cross-platform gaps in the
  current runtime: verifier commands no longer have to rely on shell string
  parsing for canonical skills.
- This reduces Windows/macOS/Linux drift and makes the artifact loop more
  believable for a real internal platform trial.
- The CRLF-safe constrained refactor fix removes a main-branch Windows failure
  from one of the canonical examples Maya would use in evaluation.

## Acceptance Criteria

- Canonical skills should use `run: [...]` for verifier execution when possible.
- Command verifier manifests must be rejected if they define neither `cmd` nor
  `run`.
- `go test ./... -count=1` must pass on the development host after the change.
- The constrained refactor skill must succeed even when `greeter.py` uses CRLF
  line endings.

## Blockers

- Verifier execution still lacks its own full policy contract for executable,
  cwd, timeout, env, network, stdout, and stderr capture.
- This change is currently only local because GitHub push from this machine is
  blocked by intermittent TLS / `schannel` handshake failures.
- Procurement remains blocked by the same larger items Maya already called out:
  fake adapter, incomplete verifier policy, no redaction, and non-audit-grade
  replay.

## Do Not Build Next

- marketplace
- public registry
- UI dashboard
- daemonized multi-agent scheduler
- remote executor

## Buyer Summary

`This is the right kind of contract-hardening change for a technical trial, but it is still a trial-shaping improvement, not a procurement trigger.`
