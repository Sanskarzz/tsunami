> Historical reference, indexed 2026-11-09. Statements below describe their recorded milestone/build. Use [current status](track.md) and the [demo guide](demo-walkthrough.md) for the delivered journey and current readiness. Original findings and failures are preserved.

> This is the original three-read fixture procedure, retained for regression/evidence interpretation. Current configured assessments use the seven-read [selected procedures](demo-configuration.md#selected-applications-and-procedures).

# First workflow: cross-user document reads

This is an original, narrow test procedure informed by the reviewed VAPT methodology. It does not import the toolkit's scripts or grant its prompts authority.

## Preconditions

- A synthetic lab API controlled by the developer.
- Two test identities, Alice and Bob, with known separate private documents.
- Explicitly approved origin, port, and read-only fixture paths.
- Server-owned scope and request budget. No arbitrary target input.

These accounts are users of the target application. They are different from tenants of the testing platform.

## Procedure

1. Establish Alice can read `doc-alice` and inspect its owner/content.
2. Establish Bob can read `doc-bob` and inspect its owner/content.
3. Request `doc-alice` using Bob's account alias, through the broker.
4. Compare the actual returned document with the known ownership rule.
5. Submit no verdict from the worker. Ask the trusted validator to evaluate broker records.
6. Generate a draft with request IDs, timing, account aliases, response evidence, reproduction steps, and remediation.

The vulnerable fixture deliberately omits the ownership check. The fixed fixture checks it. The inconclusive scenario requests an unknown ID instead; it must not be reported as a passed authorization check.

## Result vocabulary

- `confirmed`: the forbidden cross-user read returns the known private fixture document after both baselines succeed.
- `no_issue_observed`: these known-owner checks behaved as expected. It is not a blanket security conclusion.
- `inconclusive`: missing prerequisites, incomplete evidence, transport failures, or an ambiguous result.
- `stopped`: permission was withdrawn. This is a run state, not a finding severity.

HTTP GET is used only because these fixture routes are known not to change business state. GET is not universally harmless on real applications.

## Who owns which decisions

- The operator/control plane owns scope, accounts, run identity, budgets, and stop state.
- The model selects tools. It has no authority to expand scope or mark a finding confirmed.
- The broker owns the actual connection, cancellation, and evidence capture.
- The validator owns this fixture-specific result.
- A human reviews any report before it is shared as a security assessment.

For a real application, a security engineer must define expected access rules and approved accounts/data. Generic LLM guesses about object ownership are not sufficient.
