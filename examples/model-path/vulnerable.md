# Tsunami lab assessment

Run: `d67518489f7872737ff36cabcd27ddf8`

Mode: Responses wire fixture / no live model

Scenario: vulnerable

State: completed

This uses a local Responses wire fixture, not a live model. It tests the real adapter and Mecatl loop without paid inference.

Workflow: object-ownership/v1; provider: fixture; model: tsunami-wire-fixture.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 200 / 100; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

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
| `a1be5fe86850560bf3cba4e6f56f03d2` | alice | 0 | true | destination is outside the approved origin |
| `883c8f28e5be4361d95ddfe7726de61f` | alice | 200 | false |  |
| `bfebfb1e7ccf7c0235cf7604b0198021` | bob | 200 | false |  |
| `5d4847a2115635f329c8882f9cbb1caf` | bob | 200 | false |  |

### Request a1be5fe86850560bf3cba4e6f56f03d2

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T14:12:33.271370235Z; finished: 2026-11-01T14:12:33.271371846Z

```json

```

### Request 883c8f28e5be4361d95ddfe7726de61f

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-01T14:12:33.281118096Z; finished: 2026-11-01T14:12:33.284490671Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request bfebfb1e7ccf7c0235cf7604b0198021

GET http://tsunami-vulnerable:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-01T14:12:33.296874805Z; finished: 2026-11-01T14:12:33.299831763Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 5d4847a2115635f329c8882f9cbb1caf

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-01T14:12:33.312015031Z; finished: 2026-11-01T14:12:33.314177411Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
