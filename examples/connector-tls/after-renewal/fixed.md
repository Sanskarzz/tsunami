# Tsunami lab assessment

Run: `6d9fa98cb360ef3460da6971fdf27dbd`

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
| `fd64692e938da97c6cd33c22e2a3f1a6` | alice | 0 | true | destination is outside the approved origin |
| `cb4e976ca3a2fced768be0770d99f5d0` | alice | 200 | false |  |
| `6f52de340ac3eb7c713e8ac6e0c06a48` | bob | 200 | false |  |
| `3f89ce6ea41670ffccd3598234652289` | bob | 403 | false |  |

### Request fd64692e938da97c6cd33c22e2a3f1a6

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:11:51.403058534Z; finished: 2026-11-01T21:11:51.403062664Z

```json

```

### Request cb4e976ca3a2fced768be0770d99f5d0

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: alice; connected peer: 10.97.50.17:8080

Started: 2026-11-01T21:11:51.434042724Z; finished: 2026-11-01T21:11:51.447176851Z

```json
{"content":"synthetic private invoice for alice","id":"doc-alice","owner":"alice"}

```

### Request 6f52de340ac3eb7c713e8ac6e0c06a48

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-bob

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-01T21:11:51.451004536Z; finished: 2026-11-01T21:11:51.464367595Z

```json
{"content":"synthetic private invoice for bob","id":"doc-bob","owner":"bob"}

```

### Request 3f89ce6ea41670ffccd3598234652289

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/documents/doc-alice

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-01T21:11:51.468283777Z; finished: 2026-11-01T21:11:51.496094063Z

```json
{"error":"forbidden"}

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
