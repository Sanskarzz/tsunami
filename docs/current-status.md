# Supported functionality and limitations

Tsunami is an owned-lab prototype for bounded HTTP authorization assessments. The default uses scripted Mecatl and makes no external model calls. The setup advisor uses checked rules. [Verification](verification.md) distinguishes source checks, historical qualification and current readiness.

## Implemented journey

| Capability | Current behavior |
|---|---|
| Onboarding | Typed facts/questions, protected origin-bound bearer input, separately approved four-read baselines, atomic registration and unapproved draft. Unsupported authentication stays unavailable. |
| Scope and approval | Immutable origin/address/account/GET/path/rule/budget/expiry/executor bindings. Approval compares the exact reviewed scope digest in the durable transaction. |
| Start | Client action key and atomic tenant/engagement/child/credential reservation. Explicit recovery identifies the same run after Stop/restart/expiry without renewed permission or replay. |
| Execution | Kubernetes Jobs and real Mecatl. Native Go or fixed curl-to-broker adapter; no general shell or unrestricted target-fetch tool. |
| Assessment | Seven approved reads: identities, legitimate objects, peer probe and rechecks. A separate fixed escape intent exercises rejection. |
| Activity | Bounded persisted broker/controller/validator events; metadata polling with stale/error/uncertainty states. No fictional progress or raw-body stream. |
| Findings/reporting | Deterministic verdicts, unreviewed candidates, explicit artifact byte/provenance checks, draft exports and remediation. Separate evidence-bound review API. |
| Stop | Durable run/engagement/tenant/global revocation, cancellation and separate drain/execution/cleanup observations. Restart never resumes stopped work. |
| Retest | Explicit destination, linked new draft and fresh approval/Start. Original evidence preserved; incompatible checks labeled. |

The loopback UI is an operator interface, not a hosted customer portal. The CLI uses the same APIs. [Architecture](design.md) maps actions to stored state.

## Result meanings

- `confirmed`: captures support the configured forbidden-access observation; human impact/severity review remains separate.
- `no_issue_observed`: the completed configured check behaved as expected. It is not application-wide security or independent patch verification.
- `inconclusive`: evidence/prerequisites are incomplete, invalid or ambiguous.
- `stopped`: revoked permission; a run state, not a clean result.

Vulnerable/fixed VAmPI tests object access. Separate fixtures test property leak/filter/ambiguity and Stop during a held request. These are supplied procedures and synthetic contracts, not blind discovery. Target accounts are distinct from platform tenants.

## Optional but maintained

The catalog, model gateway/Responses fixture, historical opt-in free-model evaluations, signed private connector and connector mTLS paths remain reachable and tested. They have [separate prerequisites/evidence](README.md#optional-experiments). Configured applications reject live-model execution. No provider key is required for source checks or the default demo.

## Unimplemented or unqualified

- Production customer identity/roles, billing and independently qualified tenant isolation.
- Generic discovery, arbitrary application contracts, cookie/password/SSO/MFA login, automatic refresh and live-model application planning.
- General redaction, retention/deletion, external secret management, complete evidence encryption, immutable external audit and operational backup/restore qualification.
- Multi-replica storage, distributed customer-wide budgets, production capacity/load and broad crash/partition guarantees.
- Hostile-worker/connector containment, sandbox escape resistance, IPv6/metadata/all-protocol coverage and cross-node isolation.
- Independent external emergency network cutoff. Normal Stop needs the trusted process/store and cannot undo remote effects.
- Network/mobile/cloud/VM/MCP testing, general scanners/shell commands and specialist-agent coordination.
- Independent procedure review and a less guided assessment-quality benchmark.

These are boundaries, not an automatic feature queue. [Security](security.md) explains enforced controls and assumptions. Production customer data is unsupported.

## Licensing and attribution

Apache-2.0 is under consideration; no root license is installed pending final confirmation. Git records one author, but this alone does not establish rights under assignment, employment or contribution terms. The owner must confirm rights to publish original code, documentation and recorded assets.

Inspected direct Go dependencies use Apache-2.0 (Mecatl engine/provider, OpenAI Go SDK) or MIT (bbolt). Inspected transitive module licenses include MIT/BSD-style and dual MIT/Apache terms; original licenses remain applicable. A Tsunami license would not relicense dependencies. Preserve applicable copyright/license/NOTICE obligations for distributed binaries and containers.

VAmPI is fetched at `f16052dce83f05847133ec98f01c5193a41de7d8` into ignored `.local/external/`, under upstream MIT terms. Its source/license enter its lab image, not this Git tree. Debian/curl/Python images have separately licensed components. Complete binary/container attribution packaging remains necessary if those artifacts are distributed.

The external methodology toolkit has no detected license in its separate checkout. Tsunami does not vendor or execute it; local inventory/schema checks grant no redistribution rights. Do not copy its scripts, prompts, schemas, assignment text or other assets without resolving their rights. The embedded frontend has no external UI asset package; retained screenshots are synthetic qualification records.
