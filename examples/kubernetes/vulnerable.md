# Tsunami lab assessment

Run: `f14d987c18593da652b89e06a9febb67`

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
| `3e22fb3e9f042caf73177cfae00823db` | alice | 0 | true | destination is outside the approved origin |
| `166fe4f2b8433023e7a6720775b32bb2` | alice | 200 | false |  |
| `be489791a7d3b462c0650a5db172cb76` | bob | 200 | false |  |
| `94cdb46b5a0a8b567321a38af4a2aa35` | bob | 200 | false |  |

### Request 3e22fb3e9f042caf73177cfae00823db

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:15:33.852064626Z; finished: 2026-11-01T10:15:33.852066522Z

```json

```

### Request 166fe4f2b8433023e7a6720775b32bb2

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.96.118.32:8080

Started: 2026-11-01T10:15:33.853793631Z; finished: 2026-11-01T10:15:33.85638416Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request be489791a7d3b462c0650a5db172cb76

GET http://tsunami-vulnerable:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.96.118.32:8080

Started: 2026-11-01T10:15:33.858178155Z; finished: 2026-11-01T10:15:33.860353399Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 94cdb46b5a0a8b567321a38af4a2aa35

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.96.118.32:8080

Started: 2026-11-01T10:15:33.862537139Z; finished: 2026-11-01T10:15:33.864603483Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, live model, general scanner integration, or production claim is included.
