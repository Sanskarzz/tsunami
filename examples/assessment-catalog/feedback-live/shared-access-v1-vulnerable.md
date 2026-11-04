# Tsunami lab assessment

Run: `99314885623bb13a52fbb10df73df307`

Mode: OpenRouter free model / synthetic lab only

Scenario: vulnerable

State: completed

A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.

Workflow: shared-access/v1; provider: openrouter-free; model: nvidia/nemotron-3.5-lightning:free.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 6661 / 212; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

Price policy: openrouter-free/v1 (free model required; zero price ceilings; provider/model fallback disabled).

## Result

no_issue_observed: Approved shared-object access

The captured shared-object response matches the explicitly approved access rule. This is a positive control, not a finding.

Procedure: `shared-access/v1`; digest: `df94143d286bea5ab5b89870b351aa929f0310652970d8d83d5403057544420f`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: allow

Missing-evidence feedback issued: 1 / 2

## Reproduction

1. Read `/api/reviewed/objects/r-shared-40a` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/objects/r-shared-40a` using the `bob` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Preserve the explicit sharing rule. A successful approved share is not a security finding.

## Evidence

Requests reserved: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `b67d7aa7a9e1e3e2a879c0f72b5b8ba3` | bob | 200 | false |  |
| `a278eee58f1bc0f1d99e50220b0aba0c` | alice | 200 | false |  |
| `ac4618bc6ccc49273d68fb1b84e4442e` | bob | 200 | false |  |

### Request b67d7aa7a9e1e3e2a879c0f72b5b8ba3

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-shared-40a

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:40:38.625482026Z; finished: 2026-11-02T07:40:38.62855976Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request a278eee58f1bc0f1d99e50220b0aba0c

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-shared-40a

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:40:41.542567282Z; finished: 2026-11-02T07:40:41.546455891Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request ac4618bc6ccc49273d68fb1b84e4442e

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:40:44.715361075Z; finished: 2026-11-02T07:40:44.718647434Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
