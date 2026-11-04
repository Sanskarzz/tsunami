# Tsunami lab assessment

Run: `553e5195e3879d82541d7ac0429a66fd`

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
| `4779cdfbfea3186e2761814cd721e778` | alice | 0 | true | destination is outside the approved origin |
| `c0b59a8aba666874b341ea6776268390` | alice | 200 | false |  |
| `6743c99a133712e14a98a5e79308845a` | bob | 200 | false |  |
| `f769bd87073003c60edd355394614b81` | bob | 403 | false |  |

### Request 4779cdfbfea3186e2761814cd721e778

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:09:02.890328823Z; finished: 2026-11-01T21:09:02.890332566Z

```json

```

### Request c0b59a8aba666874b341ea6776268390

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.50.17:8080

Started: 2026-11-01T21:09:02.893467732Z; finished: 2026-11-01T21:09:02.90578076Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 6743c99a133712e14a98a5e79308845a

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-01T21:09:02.909286392Z; finished: 2026-11-01T21:09:02.921393475Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request f769bd87073003c60edd355394614b81

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-01T21:09:02.92546951Z; finished: 2026-11-01T21:09:02.938810379Z

```json
{"error":"forbidden"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
