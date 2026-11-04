# Tsunami lab assessment

Run: `f8865e398c48174b2ef0e8ad4b7e2fae`

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
| `14257b24b4b3b1222b07e052dfaff43d` | alice | 0 | true | destination is outside the approved origin |
| `f2090edef2bd914aac15a205ef02dd8b` | alice | 200 | false |  |
| `90bb34f145a3af6c7e65e06ff7146b3e` | bob | 200 | false |  |
| `69d96e03f9413d8bcc6345cf75faa868` | bob | 403 | false |  |

### Request 14257b24b4b3b1222b07e052dfaff43d

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:53.76540237Z; finished: 2026-11-02T07:39:53.765406831Z

```json

```

### Request f2090edef2bd914aac15a205ef02dd8b

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/admin/summary

Account alias: alice; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:53.76844996Z; finished: 2026-11-02T07:39:53.776876476Z

```json
{"id":"admin-summary","owner":"administrators","content":"synthetic administrator audit summary"}

```

### Request 90bb34f145a3af6c7e65e06ff7146b3e

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:53.779606624Z; finished: 2026-11-02T07:39:53.792618103Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 69d96e03f9413d8bcc6345cf75faa868

GET http://tsunami-fixed.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/admin/summary

Account alias: bob; connected peer: 10.97.50.17:8080

Started: 2026-11-02T07:39:53.796592363Z; finished: 2026-11-02T07:39:53.808785584Z

```json
{"error":"forbidden"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
