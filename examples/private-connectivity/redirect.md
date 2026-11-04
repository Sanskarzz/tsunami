# Tsunami lab assessment

Run: `d12b1f8192fda8631cc844977d653d6a`

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
| `c305110aad370166e270677fd43b4135` | alice | 0 | true | destination is outside the approved origin |
| `d391bf9548792f07a4ce4ce59f5ec842` | alice | 302 | false |  |
| `16e5414f02aafff266aec86a1096bc85` | alice | 0 | true | destination is outside the approved origin |

### Request c305110aad370166e270677fd43b4135

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T18:17:07.272063378Z; finished: 2026-11-01T18:17:07.272065663Z

```json

```

### Request d391bf9548792f07a4ce4ce59f5ec842

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/redirect

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T18:17:07.274355025Z; finished: 2026-11-01T18:17:07.280046426Z

```json

```

### Request 16e5414f02aafff266aec86a1096bc85

GET http://tsunami-tripwire:8080/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T18:17:07.281529782Z; finished: 2026-11-01T18:17:07.28153113Z

```json

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
