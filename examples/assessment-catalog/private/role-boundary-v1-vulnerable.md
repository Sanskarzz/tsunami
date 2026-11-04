# Tsunami lab assessment

Run: `8a410ce0b0db80a51c574a1513bd1219`

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
| `8c0df844976591425275e139b320134a` | alice | 0 | true | destination is outside the approved origin |
| `cf59aae4963744e217c21ad03b10e07f` | alice | 200 | false |  |
| `c56d2c24c780756bf25c2a7c70032082` | bob | 200 | false |  |
| `52c22b0b579609592f45e0b2b84acaf7` | bob | 200 | false |  |

### Request 8c0df844976591425275e139b320134a

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:52.757283401Z; finished: 2026-11-02T07:39:52.757290001Z

```json

```

### Request cf59aae4963744e217c21ad03b10e07f

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/admin/summary

Account alias: alice; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:52.764091461Z; finished: 2026-11-02T07:39:52.776631643Z

```json
{"id":"admin-summary","owner":"administrators","content":"synthetic administrator audit summary"}

```

### Request c56d2c24c780756bf25c2a7c70032082

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:52.78009127Z; finished: 2026-11-02T07:39:52.789712333Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 52c22b0b579609592f45e0b2b84acaf7

GET http://tsunami-vulnerable.tsunami-customer-a.svc.cluster.local:8080/api/reviewed/admin/summary

Account alias: bob; connected peer: 10.97.214.163:8080

Started: 2026-11-02T07:39:52.7924843Z; finished: 2026-11-02T07:39:52.799867545Z

```json
{"id":"admin-summary","owner":"administrators","content":"synthetic administrator audit summary"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
