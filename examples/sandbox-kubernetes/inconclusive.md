# Tsunami lab assessment

Run: `ab5f320a465f52939d808ca8130216f6`

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
| `d01db65dcfe9bddc3db9f95cb82eee5f` | alice | 0 | true | destination is outside the approved origin |
| `e9bee2c1f621b1fcd1177f5ebd862c06` | alice | 200 | false |  |
| `a979f61f3a2d30c541ebc496a21ac366` | bob | 200 | false |  |
| `d0786e0aee799e6a32c4dbfac627cbab` | bob | 404 | false |  |

### Request d01db65dcfe9bddc3db9f95cb82eee5f

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:45:23.749959821Z; finished: 2026-11-01T10:45:23.749964052Z

```json

```

### Request e9bee2c1f621b1fcd1177f5ebd862c06

GET http://tsunami-vulnerable:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-01T10:45:23.762682743Z; finished: 2026-11-01T10:45:23.77419764Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request a979f61f3a2d30c541ebc496a21ac366

GET http://tsunami-vulnerable:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-01T10:45:23.786301378Z; finished: 2026-11-01T10:45:23.798562821Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request d0786e0aee799e6a32c4dbfac627cbab

GET http://tsunami-vulnerable:8080/api/documents/unknown

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-01T10:45:23.81075177Z; finished: 2026-11-01T10:45:23.822650806Z

```json
{"error":"not found"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, live model, general scanner integration, or production claim is included.
