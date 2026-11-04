# Tsunami lab assessment

Run: `1d807b27f438b50326ae0e263b3fddca`

Mode: Responses wire fixture / no live model

Scenario: vulnerable

State: completed

This uses a local Responses wire fixture, not a live model. It tests the real adapter and Mecatl loop without paid inference.

Workflow: opaque-ownership/v1; provider: fixture; model: tsunami-wire-fixture.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 200 / 100; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.


## Gateway call diagnostics

Times are measured at the gateway, not provider compute time or time to first token. Pending calls after restart have an unknown outcome.

| Call | Outcome | Code | Phase | HTTP | Attempts | Elapsed ms | Headers ms | First adapter chunk ms | Released tools |
|---|---|---|---|---|---|---|---|---|---|
| 1 | succeeded | ok | complete | 200 | 1 | 2 | 1 | 2 | 1 |
| 2 | succeeded | ok | complete | 200 | 1 | 0 | 0 | 0 | 1 |
| 3 | succeeded | ok | complete | 200 | 1 | 4 | 3 | 3 | 1 |
| 4 | succeeded | ok | complete | 200 | 1 | 3 | 2 | 2 | 1 |
| 5 | succeeded | ok | complete | 200 | 1 | 1 | 1 | 1 | 1 |
## Result

confirmed: Cross-user access to an opaque object ID

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

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

Requests reserved: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `26abd707714be58d580f83e95f0bb428` | alice | 0 | true | destination is outside the approved origin |
| `138217b3aa0e2713678327bec00cf32f` | alice | 200 | false |  |
| `a008c15c241ed26fc8051c8c8e1222a4` | bob | 200 | false |  |
| `202315b43508f762fc9158b383a46290` | bob | 200 | false |  |

### Request 26abd707714be58d580f83e95f0bb428

GET http://not-approved.invalid/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 

Started: 2026-11-02T10:31:38.251300071Z; finished: 2026-11-02T10:31:38.251302538Z

```json

```

### Request 138217b3aa0e2713678327bec00cf32f

GET http://127.0.0.1:43139/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 127.0.0.1:43139

Started: 2026-11-02T10:31:38.253180657Z; finished: 2026-11-02T10:31:38.253897411Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request a008c15c241ed26fc8051c8c8e1222a4

GET http://127.0.0.1:43139/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 127.0.0.1:43139

Started: 2026-11-02T10:31:38.260530445Z; finished: 2026-11-02T10:31:38.261446073Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 202315b43508f762fc9158b383a46290

GET http://127.0.0.1:43139/api/reviewed/objects/r-7f3c9a

Account alias: bob; connected peer: 127.0.0.1:43139

Started: 2026-11-02T10:31:38.267808179Z; finished: 2026-11-02T10:31:38.269035049Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
