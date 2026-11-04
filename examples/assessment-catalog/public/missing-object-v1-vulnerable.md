# Tsunami lab assessment

Run: `ed27dfec5a8b13bcb3dd26651503e942`

Mode: scripted Mecatl / synthetic lab only

Scenario: vulnerable

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
| `f29c6320c107b5bc927d1e739b1214de` | alice | 0 | true | destination is outside the approved origin |
| `9ef0421cd934e832ac31eb1ada7fed98` | alice | 200 | false |  |
| `1e2ed90f5583036521e656651e4e0751` | bob | 200 | false |  |
| `9d128a499205bc032cc4643abbcde607` | bob | 404 | false |  |

### Request f29c6320c107b5bc927d1e739b1214de

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:48.832316353Z; finished: 2026-11-02T07:39:48.832319769Z

```json

```

### Request 9ef0421cd934e832ac31eb1ada7fed98

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:48.835321366Z; finished: 2026-11-02T07:39:48.838488525Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 1e2ed90f5583036521e656651e4e0751

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:48.841332483Z; finished: 2026-11-02T07:39:48.847040327Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 9d128a499205bc032cc4643abbcde607

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-missing-99e

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:48.852572038Z; finished: 2026-11-02T07:39:48.856235033Z

```json
{"error":"not found"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
