# Tsunami lab assessment

Run: `1660aaf1b2514c65d75a58a236de25f1`

Mode: scripted Mecatl / synthetic lab only

Scenario: fixed

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

no_issue_observed: Unauthenticated access to a known private object

Known immutable fixture baselines succeeded and the prohibited read returned HTTP 401. This applies only to this procedure, not application-wide security.

Procedure: `anonymous-access/v1`; digest: `a268470c93713fd723efb6c8721d9e1c240d3f2e4beeaec1c6501897d57d08aa`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: deny

Missing-evidence feedback issued: 0 / 2

## Reproduction

1. Read `/api/reviewed/objects/r-7f3c9a` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/objects/r-7f3c9a` using the `anonymous` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Require authentication and object authorization before returning private data.

## Evidence

Requests reserved: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `c30c4ebba5352f66adcee2629ff83734` | alice | 0 | true | destination is outside the approved origin |
| `edfe4430ea5513463d8bc79da61763b9` | alice | 200 | false |  |
| `57647073c01ee50279cf976344613679` | bob | 200 | false |  |
| `9263ffd6f762a0fe8c57632a2a3e905e` | anonymous | 401 | false |  |

### Request c30c4ebba5352f66adcee2629ff83734

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:47.714240733Z; finished: 2026-11-02T07:39:47.714242205Z

```json

```

### Request edfe4430ea5513463d8bc79da61763b9

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:47.721213175Z; finished: 2026-11-02T07:39:47.736684231Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 57647073c01ee50279cf976344613679

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:47.741178214Z; finished: 2026-11-02T07:39:47.750690147Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 9263ffd6f762a0fe8c57632a2a3e905e

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: anonymous; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:47.753233675Z; finished: 2026-11-02T07:39:47.761077066Z

```json
{"error":"authentication required"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
