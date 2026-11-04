# Tsunami lab assessment

Run: `18a0fbf5406acdd5515f48605ba6ad77`

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
| `9f0088fae80e70b92557b5326caa227d` | alice | 0 | true | destination is outside the approved origin |
| `c2c04dfb52121b5eb4d3ca2ba5f01327` | alice | 200 | false |  |
| `10239e0ed802f61bc3e0f626a466bc30` | bob | 200 | false |  |
| `7bf55240d588f8e9c88a1ca90a8a2b7f` | bob | 404 | false |  |

### Request 9f0088fae80e70b92557b5326caa227d

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:49.770571431Z; finished: 2026-11-02T07:39:49.770576496Z

```json

```

### Request c2c04dfb52121b5eb4d3ca2ba5f01327

GET http://tsunami-fixed:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:49.775191903Z; finished: 2026-11-02T07:39:49.778215261Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 10239e0ed802f61bc3e0f626a466bc30

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:49.781542366Z; finished: 2026-11-02T07:39:49.785105393Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 7bf55240d588f8e9c88a1ca90a8a2b7f

GET http://tsunami-fixed:8080/api/reviewed/objects/r-missing-99e

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:49.791168889Z; finished: 2026-11-02T07:39:49.795132426Z

```json
{"error":"not found"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
