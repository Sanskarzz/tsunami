# Tsunami lab assessment

Run: `f0bb1dc8b27b5934e35aca08906a41d1`

Mode: scripted Mecatl / synthetic lab only

Scenario: vulnerable

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

confirmed: Member access to an administrator summary

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

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
| `045c73ec696bd81b82f79dc38c4329ed` | alice | 0 | true | destination is outside the approved origin |
| `3e12959e24cdb604a9016c1dea653d14` | alice | 200 | false |  |
| `141f4d37a67758db815a3a4d44bfd4fc` | bob | 200 | false |  |
| `c0f782dc7fe760d2f812f463a93df5cd` | bob | 200 | false |  |

### Request 045c73ec696bd81b82f79dc38c4329ed

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:52.731561555Z; finished: 2026-11-02T07:39:52.731565191Z

```json

```

### Request 3e12959e24cdb604a9016c1dea653d14

GET http://tsunami-vulnerable:8080/api/reviewed/admin/summary

Account alias: alice; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:52.734088806Z; finished: 2026-11-02T07:39:52.737212258Z

```json
{"id":"admin-summary","owner":"administrators","content":"synthetic administrator audit summary"}

```

### Request 141f4d37a67758db815a3a4d44bfd4fc

GET http://tsunami-vulnerable:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:52.741576745Z; finished: 2026-11-02T07:39:52.745719565Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request c0f782dc7fe760d2f812f463a93df5cd

GET http://tsunami-vulnerable:8080/api/reviewed/admin/summary

Account alias: bob; connected peer: 10.97.190.42:8080

Started: 2026-11-02T07:39:52.749169554Z; finished: 2026-11-02T07:39:52.755977335Z

```json
{"id":"admin-summary","owner":"administrators","content":"synthetic administrator audit summary"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
