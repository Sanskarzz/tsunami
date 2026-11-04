# Tsunami lab assessment

Run: `852646b48a615e74b835e73247618b51`

Mode: OpenRouter free model / synthetic lab only

Scenario: fixed

State: completed

A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.

Workflow: shared-access/v1; provider: openrouter-free; model: nvidia/nemotron-3.5-lightning:free.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 6707 / 203; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

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
| `4ec5d4f2917998fafceabd818471b97f` | alice | 200 | false |  |
| `93f207667fcb9f70bfbd2051b94541d8` | bob | 200 | false |  |
| `738d359740d0880bb6656bf955aa91d2` | bob | 200 | false |  |

### Request 4ec5d4f2917998fafceabd818471b97f

GET http://tsunami-fixed:8080/api/reviewed/objects/r-shared-40a

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:40:49.322269658Z; finished: 2026-11-02T07:40:49.32485251Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request 93f207667fcb9f70bfbd2051b94541d8

GET http://tsunami-fixed:8080/api/reviewed/objects/r-shared-40a

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:40:50.96167662Z; finished: 2026-11-02T07:40:50.965189749Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request 738d359740d0880bb6656bf955aa91d2

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:40:54.254088281Z; finished: 2026-11-02T07:40:54.257442826Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
