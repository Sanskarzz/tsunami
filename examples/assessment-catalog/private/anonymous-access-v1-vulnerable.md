# Tsunami lab assessment

Run: `b66d015bf192b3b40831ed88f1f67a08`

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
| `66b414c671f2c8e4d3d0183f4d9e270f` | alice | 0 | true | destination is outside the approved origin |
| `7ea56020532c785da7d505e026b19f18` | alice | 200 | false |  |
| `15819bc7253024fdfb70696660ce0e13` | bob | 200 | false |  |
| `009610529d5e98aacc3fc95d4f88eb22` | anonymous | 200 | false |  |

### Request 66b414c671f2c8e4d3d0183f4d9e270f

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:46.635234006Z; finished: 2026-11-02T07:39:46.635236469Z

```json

```

### Request 7ea56020532c785da7d505e026b19f18

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:46.637496344Z; finished: 2026-11-02T07:39:46.64676484Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 15819bc7253024fdfb70696660ce0e13

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:46.653036692Z; finished: 2026-11-02T07:39:46.659833389Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 009610529d5e98aacc3fc95d4f88eb22

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: anonymous; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:46.662271311Z; finished: 2026-11-02T07:39:46.67341481Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
