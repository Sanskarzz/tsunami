# Tsunami lab assessment

Run: `4e167faa2c6a7a9e831b999d6e23dcc7`

Mode: scripted Mecatl / synthetic lab only

Scenario: redirect

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

inconclusive: Cross-user document access

Both known-owner baselines and a conclusive cross-user response are required.

## Evidence

Outbound requests: 1 / 12

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `88fd2214e60bc4186e1801dbb1511742` | alice | 0 | true | destination is outside the approved origin |
| `b15e5ccd1cf1f8f490c31664d0d82d87` | alice | 302 | false |  |
| `2b8e8dcf920b7a6e76313b298a5d4546` | alice | 0 | true | destination is outside the approved origin |

### Request 88fd2214e60bc4186e1801dbb1511742

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:45:25.150208107Z; finished: 2026-11-01T10:45:25.150210468Z

```json

```

### Request b15e5ccd1cf1f8f490c31664d0d82d87

GET http://tsunami-vulnerable:8080/api/redirect

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-01T10:45:25.155357891Z; finished: 2026-11-01T10:45:25.168387504Z

```json

```

### Request 2b8e8dcf920b7a6e76313b298a5d4546

GET http://tsunami-tripwire:8080/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:45:25.178865187Z; finished: 2026-11-01T10:45:25.178868172Z

```json

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, live model, general scanner integration, or production claim is included.
