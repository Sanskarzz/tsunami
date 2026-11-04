# Tsunami lab assessment

Run: `5281dc7f709dd81836073b8aced75629`

Mode: scripted Mecatl / synthetic lab only

Scenario: fixed

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

inconclusive: Missing-object negative control

The probe uses an object with no confirmed existence. Its rejection cannot prove ownership enforcement.

Procedure: `missing-object/v1`; digest: `7677e75cb7264c9c51f4d51483b729d1e04a5af30a8c1725477160950cfc09c6`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: unknown

Missing-evidence feedback issued: 0 / 2

## Reproduction

1. Read `/api/reviewed/objects/r-7f3c9a` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/objects/r-missing-99e` using the `bob` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Use a known existing private object and approved second account before making an authorization conclusion.

## Evidence

Requests reserved: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `6a039b43d80506dd9824428f2e2cb3c0` | alice | 0 | true | destination is outside the approved origin |
| `297d1989b3d1c9876bf7c04235c973fd` | alice | 200 | false |  |
| `16effd2b3a986655f01d1838989d6be7` | bob | 200 | false |  |
| `445cca3e6de294a94a2b77a5b1aa1163` | bob | 404 | false |  |

### Request 6a039b43d80506dd9824428f2e2cb3c0

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:49.744799051Z; finished: 2026-11-02T07:39:49.744801266Z

```json

```

### Request 297d1989b3d1c9876bf7c04235c973fd

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:49.748589683Z; finished: 2026-11-02T07:39:49.762102919Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 16effd2b3a986655f01d1838989d6be7

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:49.767599214Z; finished: 2026-11-02T07:39:49.780373749Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 445cca3e6de294a94a2b77a5b1aa1163

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-missing-99e

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:49.785548243Z; finished: 2026-11-02T07:39:49.84880227Z

```json
{"error":"not found"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
