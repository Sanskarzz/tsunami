# Tsunami lab assessment

Run: `55eb384fe63f4dafb4dea97c208a3059`

Mode: scripted Mecatl / synthetic lab only / private connector

Scenario: fixed

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

no_issue_observed: Cross-user document access

The two known-owner baselines succeeded; the cross-user read was forbidden. This is not proof of application-wide security.

## Evidence

Outbound requests: 3 / 12

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `a73da56b8cbfdaa55a7e99544ff9b1bf` | alice | 0 | true | destination is outside the approved origin |
| `8d11b482b1450acd49f75d1b287f996f` | alice | 200 | false |  |
| `6c3c260492080a2b097b1a3308b1399d` | bob | 200 | false |  |
| `e2888a7549ca8369880df4475e499804` | bob | 403 | false |  |

### Request a73da56b8cbfdaa55a7e99544ff9b1bf

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T18:17:05.410189414Z; finished: 2026-11-01T18:17:05.410194536Z

```json

```

### Request 8d11b482b1450acd49f75d1b287f996f

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.50.17:8080

Started: 2026-11-01T18:17:05.41405671Z; finished: 2026-11-01T18:17:05.427055674Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 6c3c260492080a2b097b1a3308b1399d

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-01T18:17:05.432227489Z; finished: 2026-11-01T18:17:05.444107065Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request e2888a7549ca8369880df4475e499804

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-01T18:17:05.447090436Z; finished: 2026-11-01T18:17:05.453810282Z

```json
{"error":"forbidden"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
