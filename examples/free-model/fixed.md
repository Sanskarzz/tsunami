# Tsunami lab assessment

Run: `17854af34144c1929aa549dba4b96695`

Mode: OpenRouter free model / synthetic lab only

Scenario: fixed

State: completed

A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.

Workflow: object-ownership/v1; provider: openrouter-free; model: nvidia/nemotron-3.5-lightning:free.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 5537 / 216; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

Price policy: openrouter-free/v1 (free model required; zero price ceilings; provider/model fallback disabled).

## Result

no_issue_observed: Cross-user document access

The two known-owner baselines succeeded; the cross-user read was forbidden. This is not proof of application-wide security.

## Evidence

Outbound requests: 4 / 12

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `a14988a1bfa457a3db6ae3f7cbc88bb7` | alice | 200 | false |  |
| `c8ca64ae6d4b4f0ff3ffd07208b91e12` | bob | 200 | false |  |
| `c607d4f30335b458ac89e282c8cf9291` | bob | 403 | false |  |
| `869e454b8a20c7e3e917bc81f898247f` | alice | 403 | false |  |

### Request a14988a1bfa457a3db6ae3f7cbc88bb7

GET http://tsunami-fixed:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-01T15:03:58.540490176Z; finished: 2026-11-01T15:03:58.544305612Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request c8ca64ae6d4b4f0ff3ffd07208b91e12

GET http://tsunami-fixed:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-01T15:03:59.671807423Z; finished: 2026-11-01T15:03:59.675618005Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request c607d4f30335b458ac89e282c8cf9291

GET http://tsunami-fixed:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-01T15:04:00.68728387Z; finished: 2026-11-01T15:04:00.69067094Z

```json
{"error":"forbidden"}

```

### Request 869e454b8a20c7e3e917bc81f898247f

GET http://tsunami-fixed:8080/api/documents/doc-bob

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-01T15:04:02.12343302Z; finished: 2026-11-01T15:04:02.126799306Z

```json
{"error":"forbidden"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
