# Tsunami lab assessment

Run: `e8b6a96a7763833d3d2eee22005228b9`

Mode: scripted Mecatl / synthetic lab only

Scenario: vulnerable

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

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
| `e3423095bff56323978528b2ca167f36` | alice | 0 | true | destination is outside the approved origin |
| `ea004e016b849882148149002783782b` | alice | 200 | false |  |
| `4983a6bf482e65d51936a5305940cdf6` | bob | 200 | false |  |
| `06a2945b60935aa6c3e1ca5f21a06e16` | bob | 200 | false |  |

### Request e3423095bff56323978528b2ca167f36

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:50.76330787Z; finished: 2026-11-02T07:39:50.763310098Z

```json

```

### Request ea004e016b849882148149002783782b

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:50.766373556Z; finished: 2026-11-02T07:39:50.777371576Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 4983a6bf482e65d51936a5305940cdf6

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:50.781220852Z; finished: 2026-11-02T07:39:50.794978069Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 06a2945b60935aa6c3e1ca5f21a06e16

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:50.799553911Z; finished: 2026-11-02T07:39:50.869750047Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
