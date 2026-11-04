# Tsunami lab assessment

Run: `957e0ff21a276e7dc74bd984617ebfb9`

Mode: OpenRouter free model / synthetic lab only

Scenario: vulnerable

State: completed

A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.

Workflow: object-ownership/v1; provider: openrouter-free; model: nvidia/nemotron-3.5-lightning:free.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 5679 / 228; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

Price policy: openrouter-free/v1 (free model required; zero price ceilings; provider/model fallback disabled).

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

Outbound requests: 4 / 12

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `631ecf00f875e2a04b5905d513656731` | alice | 200 | false |  |
| `48dcf82afbf88a47e6671da30cc3c44b` | bob | 200 | false |  |
| `210e775d703d41973c572290a3aa229e` | bob | 200 | false |  |
| `4477378392958ca6a72a753740635171` | alice | 200 | false |  |

### Request 631ecf00f875e2a04b5905d513656731

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-01T15:03:50.620171215Z; finished: 2026-11-01T15:03:50.623527976Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 48dcf82afbf88a47e6671da30cc3c44b

GET http://tsunami-vulnerable:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-01T15:03:51.879866405Z; finished: 2026-11-01T15:03:51.882621159Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 210e775d703d41973c572290a3aa229e

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-01T15:03:53.313922619Z; finished: 2026-11-01T15:03:53.317089482Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 4477378392958ca6a72a753740635171

GET http://tsunami-vulnerable:8080/api/documents/doc-bob

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-01T15:03:54.591067311Z; finished: 2026-11-01T15:03:54.594353337Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
