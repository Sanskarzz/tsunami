# Tsunami lab assessment

Run: `7f1a5bab8dc829bcb575c8234bda2e08`

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
| `f411b0c9dc0a447137b9c75a601f6289` | alice | 0 | true | destination is outside the approved origin |
| `38e5d2fc34daa74124f6531d5baa6e1d` | alice | 200 | false |  |
| `9eed538dad54f3a477e034212b024152` | bob | 200 | false |  |
| `a35961ce5f65fb15622e42c1662a06d7` | bob | 404 | false |  |

### Request f411b0c9dc0a447137b9c75a601f6289

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:51.693619825Z; finished: 2026-11-02T07:39:51.693622429Z

```json

```

### Request 38e5d2fc34daa74124f6531d5baa6e1d

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:51.696951177Z; finished: 2026-11-02T07:39:51.705638571Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 9eed538dad54f3a477e034212b024152

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:51.708333377Z; finished: 2026-11-02T07:39:51.717323554Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request a35961ce5f65fb15622e42c1662a06d7

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:51.719833468Z; finished: 2026-11-02T07:39:51.728405914Z

```json
{"error":"not found"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
