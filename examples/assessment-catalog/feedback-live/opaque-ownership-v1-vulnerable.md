# Tsunami lab assessment

Run: `f70a97c15ddd8565bdc0e8a0d229ca90`

Mode: OpenRouter free model / synthetic lab only

Scenario: vulnerable

State: failed

A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.

Workflow: opaque-ownership/v1; provider: openrouter-free; model: nvidia/nemotron-3.5-lightning:free.

Model calls reserved: 1 / 8; upstream attempts: 1 / 8. Reported input/output tokens: 0 / 0; usage uncertain: true. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

Price policy: openrouter-free/v1 (free model required; zero price ceilings; provider/model fallback disabled).

Model error category: model_error.

## Result

inconclusive: Assessment incomplete

Worker ended before trusted validation; no automatic replay.

Procedure: `opaque-ownership/v1`; digest: `414b6dd1126f39b8a01d8d52f82ac7f31d8d33d57847150be36a9f13ec6764d5`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: deny

Missing-evidence feedback issued: 0 / 2

## Reproduction

1. Read `/api/reviewed/objects/r-7f3c9a` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/objects/r-7f3c9a` using the `bob` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Check object ownership against the authenticated account on every object read. Opaque IDs alone do not authorize access.

## Evidence

Requests reserved: 0 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
