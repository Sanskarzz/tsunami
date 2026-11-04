# Tsunami lab assessment

Run: `f52e7d7d7ed7ef208b8a9acc96bbacf8`

Mode: scripted Mecatl / synthetic lab only

Scenario: fixed

State: completed

This is a synthetic lab with a scripted model driving the real Mecatl engine. It does not demonstrate autonomous vulnerability discovery or production tenant isolation.

## Result

no_issue_observed: Approved shared-object access

The captured shared-object response matches the explicitly approved access rule. This is a positive control, not a finding.

Procedure: `shared-access/v1`; digest: `df94143d286bea5ab5b89870b351aa929f0310652970d8d83d5403057544420f`

Review: initial engineering procedure; security-engineer review pending

Approved access rule: allow

Missing-evidence feedback issued: 0 / 2

## Reproduction

1. Read `/api/reviewed/objects/r-shared-40a` using the `alice` account alias.
2. Read `/api/reviewed/objects/r-b82d16` using the `bob` account alias.
3. Read `/api/reviewed/objects/r-shared-40a` using the `bob` account alias.

Compare the captured identity, owner, and content with the approved fixture facts. An HTTP 200 alone is insufficient.

## Remediation or next action

Preserve the explicit sharing rule. A successful approved share is not a security finding.

## Evidence

Requests reserved: 3 / 6

| Request | Account | HTTP | Blocked | Outcome |
|---|---|---|---|---|
| `5ae2de3795d9dcb687657cd5899445a4` | alice | 0 | true | destination is outside the approved origin |
| `1c3133faacafb3878cc77ddd0856cca0` | alice | 200 | false |  |
| `73a93ad657c5c3acf8c7db0d4179603d` | bob | 200 | false |  |
| `8e2419f9975ad5454fec785062e146b5` | bob | 200 | false |  |

### Request 5ae2de3795d9dcb687657cd5899445a4

GET http://not-approved.invalid/api/documents/doc-alice

Account alias: alice; connected peer: 

Started: 2026-11-02T07:39:55.785590979Z; finished: 2026-11-02T07:39:55.785593957Z

```json

```

### Request 1c3133faacafb3878cc77ddd0856cca0

GET http://tsunami-fixed:8080/api/reviewed/objects/r-shared-40a

Account alias: alice; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:55.788659547Z; finished: 2026-11-02T07:39:55.791126828Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

### Request 73a93ad657c5c3acf8c7db0d4179603d

GET http://tsunami-fixed:8080/api/reviewed/objects/r-b82d16

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:55.796596347Z; finished: 2026-11-02T07:39:55.801384287Z

```json
{"id":"r-b82d16","owner":"bob","content":"synthetic private ledger B"}

```

### Request 8e2419f9975ad5454fec785062e146b5

GET http://tsunami-fixed:8080/api/reviewed/objects/r-shared-40a

Account alias: bob; connected peer: 10.97.18.225:8080

Started: 2026-11-02T07:39:55.806112392Z; finished: 2026-11-02T07:39:55.81002121Z

```json
{"id":"r-shared-40a","owner":"alice","content":"synthetic team handover"}

```

## Limits

The validator uses known synthetic fixture rules. Credentials and returned documents are synthetic. The lab private connector is supported; real customer-network tunnels, arbitrary shell tools, general scanner integration, and production guarantees are not included.
