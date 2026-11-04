# Tsunami lab assessment

Run: `fc38a77e21c0bf6ffe7ec958aa292dd8`

Mode: scripted Mecatl / synthetic lab only

Scenario: vulnerable

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

no_issue_observed: Approved shared-object access

The captured shared-object response matches the explicitly approved access rule. This is a positive control, not a finding.

Procedure: `shared-access/v1`; digest: `df94143d286bea5ab5b89870b351aa929f0310652970d8d83d5403057544420f`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: allow

Missing-evidence feedback issued: 0 / 2

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
| `5d4eec74e0e8fd37ad085bfab2d84c56` | alice | 0 | true | destination is outside the approved origin |
| `adb6ab049c928c649cb089cb58830d98` | alice | 200 | false |  |
| `fe7f32980fdf365fc0f15b0d42b962f7` | bob | 200 | false |  |
| `1834d9a7e1ded6dcea2878d03865687c` | bob | 200 | false |  |

### Request 5d4eec74e0e8fd37ad085bfab2d84c56

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:54.736728847Z; finished: 2026-11-02T07:39:54.736731097Z

```json

```

### Request adb6ab049c928c649cb089cb58830d98

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-shared-40a

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:54.73875433Z; finished: 2026-11-02T07:39:54.747774534Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request fe7f32980fdf365fc0f15b0d42b962f7

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:54.750582749Z; finished: 2026-11-02T07:39:54.760139534Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 1834d9a7e1ded6dcea2878d03865687c

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-shared-40a

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:54.762923073Z; finished: 2026-11-02T07:39:54.774115775Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
