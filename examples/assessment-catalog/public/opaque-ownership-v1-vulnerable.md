# Tsunami lab assessment

Run: `8a0496c75cae504507bfae21057a16c4`

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
| `d19cf439a27ecf973aaf603d5f534dde` | alice | 0 | true | destination is outside the approved origin |
| `d46ea22d028593f452dc26dc698c38cf` | alice | 200 | false |  |
| `55a986be58f4b92d1b83e3fe12e7a09b` | bob | 200 | false |  |
| `618b623929e4864c6bcc0e00eef70e48` | bob | 200 | false |  |

### Request d19cf439a27ecf973aaf603d5f534dde

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:50.815444014Z; finished: 2026-11-02T07:39:50.81545072Z

```json

```

### Request d46ea22d028593f452dc26dc698c38cf

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:50.8200923Z; finished: 2026-11-02T07:39:50.824186109Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 55a986be58f4b92d1b83e3fe12e7a09b

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:50.827208434Z; finished: 2026-11-02T07:39:50.829142722Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 618b623929e4864c6bcc0e00eef70e48

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-7f3c9a

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:50.831678132Z; finished: 2026-11-02T07:39:50.833672408Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
