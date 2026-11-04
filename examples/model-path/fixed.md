# Tsunami lab assessment

Run: `dc5e86090dbbb046b246d20b99f15f64`

Mode: Responses wire fixture / no live model

Scenario: fixed

State: completed

This uses a local Responses wire fixture, not a live model. It tests the real adapter and Mecatl loop without paid inference.

Workflow: object-ownership/v1; provider: fixture; model: tsunami-wire-fixture.

Model calls reserved: 5 / 8; upstream attempts: 5 / 8. Reported input/output tokens: 200 / 100; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

## Result

no_issue_observed: Cross-user document access

The two known-owner baselines succeeded; the cross-user read was forbidden. This is not proof of application-wide security.

## Evidence

Outbound requests: 3 / 12

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `494242057eac319be6d882a153e888c9` | alice | 0 | true | destination is outside the approved origin |
| `211711005701837ef19ab3711343e4f9` | alice | 200 | false |  |
| `5f2425aa5232d12dfeb013318c387f17` | bob | 200 | false |  |
| `d40d013451cd778fcafdf0e8e5cec1d0` | bob | 403 | false |  |

### Request 494242057eac319be6d882a153e888c9

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T14:12:34.4819934Z; finished: 2026-11-01T14:12:34.481995115Z

```json

```

### Request 211711005701837ef19ab3711343e4f9

GET http://tsunami-fixed:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-01T14:12:34.491765757Z; finished: 2026-11-01T14:12:34.493756295Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 5f2425aa5232d12dfeb013318c387f17

GET http://tsunami-fixed:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-01T14:12:34.502042183Z; finished: 2026-11-01T14:12:34.50427873Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request d40d013451cd778fcafdf0e8e5cec1d0

GET http://tsunami-fixed:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-01T14:12:34.512581979Z; finished: 2026-11-01T14:12:34.51466778Z

```json
{"error":"forbidden"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
