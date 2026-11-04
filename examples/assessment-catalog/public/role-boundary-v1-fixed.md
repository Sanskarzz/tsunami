# Tsunami lab assessment

Run: `6299ab27d810b1c2896ca3b125713e12`

Mode: scripted Mecatl / synthetic lab only

Scenario: fixed

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

no_issue_observed: Member access to an administrator summary

Known immutable fixture baselines succeeded and the prohibited read returned HTTP 403. This applies only to this procedure, not application-wide security.

Procedure: `role-boundary/v1`; digest: `a7ef0a78f172a2f6c96bd05a6980b366de8bc59b094eff09e15a611deecb829c`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: deny

Missing-evidence feedback issued: 0 / 2

## Reproduction

1. Read `/api/reviewed/admin/summary` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/admin/summary` using the `bob` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Enforce the administrator role on the server before returning the summary. A valid member session is insufficient.

## Evidence

Requests reserved: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `f1d965cfb579e03f38c441f7d3cafac0` | alice | 0 | true | destination is outside the approved origin |
| `dad9aeced1a7a1ce3eff0b50e515fe12` | alice | 200 | false |  |
| `4834f532a52d4f25360fe4fd1dbe22ce` | bob | 200 | false |  |
| `bd6936625875a2b4faf95bbfe543b191` | bob | 403 | false |  |

### Request f1d965cfb579e03f38c441f7d3cafac0

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:53.69497901Z; finished: 2026-11-02T07:39:53.694980804Z

```json

```

### Request dad9aeced1a7a1ce3eff0b50e515fe12

GET http://tsunami-fixed:8080/api/reviewed/admin/summary

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:53.697656212Z; finished: 2026-11-02T07:39:53.699845685Z

```json
{"id":"admin-summary","owner":"administrators","content":"synthetic administrator audit summary"}

```

### Request 4834f532a52d4f25360fe4fd1dbe22ce

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:53.702969461Z; finished: 2026-11-02T07:39:53.705278792Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request bd6936625875a2b4faf95bbfe543b191

GET http://tsunami-fixed:8080/api/reviewed/admin/summary

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:53.70770714Z; finished: 2026-11-02T07:39:53.710078558Z

```json
{"error":"forbidden"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
