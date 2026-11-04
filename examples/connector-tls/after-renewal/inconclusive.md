# Tsunami lab assessment

Run: `989f7e772a0496379a0884ada6a65a89`

Mode: scripted Mecatl / synthetic lab only / private connector

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
| `9eda53be887e576b6b2683ee6a783e80` | alice | 0 | true | destination is outside the approved origin |
| `b1fcd6875b86446c75dd8ce9254735bc` | alice | 200 | false |  |
| `bf491c4e152680ef12533e514be21e94` | bob | 200 | false |  |
| `224de5af7b143828b031206531c43736` | bob | 404 | false |  |

### Request 9eda53be887e576b6b2683ee6a783e80

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:11:52.376202776Z; finished: 2026-11-01T21:11:52.376206822Z

```json

```

### Request b1fcd6875b86446c75dd8ce9254735bc

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:11:52.379014901Z; finished: 2026-11-01T21:11:52.38526475Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request bf491c4e152680ef12533e514be21e94

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:11:52.388517382Z; finished: 2026-11-01T21:11:52.396936785Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 224de5af7b143828b031206531c43736

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/unknown

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:11:52.399385685Z; finished: 2026-11-01T21:11:52.409932681Z

```json
{"error":"not found"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
