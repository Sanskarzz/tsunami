# Tsunami lab assessment

Run: `e54f406239cde725118b714c06872ca9`

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
| `9548c7a83e6777aa264e947c29a808e5` | alice | 0 | true | destination is outside the approved origin |
| `f03dae0165f5064d7e8b867f6b46e089` | alice | 200 | false |  |
| `0e1851361ca51bfe97acdd562d712c22` | bob | 200 | false |  |
| `1a08cc418f9b069c1ad31b996dc4cb66` | bob | 200 | false |  |

### Request 9548c7a83e6777aa264e947c29a808e5

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:11:50.426398388Z; finished: 2026-11-01T21:11:50.426404223Z

```json

```

### Request f03dae0165f5064d7e8b867f6b46e089

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:11:50.429091151Z; finished: 2026-11-01T21:11:50.437263642Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 0e1851361ca51bfe97acdd562d712c22

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:11:50.439630223Z; finished: 2026-11-01T21:11:50.448800952Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 1a08cc418f9b069c1ad31b996dc4cb66

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:11:50.451505628Z; finished: 2026-11-01T21:11:50.460220881Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
