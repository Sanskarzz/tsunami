# Tsunami lab assessment

Run: `ac827e32f8fac65815a96fc8d4dde7c9`

Mode: scripted Mecatl / synthetic lab only

Scenario: vulnerable

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

confirmed: Unauthenticated access to a known private object

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

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
| `6f1c35cfb81dd552579b29b1a85f52be` | alice | 0 | true | destination is outside the approved origin |
| `098750e65e4c4db577af9ca563d79c2c` | alice | 200 | false |  |
| `c0c23360d4c2e9a098707e3bb53293ac` | bob | 200 | false |  |
| `a3cb1dc0f466115efcd6dcdf8b981f3c` | anonymous | 200 | false |  |

### Request 6f1c35cfb81dd552579b29b1a85f52be

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:46.67426701Z; finished: 2026-11-02T07:39:46.674268504Z

```json

```

### Request 098750e65e4c4db577af9ca563d79c2c

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:46.67960902Z; finished: 2026-11-02T07:39:46.682768545Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request c0c23360d4c2e9a098707e3bb53293ac

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:46.686489897Z; finished: 2026-11-02T07:39:46.689416115Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request a3cb1dc0f466115efcd6dcdf8b981f3c

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-7f3c9a

Account alias: anonymous; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:46.695401179Z; finished: 2026-11-02T07:39:46.699621619Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
