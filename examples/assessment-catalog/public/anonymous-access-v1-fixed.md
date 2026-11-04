# Tsunami lab assessment

Run: `8c4597bee5db2547e3360f05a4ff0a5d`

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
| `e681b6aac9e730ae17fb3a3e0d3d1e7b` | alice | 0 | true | destination is outside the approved origin |
| `621bbf0b9df81a8f22ab5e651420f14f` | alice | 200 | false |  |
| `d5c1f72ba687c84b21379c2933365d4f` | bob | 200 | false |  |
| `22a294cc55e2f79f2e0802fa5f317380` | anonymous | 401 | false |  |

### Request e681b6aac9e730ae17fb3a3e0d3d1e7b

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:47.685689404Z; finished: 2026-11-02T07:39:47.685691856Z

```json

```

### Request 621bbf0b9df81a8f22ab5e651420f14f

GET http://tsunami-fixed:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:47.690397631Z; finished: 2026-11-02T07:39:47.692990158Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request d5c1f72ba687c84b21379c2933365d4f

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:47.6960983Z; finished: 2026-11-02T07:39:47.698710583Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 22a294cc55e2f79f2e0802fa5f317380

GET http://tsunami-fixed:8080/api/reviewed/objects/r-7f3c9a

Account alias: anonymous; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:47.710161487Z; finished: 2026-11-02T07:39:47.71279281Z

```json
{"error":"authentication required"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
