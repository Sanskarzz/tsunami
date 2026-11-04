# Tsunami lab assessment

Run: `e5f274e512b83af022028598ad3e5578`

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
| `10b3dd7e6022d2096f6ca211eeeedb59` | alice | 0 | true | destination is outside the approved origin |
| `66f46bdac98cf25a9041fefde03b1968` | alice | 200 | false |  |
| `24896a61f5142fedff5b55cee80a5cd6` | bob | 200 | false |  |
| `e39c75b821f46821e084af9ce09539f4` | bob | 404 | false |  |

### Request 10b3dd7e6022d2096f6ca211eeeedb59

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T18:17:06.394111469Z; finished: 2026-11-01T18:17:06.394115646Z

```json

```

### Request 66f46bdac98cf25a9041fefde03b1968

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T18:17:06.397623023Z; finished: 2026-11-01T18:17:06.410032969Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 24896a61f5142fedff5b55cee80a5cd6

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T18:17:06.413918572Z; finished: 2026-11-01T18:17:06.422995789Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request e39c75b821f46821e084af9ce09539f4

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/unknown

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T18:17:06.426555689Z; finished: 2026-11-01T18:17:06.436899605Z

```json
{"error":"not found"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
