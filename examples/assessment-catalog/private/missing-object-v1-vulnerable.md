# Tsunami lab assessment

Run: `ffca288437f355acdf805b484dceaa13`

Mode: scripted Mecatl / synthetic lab only

Scenario: vulnerable

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

inconclusive: Missing-object negative control

The probe uses an object with no confirmed existence. Its rejection cannot prove ownership enforcement.

Procedure: `missing-object/v1`; digest: `7677e75cb7264c9c51f4d51483b729d1e04a5af30a8c1725477160950cfc09c6`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: unknown

Missing-evidence feedback issued: 0 / 2

## Reproduction

1. Read `/api/reviewed/objects/r-7f3c9a` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/objects/r-missing-99e` using the `bob` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Use a known existing private object and approved second account before making an authorization conclusion.

## Evidence

Requests reserved: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `fcc7f76cf48ee4cd25b079b95340ed9f` | alice | 0 | true | destination is outside the approved origin |
| `132c1400d0e4a378a69fc7de6149d21f` | alice | 200 | false |  |
| `b338261df5077b66102359d724481182` | bob | 200 | false |  |
| `ee2b42144ee3cd233982e5fee765115c` | bob | 404 | false |  |

### Request fcc7f76cf48ee4cd25b079b95340ed9f

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:48.838964886Z; finished: 2026-11-02T07:39:48.838966391Z

```json

```

### Request 132c1400d0e4a378a69fc7de6149d21f

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:48.847093188Z; finished: 2026-11-02T07:39:48.859962261Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request b338261df5077b66102359d724481182

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:48.863573166Z; finished: 2026-11-02T07:39:48.871839607Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request ee2b42144ee3cd233982e5fee765115c

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-missing-99e

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:48.875229558Z; finished: 2026-11-02T07:39:48.886442105Z

```json
{"error":"not found"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
