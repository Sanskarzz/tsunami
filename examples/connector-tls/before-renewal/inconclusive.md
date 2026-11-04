# Tsunami lab assessment

Run: `2f66fa0c8b93a6bc3433cdfb01842528`

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
| `b39cbb4456ab929f9eeda88567c5a1fb` | alice | 0 | true | destination is outside the approved origin |
| `9ce14c62b1e43fef61a87a73298f617e` | alice | 200 | false |  |
| `4ef0758895d6c1a088f4bb7a17990e33` | bob | 200 | false |  |
| `424135c3423481e45e682968a7b747e9` | bob | 404 | false |  |

### Request b39cbb4456ab929f9eeda88567c5a1fb

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:09:03.917672678Z; finished: 2026-11-01T21:09:03.917676245Z

```json

```

### Request 9ce14c62b1e43fef61a87a73298f617e

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:09:03.920760694Z; finished: 2026-11-01T21:09:03.933147982Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 4ef0758895d6c1a088f4bb7a17990e33

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:09:03.936148962Z; finished: 2026-11-01T21:09:03.947377258Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 424135c3423481e45e682968a7b747e9

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/documents/unknown

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:09:03.950680296Z; finished: 2026-11-01T21:09:03.962481726Z

```json
{"error":"not found"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
