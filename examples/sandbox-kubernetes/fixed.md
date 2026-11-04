# Tsunami lab assessment

Run: `537b52aae6c933cb233e8e19e5c14815`

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
| `3a3acfbe59b46efef1a242d9d0520945` | alice | 0 | true | destination is outside the approved origin |
| `a658af8852c74f7012f5be2e280dd509` | alice | 200 | false |  |
| `afd3a7563b5edd68e4176a462d469c20` | bob | 200 | false |  |
| `ec63920b10fca20ca243a6967c0a9ebe` | bob | 403 | false |  |

### Request 3a3acfbe59b46efef1a242d9d0520945

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:45:22.571045195Z; finished: 2026-11-01T10:45:22.571049017Z

```json

```

### Request a658af8852c74f7012f5be2e280dd509

GET http://tsunami-fixed:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-01T10:45:22.580957432Z; finished: 2026-11-01T10:45:22.592453501Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request afd3a7563b5edd68e4176a462d469c20

GET http://tsunami-fixed:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-01T10:45:22.603916985Z; finished: 2026-11-01T10:45:22.615539453Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request ec63920b10fca20ca243a6967c0a9ebe

GET http://tsunami-fixed:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-01T10:45:22.629295744Z; finished: 2026-11-01T10:45:22.642650355Z

```json
{"error":"forbidden"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, live model, general scanner integration, or production claim is included.
