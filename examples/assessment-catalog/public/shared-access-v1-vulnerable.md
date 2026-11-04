# Tsunami lab assessment

Run: `803d1e14c7128b9754377719cbcb848a`

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
| `e5d37d1746e7041243bfefc00f41c783` | alice | 0 | true | destination is outside the approved origin |
| `6f7a289979dbc924fafe1d7d3d2a561e` | alice | 200 | false |  |
| `291ae132695fa3f6ec8d33aacb1e5079` | bob | 200 | false |  |
| `3750cc3d0955f65b1d52b156dc3d3624` | bob | 200 | false |  |

### Request e5d37d1746e7041243bfefc00f41c783

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:54.765050645Z; finished: 2026-11-02T07:39:54.765053443Z

```json

```

### Request 6f7a289979dbc924fafe1d7d3d2a561e

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-shared-40a

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:54.768372496Z; finished: 2026-11-02T07:39:54.771007539Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request 291ae132695fa3f6ec8d33aacb1e5079

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:54.774570078Z; finished: 2026-11-02T07:39:54.779293303Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 3750cc3d0955f65b1d52b156dc3d3624

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-shared-40a

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:54.784901124Z; finished: 2026-11-02T07:39:54.788293488Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
