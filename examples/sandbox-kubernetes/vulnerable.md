# Tsunami lab assessment

Run: `4e99d809533cee1f8a7ae0c4a2ed1971`

Mode: scripted Mecatl / synthetic lab only

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
| `52edcc44d0264b6ecc7790112e014bb0` | alice | 0 | true | destination is outside the approved origin |
| `dcaad283087a01c88c8f417d08595b84` | alice | 200 | false |  |
| `0843ef37c2d2c17611558d47f4b0d164` | bob | 200 | false |  |
| `8e91efa4773e7e4cb6b6d94f6a3b4bd5` | bob | 200 | false |  |

### Request 52edcc44d0264b6ecc7790112e014bb0

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:45:21.555283686Z; finished: 2026-11-01T10:45:21.555288699Z

```json

```

### Request dcaad283087a01c88c8f417d08595b84

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-01T10:45:21.55799556Z; finished: 2026-11-01T10:45:21.561162921Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 0843ef37c2d2c17611558d47f4b0d164

GET http://tsunami-vulnerable:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-01T10:45:21.564206764Z; finished: 2026-11-01T10:45:21.566624741Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 8e91efa4773e7e4cb6b6d94f6a3b4bd5

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-01T10:45:21.568927936Z; finished: 2026-11-01T10:45:21.572019272Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, live model, general scanner integration, or production claim is included.
