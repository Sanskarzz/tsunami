# Tsunami lab assessment

Run: `bb418201421c0d98ba24a95d1533b01f`

Mode: scripted Mecatl / synthetic lab only

Scenario: inconclusive

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

inconclusive: Cross-user document access

Both known-owner baselines and a conclusive cross-user response are required.

## Evidence

Outbound requests: 3 / 12

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `22cace25a900b926ca86a9694bff29ad` | alice | 0 | true | destination is outside the approved origin |
| `ae1ea1a48a3309fe8547ba7fd17a39c9` | alice | 200 | false |  |
| `e0d61ada1319a0c168e54334052ab473` | bob | 200 | false |  |
| `ee2c498617ed46e0f91d91128586cbe3` | bob | 404 | false |  |

### Request 22cace25a900b926ca86a9694bff29ad

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:15:35.545882521Z; finished: 2026-11-01T10:15:35.545884341Z

```json

```

### Request ae1ea1a48a3309fe8547ba7fd17a39c9

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.96.118.32:8080

Started: 2026-11-01T10:15:35.547768719Z; finished: 2026-11-01T10:15:35.5502257Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request e0d61ada1319a0c168e54334052ab473

GET http://tsunami-vulnerable:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.96.118.32:8080

Started: 2026-11-01T10:15:35.552183766Z; finished: 2026-11-01T10:15:35.554103473Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request ee2c498617ed46e0f91d91128586cbe3

GET http://tsunami-vulnerable:8080/api/documents/unknown

Account alias: bob; connected peer: 10.96.118.32:8080

Started: 2026-11-01T10:15:35.555674205Z; finished: 2026-11-01T10:15:35.557673153Z

```json
{"error":"not found"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, live model, general scanner integration, or production claim is included.
