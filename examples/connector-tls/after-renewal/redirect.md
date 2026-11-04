# Tsunami lab assessment

Run: `26fbf9454c012646e7ea22d4ba788a01`

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
| `3427415e635e013c42768bf6c99a851e` | alice | 0 | true | destination is outside the approved origin |
| `8d370af2bd95ce69b89d01e3ad1efda7` | alice | 302 | false |  |
| `9a112a59fa220b09d9ff01816823fe61` | alice | 0 | true | destination is outside the approved origin |

### Request 3427415e635e013c42768bf6c99a851e

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:11:53.37544652Z; finished: 2026-11-01T21:11:53.375448878Z

```json

```

### Request 8d370af2bd95ce69b89d01e3ad1efda7

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/redirect

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-01T21:11:53.377290499Z; finished: 2026-11-01T21:11:53.384289155Z

```json

```

### Request 9a112a59fa220b09d9ff01816823fe61

GET http://tsunami-tripwire:8080/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-01T21:11:53.385784858Z; finished: 2026-11-01T21:11:53.385786286Z

```json

```

## Limits

The validator knows this fixture's ownership model. Credentials and returned documents are synthetic. No arbitrary shell tool, private customer connector, general scanner integration, or production claim is included.
