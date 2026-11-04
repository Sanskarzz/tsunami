# Tsunami lab assessment

Run: `e875556e0c988d6462ec5a63961dc5df`

Mode: scripted Mecatl / synthetic lab only

Scenario: fixed

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

no_issue_observed: Cross-user access to an opaque object ID

Known immutable fixture baselines succeeded and the prohibited read returned HTTP 404. This applies only to this procedure, not application-wide security.

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
| `56a2c4ddb8ed0b77b7129291d7c3bb47` | alice | 0 | true | destination is outside the approved origin |
| `05c2e878f9b38270e1286af2220fbd97` | alice | 200 | false |  |
| `534d8f1c248bd53396a4ef5aaadd26f1` | bob | 200 | false |  |
| `1149af666cef96fba86ef8296f22e1f0` | bob | 404 | false |  |

### Request 56a2c4ddb8ed0b77b7129291d7c3bb47

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:51.738944231Z; finished: 2026-11-02T07:39:51.738946544Z

```json

```

### Request 05c2e878f9b38270e1286af2220fbd97

GET http://tsunami-fixed:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:51.741282317Z; finished: 2026-11-02T07:39:51.743525135Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 534d8f1c248bd53396a4ef5aaadd26f1

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:51.746488422Z; finished: 2026-11-02T07:39:51.749882853Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 1149af666cef96fba86ef8296f22e1f0

GET http://tsunami-fixed:8080/api/reviewed/objects/r-7f3c9a

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:51.774326318Z; finished: 2026-11-02T07:39:51.778619692Z

```json
{"error":"not found"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
