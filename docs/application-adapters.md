# Configured applications and reviewed exports

This milestone adds an application configuration path alongside the embedded fixtures. It tests an existing application, [VAmPI](https://github.com/erev0s/VAmPI), deployed in our own isolated lab. It does not test an unrelated public service or a customer's data.

VAmPI source is pinned to `f16052dce83f05847133ec98f01c5193a41de7d8` under its MIT license. It stays under ignored `.local/external/`; it is not copied into Tsunami. Our Dockerfile disables the upstream debugger/reloader and adds a packaging compatibility pin for Werkzeug. The recorded image identifies this build; transitive Python dependencies are not a production supply-chain qualification. VAmPI's fixed switch is a negative control for this access rule, not a claim that every other vulnerability is fixed.

## What is configured

A trusted startup file supplies a tenant, application ID, origin, explicit destination addresses, two bearer credentials, and a public read plan. Assessment callers select only an application registered for their tenant; workers cannot register a URL, choose credentials, modify selectors or provide a verdict. The subsequent [protected onboarding path](application-onboarding.md) can register supported synthetic bearer/JSON integrations after explicit prerequisite validation, then creates an unapproved assessment draft. `GET /v1/applications` returns public plans to the authenticated operator; workers cannot read that registry.

The [local preparation command](binding-preparation.md) can now derive the plan from supplied synthetic baselines and explicit selectors/rules. VAmPI setup captures legitimate responses and invokes it. That preparation does not register the application, assign credentials, approve scope or substitute for actual run evidence.

The generic matcher supports string fields selected by bounded JSON pointers in objects, not arrays, scripts or arbitrary expressions. The configuration supplies each account's identity assertion, two legitimate object reads, the cross-user probe, proof hashes, and the intended allow/deny rule. The VAmPI response fields are configuration, not a special branch in the validator.

This document describes `http-object-access/v2`; its procedure is `{application}/object-access/v2`. The separate [property-access profile](property-access.md) now checks field filtering in an allowed object response without changing this profile's semantics. The object-access sequence is:

1. Verify the owner identity.
2. Verify the peer identity.
3. Read the owner's object as owner.
4. Read the peer's object as peer.
5. Read the owner's object as peer.
6. Recheck the owner's object as owner.
7. Recheck the peer identity.

The final identity check prevents an expired peer session from masquerading as an authorization denial. The owner recheck reduces conclusions caused by a deleted object. These are black-box checks, not a transactional snapshot: concurrent application changes can still make results ambiguous. The intended authorization rule and identity endpoint must come from trusted application knowledge.

The validator uses the latest probe and requires the corresponding before/after evidence. It compares selected identity/object fields and a hash of the seeded private proof. A generic HTTP 200, missing prerequisite, wrong identity, duplicate-key JSON, interrupted response, deleted owner object or expired peer stays inconclusive. Configured denial statuses only count after the rechecks succeed.

Early development used a six-read v1 profile. Those local records remain untouched. V2 refuses unsupported old plans rather than silently applying stronger validation to them. Embedded fixture procedures and their historical evaluations retain their existing versions.

## Scope and credentials

The broker still performs origin/path/account checks, redirect validation, request reservations and validated numeric-address dialing. Application plans accept exact addresses; they do not turn a DNS answer into customer approval. Link-local destinations and the configured Kubernetes API Service address are refused. Complete metadata/provider-specific and infrastructure-range protection, plus customer network routing, remain separate qualification work.

Only approved account/path pairs receive their registered bearer credential. An agent cannot set Authorization/Host headers, select another tenant's application, request login/debug/setup paths, or use an anonymous alias in this profile. Registered credentials are not put in the public plan or worker configuration, and request Authorization headers are not captured. Arbitrary response bodies and agent-supplied metadata are not a general redaction boundary: echoed, pasted or encoded secrets need additional handling before customer use. The current supported endpoints/data are synthetic, and the recorded exports are checked for actual credential values. A configuration fingerprint binds the credentials to the plan; replacing a credential makes an older run fail closed instead of silently changing its identity.

The trusted broker and its startup configuration remain assumptions. This is not a secret manager or protection against a broker compromise. The VAmPI helper creates synthetic application accounts with one-hour tokens. They are application test credentials reused across permitted runs, not per-run tokens issued by a customer identity provider. Expiry is not automatically refreshed.

Runs remain limited to eight target requests and 90 seconds, with seven normal approved reads. The worker's deliberate out-of-scope attempt is blocked before dispatch and recorded separately. New application runs currently use scripted Mecatl execution. Live-model use is refused until application data-handling and provider disclosure are qualified; the model key is not needed here. A separate optional [reviewed curl adapter](tool-execution.md) can execute these same plans. Arbitrary shell/scanner tools and private connector support for configurable plans remain disabled.

## Review and report contracts

Raw candidates remain unrated. An operator stops the engagement, then submits a candidate ID, its evidence fingerprint, an expected review revision, a decision, purpose, rationale, impact and canonical CVSS 3.1 base vector. The service computes the score and severity using the [FIRST equations](https://www.first.org/cvss/v3.1/specification-document); metrics and business impact are the reviewer's judgments. It does not accept an invented numeric score from the caller.

Reviews are tenant-bound and worker tokens cannot submit them. Conflicting revisions or changed fingerprints are refused. The API preserves revision history in the trusted database. This is not an immutable external audit log or individual human identity system. `operator_review` is an operator attestation; the current tenant credential cannot prove a biological human performed it. Qualification tests explicitly use `automated_lab_test`, including in the finding tags and report. Their example rating is not human signoff or a customer severity assessment.

False-positive dispositions retain the observation and rationale, but create no rated finding and no fabricated zero score. A matching confirm review produces a toolkit-shaped finding and evidence metadata. The exporter keeps finding IDs, artifact paths, timestamps, engagement/target and evidence indexes linked. SHA-256 identifies exact bytes, not independent authenticity against a compromised broker.

`GET /v1/engagements/{id}/export` now returns `tsunami.engagement-export/v2`. Legacy v1 bundles remain supported by the local checker. `POST .../{id}/reviews` records a review and `GET .../{id}/findings.csv` exports reviewed findings only. Fresh materialized exports contain `engagement.json`, `finding-candidates.json`, `findings/finding_*.json`, evidence data/metadata, a CSV and a Markdown draft. JSON preserves original values; CSV neutralizes common spreadsheet formula prefixes, and report evidence uses fences that target content cannot close. These are tested output guards, not certification for every viewer or spreadsheet.

The optional checker reads the friend's actual three schemas locally. Only the finding schema's exact `evidence.json` reference is resolved, from an in-memory local registry; remote retrieval is refused. It checks schema validity, review/rating consistency, evidence membership, hashes, paths, timestamps, index consistency, and materialized reviewed files. Toolkit schemas and utilities are not vendored or executed.

## Checklist references

The local coverage adapter imports versioned checklist IDs/hashes from the inventory and accepts explicit mappings to procedures present in the bundle. It does not execute checklist instructions or change runtime permissions. Unknown IDs, changed source hashes, unrelated procedures and full-coverage claims are refused.

One object-read test is a partial sample of a broad item such as `API1:2023`. It is recorded as sampled when its procedure completes, while fully completed standard items remain zero. Unmapped items remain visible. This is an export-side projection over durable procedure progress, not an implementation of all 26 checklists or a compliance result. Reviewed full test mappings and additional executors remain future work.

## Reproduce on the dedicated lab

After [sandbox setup](sandbox-lab.md), build/deploy the current Tsunami image, then:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
make application-setup CONTEXT=kind-tsunami-lab
make application-verify CONTEXT=kind-tsunami-lab

# Use the actual .../<engagement-id>/export directory printed/collected.
python3 scripts/toolkit.py validate-export \
  --toolkit ../vapt-toolkit \
  --bundle artifacts/application-adapters/JOB/ENGAGEMENT/export/bundle.json \
  --out artifacts/application-adapters/schema-validation.json
```

The setup helper refuses other contexts, checks for active workers, seeds only its disposable lab databases, and mounts the application configuration only in the control plane. Running setup again resets those lab databases and replaces their credentials; do it between assessments. It is a maintenance helper, not an online credential rotation protocol. Model, signing and operator secrets are preserved.

For another owned application, supply a reviewed config through `TSUNAMI_APPLICATION_CONFIG_FILE`; the local-only `application-local --config FILE --cases APPLICATION=EXPECTED` path is also available. The latter uses an in-process worker and does not prove isolation. Only synthetic test data is supported by this release; real customer data, authentication workflows, secret delivery, redaction and retention still need the customer-infrastructure work in [the product roadmap](product-roadmap.md).

The agent never receives setup capabilities. Registration/login/seed operations are trusted setup; its plan allows only the seven GET reads. Kubernetes verification uses actual gVisor workers, while the application lives in a separate namespace with restricted ingress and no egress. Network probes also use an ingress-unrestricted tripwire in that namespace, with trusted controls before/after runner denials. These qualify selected IPv4 paths, not every interface, protocol, sandbox escape or production Stop objective.

See [recorded application results](../examples/application-adapters/README.md). No real-model accuracy, full toolkit coverage, production capacity or customer-readiness claim follows from this campaign.
