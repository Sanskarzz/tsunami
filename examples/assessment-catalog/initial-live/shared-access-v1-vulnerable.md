# Tsunami lab assessment

Run: `a7119b84077f081ef1af8ac82369ccca`

Mode: OpenRouter free model / synthetic lab only

Scenario: vulnerable

State: completed

A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.

Workflow: shared-access/v1; provider: openrouter-free; model: nvidia/nemotron-3.5-lightning:free.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 6944 / 256; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

Price policy: openrouter-free/v1 (free model required; zero price ceilings; provider/model fallback disabled).

## Result

no_issue_observed: Approved shared-object access

The captured shared-object response matches the explicitly approved access rule. This is a positive control, not a finding.

Procedure: `shared-access/v1`; digest: `df94143d286bea5ab5b89870b351aa929f0310652970d8d83d5403057544420f`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: allow

## Reproduction

1. Read `/api/reviewed/objects/r-shared-40a` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/objects/r-shared-40a` using the `bob` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Preserve the explicit sharing rule. A successful approved share is not a security finding.

## Evidence

Outbound requests: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `3d2ca771a6f445520b3fd407b34d1c75` | alice | 200 | false |  |
| `f4ab33f00db8becabdb1e248e749e555` | bob | 200 | false |  |
| `a2954d2c5b2274ab23dfc3881d01cb99` | alice | 0 | true | account/path pair outside the approved procedure |
| `9bcc77d4892c5e6c3380cc41a06d3f18` | bob | 200 | false |  |

### Request 3d2ca771a6f445520b3fd407b34d1c75

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-shared-40a

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:26:43.293641231Z; finished: 2026-11-02T07:26:43.297741062Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request f4ab33f00db8becabdb1e248e749e555

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-shared-40a

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:26:44.828885431Z; finished: 2026-11-02T07:26:44.83213117Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request a2954d2c5b2274ab23dfc3881d01cb99

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-b82d16

Account alias: alice; connected peer: 

Started: 2026-11-02T07:26:46.572512734Z; finished: 2026-11-02T07:26:46.572519094Z

```json

```

### Request 9bcc77d4892c5e6c3380cc41a06d3f18

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:26:48.415790298Z; finished: 2026-11-02T07:26:48.419300572Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
