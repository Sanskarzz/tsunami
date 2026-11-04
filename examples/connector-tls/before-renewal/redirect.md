# Tsunami lab assessment

Run: `c5b4824298cb6d5552b0295ca53641c7`

Mode: scripted Mecatl / synthetic lab only / private connector

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
| `ef6c8687246dd9c76726d62bf5fe9afc` | alice | 0 | true | destination is outside the approved origin |
| `e3b766285bb9e4b460c9fef6c072af1c` | alice | 302 | false |  |
| `53dcf2bf95fd983b673c1ea89fc53921` | alice | 0 | true | destination is outside the approved origin |

### Request ef6c8687246dd9c76726d62bf5fe9afc

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:09:04.780930647Z; finished: 2026-11-01T21:09:04.780932589Z

```json

```

### Request e3b766285bb9e4b460c9fef6c072af1c

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/redirect

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:09:04.814024795Z; finished: 2026-11-01T21:09:04.823414328Z

```json

```

### Request 53dcf2bf95fd983b673c1ea89fc53921

GET http://tsunami-tripwire:8080/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:09:04.825263053Z; finished: 2026-11-01T21:09:04.825264605Z

```json

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
