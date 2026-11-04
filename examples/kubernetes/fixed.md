# Tsunami lab assessment

Run: `32a8be0a3e71e871d1de1874ede25fcf`

Mode: scripted Mecatl / synthetic lab only

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
| `4d09033481a25568be1be35b059b707b` | alice | 0 | true | destination is outside the approved origin |
| `52629968144ac77a8f5f09870dcd1766` | alice | 200 | false |  |
| `69d89a0a0dbf1004444f769a3a74c456` | bob | 200 | false |  |
| `461a3e246dfdd60f05f2c22184ce2164` | bob | 403 | false |  |

### Request 4d09033481a25568be1be35b059b707b

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:15:34.619069036Z; finished: 2026-11-01T10:15:34.619072801Z

```json

```

### Request 52629968144ac77a8f5f09870dcd1766

GET http://tsunami-fixed:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.96.13.26:8080

Started: 2026-11-01T10:15:34.621534132Z; finished: 2026-11-01T10:15:34.625591388Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 69d89a0a0dbf1004444f769a3a74c456

GET http://tsunami-fixed:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.96.13.26:8080

Started: 2026-11-01T10:15:34.627777554Z; finished: 2026-11-01T10:15:34.630205392Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 461a3e246dfdd60f05f2c22184ce2164

GET http://tsunami-fixed:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.96.13.26:8080

Started: 2026-11-01T10:15:34.631946369Z; finished: 2026-11-01T10:15:34.634668513Z

```json
{"error":"forbidden"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, live model, general scanner integration, or production claim is included.
