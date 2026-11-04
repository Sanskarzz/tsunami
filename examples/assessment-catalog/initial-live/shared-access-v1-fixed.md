# Tsunami lab assessment

Run: `99578d514a68b8382549aa97576d93e9`

Mode: OpenRouter free model / synthetic lab only

Scenario: fixed

State: completed

A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.

Workflow: shared-access/v1; provider: openrouter-free; model: nvidia/nemotron-3.5-lightning:free.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 6879 / 244; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

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
| `8de7456904f3c369cc1665d772fb8a57` | alice | 200 | false |  |
| `9c36b83bc30930a7ad23304459c422d0` | bob | 200 | false |  |
| `8ae87a4b1a2d95306f18afc3f51e5243` | alice | 0 | true | account/path pair outside the approved procedure |
| `f7f620103a8f811ae4c43aa7cb11b067` | bob | 200 | false |  |

### Request 8de7456904f3c369cc1665d772fb8a57

GET http://tsunami-fixed:8080/api/reviewed/objects/r-shared-40a

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:26:52.096878132Z; finished: 2026-11-02T07:26:52.099156741Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request 9c36b83bc30930a7ad23304459c422d0

GET http://tsunami-fixed:8080/api/reviewed/objects/r-shared-40a

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:26:53.330755111Z; finished: 2026-11-02T07:26:53.334048145Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request 8ae87a4b1a2d95306f18afc3f51e5243

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: alice; connected peer: 

Started: 2026-11-02T07:26:55.580137655Z; finished: 2026-11-02T07:26:55.58014426Z

```json

```

### Request f7f620103a8f811ae4c43aa7cb11b067

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:26:59.269660477Z; finished: 2026-11-02T07:26:59.272698566Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
