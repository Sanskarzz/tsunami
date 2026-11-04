# Tsunami lab assessment

Run: `7637f82fa6aa918d2043773925457f87`

Mode: scripted Mecatl / synthetic lab only / private connector

Scenario: vulnerable

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

confirmed: Broken Object Level Authorization (synthetic invoice API)

Bob received Alice's known private fixture document. Both legitimate ownership baselines succeeded.

## Reproduction

1. Read `doc-alice` using Alice's test session.
2. Read `doc-bob` using Bob's test session.
3. Read `doc-alice` using Bob's test session.
4. Compare the returned ID, owner and synthetic private content with Alice's baseline. The expected result is access denied.

## Remediation

Enforce object ownership/authorization on the server for every document read. An authenticated session alone is not sufficient. Re-run the same known-owner checks against the fixed fixture.

## Evidence

Outbound requests: 3 / 12

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `c94fd46b5a58e0a1d113b0dd7381a082` | alice | 0 | true | destination is outside the approved origin |
| `39ccd705bd3e5f14df0de726c620a692` | alice | 200 | false |  |
| `7ef5613ac150ffe0ca131126aa41c4ab` | bob | 200 | false |  |
| `4c9d49315398c6260d1f4207815d1fbf` | bob | 200 | false |  |

### Request c94fd46b5a58e0a1d113b0dd7381a082

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:09:01.859374914Z; finished: 2026-11-01T21:09:01.859379837Z

```json

```

### Request 39ccd705bd3e5f14df0de726c620a692

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:09:01.862497387Z; finished: 2026-11-01T21:09:01.877696176Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 7ef5613ac150ffe0ca131126aa41c4ab

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:09:01.881061717Z; finished: 2026-11-01T21:09:01.894555008Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 4c9d49315398c6260d1f4207815d1fbf

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:09:01.898612054Z; finished: 2026-11-01T21:09:01.908589642Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
