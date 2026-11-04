# Tsunami lab assessment

Run: `e429c3c9237451a9e0afd28d4478e50d`

Mode: scripted Mecatl / synthetic lab only

Scenario: fixed

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
| `58e294065b287e139a5c575c8e2eeb54` | alice | 0 | true | destination is outside the approved origin |
| `df4272b1848c9acfb07ac6371ffd0d47` | alice | 200 | false |  |
| `2e9563155087055f2f8cf6bd83ff5de7` | bob | 200 | false |  |
| `09011c12d8b5bae649c47cb2c8d3fd22` | bob | 200 | false |  |

### Request 58e294065b287e139a5c575c8e2eeb54

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:55.73599795Z; finished: 2026-11-02T07:39:55.736000376Z

```json

```

### Request df4272b1848c9acfb07ac6371ffd0d47

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-shared-40a

Account alias: alice; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:55.73847863Z; finished: 2026-11-02T07:39:55.746742474Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request 2e9563155087055f2f8cf6bd83ff5de7

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:55.750554233Z; finished: 2026-11-02T07:39:55.761104099Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 09011c12d8b5bae649c47cb2c8d3fd22

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-shared-40a

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:55.764537321Z; finished: 2026-11-02T07:39:55.776878848Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
