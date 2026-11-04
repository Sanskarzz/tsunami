# Tsunami lab assessment

Run: `b15a52619886e7320315e0a97307e4cc`

Mode: OpenRouter free model / synthetic lab only

Scenario: fixed

State: completed

A live model selected tools for a reviewed synthetic workflow. This is not general autonomous vulnerability discovery or a customer assessment.

Workflow: opaque-ownership/v1; provider: openrouter-free; model: nvidia/nemotron-3.5-lightning:free.

Model calls reserved: 4 / 8; upstream attempts: 4 / 8. Reported input/output tokens: 4997 / 189; usage uncertain: false. Reservations are retained after ambiguous failures; these counters are not a monetary spending guarantee.

Price policy: openrouter-free/v1 (free model required; zero price ceilings; provider/model fallback disabled).

## Result

no_issue_observed: Cross-user access to an opaque object ID

Known immutable fixture baselines succeeded and the prohibited read returned HTTP 404. This applies only to this procedure, not application-wide security.

Procedure: `opaque-ownership/v1`; digest: `414b6dd1126f39b8a01d8d52f82ac7f31d8d33d57847150be36a9f13ec6764d5`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: deny

## Reproduction

1. Read `/api/reviewed/objects/r-7f3c9a` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/objects/r-7f3c9a` using the `bob` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Check object ownership against the authenticated account on every object read. Opaque IDs alone do not authorize access.

## Evidence

Outbound requests: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `4171e446dc66bbccf1c0f0fc36e63229` | bob | 404 | false |  |
| `f167f50f820f1325775b7f1b4441a108` | alice | 200 | false |  |
| `94ec827ff56a8b4f05667fe573671e42` | bob | 200 | false |  |

### Request 4171e446dc66bbccf1c0f0fc36e63229

GET http://tsunami-fixed:8080/api/reviewed/objects/r-7f3c9a

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:26:34.28448671Z; finished: 2026-11-02T07:26:34.288047039Z

```json
{"error":"not found"}

```

### Request f167f50f820f1325775b7f1b4441a108

GET http://tsunami-fixed:8080/api/reviewed/objects/r-7f3c9a

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:26:35.923406529Z; finished: 2026-11-02T07:26:35.936485362Z

```json
{"id":"r-7f3c9a","owner":"alice","content":"synthetic private ledger A"}

```

### Request 94ec827ff56a8b4f05667fe573671e42

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:26:37.557767976Z; finished: 2026-11-02T07:26:37.569760388Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
