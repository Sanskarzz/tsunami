# Tsunami lab assessment

Run: `0cf2e09c99314e5ccb1f98bde1a3446e`

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
| `d893d2723ef18cf53502f5a1e9c1b698` | alice | 0 | true | destination is outside the approved origin |
| `b3540b4e58258487411acd36ce947e25` | alice | 302 | false |  |
| `e6a93bf939802ff5e29a9907aeb67ff4` | alice | 0 | true | destination is outside the approved origin |

### Request d893d2723ef18cf53502f5a1e9c1b698

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:15:36.580805285Z; finished: 2026-11-01T10:15:36.580809353Z

```json

```

### Request b3540b4e58258487411acd36ce947e25

GET http://tsunami-vulnerable:8080/api/redirect

Account alias: alice; connected peer: 10.96.118.32:8080

Started: 2026-11-01T10:15:36.582968372Z; finished: 2026-11-01T10:15:36.586387064Z

```json

```

### Request e6a93bf939802ff5e29a9907aeb67ff4

GET http://tsunami-tripwire:8080/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T10:15:36.588321544Z; finished: 2026-11-01T10:15:36.588323876Z

```json

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, live model, general scanner integration, or production claim is included.
