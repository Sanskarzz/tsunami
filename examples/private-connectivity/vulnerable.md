# Tsunami lab assessment

Run: `6ceff5d3c6517b45689c58824e7a248d`

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
| `ec703787085ca9ca9aafe228c31a87bb` | alice | 0 | true | destination is outside the approved origin |
| `9fdfe1e17be80dbaa198d3bbb5fa2407` | alice | 200 | false |  |
| `85caae7c854e8c0cfc6957e4c969e5ff` | bob | 200 | false |  |
| `d6e86737165b5eb6b19ed731a08dfbd3` | bob | 200 | false |  |

### Request ec703787085ca9ca9aafe228c31a87bb

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T18:17:04.193554259Z; finished: 2026-11-01T18:17:04.193555917Z

```json

```

### Request 9fdfe1e17be80dbaa198d3bbb5fa2407

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T18:17:04.195301714Z; finished: 2026-11-01T18:17:04.202899598Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 85caae7c854e8c0cfc6957e4c969e5ff

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T18:17:04.205132844Z; finished: 2026-11-01T18:17:04.210918768Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request d6e86737165b5eb6b19ed731a08dfbd3

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T18:17:04.213431307Z; finished: 2026-11-01T18:17:04.221339285Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
