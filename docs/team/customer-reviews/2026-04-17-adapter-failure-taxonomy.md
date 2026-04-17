# Customer Review

## Reviewer

Maya Chen

## Change Under Review

`adapter failure taxonomy`

## Trial Verdict

`approve`

## Procurement Verdict

`approve`

## Why This Matters

- malformed input and unsupported adapter selection no longer collapse into the
  same CLI class
- adapter preflight mismatch is now diagnosable without reading runtime code
- the reference runtime keeps verifier and adapter execution failures distinct

## Acceptance Criteria

- unknown adapter names fail as `UnsupportedAdapter`
- unsupported skill or capability mismatch fail as `UnsupportedAdapter`
- adapter execution failure remains `DriverError`
- verifier failure remains `VerificationFailed`
- published runtime contracts reflect the implemented taxonomy

## Blockers

- none for the taxonomy slice itself

## Do Not Build Next

- provider-specific retry logic
- adapter scoring heuristics
- per-vendor error-class forks

## Buyer Summary

`This closes the last obvious ambiguity in the adapter boundary because unsupported adapter selection is now a first-class runtime error instead of a generic input failure.`
