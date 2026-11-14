# Private fields in an allowed response

The new `http-property-access/v1` profile checks a different rule from object ownership: the peer **may read the shared object**, but must not receive its owner-only properties. Blocking the whole object is not evidence that this filtering rule works.

This is a narrow read-only case of [OWASP API3:2023 property-level authorization](https://api-security.owasp.org/editions/2023/en/0xa3-broken-object-property-level-authorization/). The local toolkit's API playbook suggested inspecting response properties rather than trusting what the UI displays. This implementation is original Go code; no toolkit scripts, prompts or schemas are copied or executed. Independent security-engineer review is still pending.

## What is approved

The startup application plan uses the existing two-account, known-object format:

- `version`: `http-property-access/v1`.
- `expected_access`: `filtered`.
- `denial_statuses`: empty. A 403/404 is not a successful filtering test.
- Owner and peer identity assertions and legitimate private-object baselines.
- The owner's baseline `sha256` pointers identify the private fields that must be omitted from the peer's view. Probe assertions/hashes must match that baseline; the agent cannot relax them.
- At least two public assertions on the probe establish the object identity and owner. Protected source fields must be direct members of the root JSON object. Nested source selectors and array-based policies are refused.

The version and all these facts are part of the application/engagement digests. The procedure ID is `{application}/property-access/v1`. Existing `http-object-access/v2` plans keep their old representation, IDs and meaning; historical exports are not rewritten.

Execution still performs seven exact GETs: two identity checks, two legitimate private reads, the peer's shared-object read, the owner's object recheck and the peer's identity recheck. Six successful baseline/recheck observations are required. The allowance stays at eight target requests. No POST/PATCH/DELETE, custom headers, query fuzzing, discovery or mass assignment is added.

## What counts as evidence

| Observed peer response, after verified prerequisites | Result |
|---|---|
| HTTP 200, expected public object, and a string matching a known owner-only value | `confirmed` candidate. The match may be under the selected field or copied to another JSON value, including nested objects/arrays. |
| HTTP 200, expected public object, all selected private fields absent, and no matching known string value elsewhere | `no_issue_observed` for these fields, path and identities. |
| A private field is present with a changed value, `[redacted]`, null or an unexpected type | `inconclusive`, unless another known private value proves a leak. This profile requires omission; it does not silently approve a different masking policy. |
| Denied/generic response, wrong object/owner, duplicate JSON keys, trailing documents, expired identity, missing/changed follow-up baseline or request failure | `inconclusive`. |

JSON parsing is bounded by the existing 64 KiB body and depth limits. Hashes compare exact string values; they are not a general secret scanner. Encoded, partial or otherwise transformed private data can escape these comparisons, particularly under unselected fields. The test does not discover every sensitive field or prove the whole response/application safe. The trusted operator must supply the actual sharing rule and suitable distinct synthetic proof values.

The broker captures the response and makes the decision. A compromised worker can skip/reorder tests or request something forbidden; that cannot manufacture validated evidence, change scope or turn missing prerequisites into a pass. It still uses the same external URL/address/account/path enforcement and network restrictions. This profile adds no direct target-network path.

A repeated permitted probe cannot erase an earlier supported leak just because a later response filters the value. The validator checks captured exposures against their own ordered prerequisites. An earlier exposure without complete ordered identity/object evidence keeps a later filtered outcome inconclusive, with the earlier capture still referenced. Unresolved exposure remains referenced when the later response is already inconclusive, including a generic HTTP 200. The correction extends exposure preservation to the [object-access deny rule](application-adapters.md); property filtering and intentional object sharing retain their distinct meanings. Archived exports are unchanged.

Raw responses remain tenant-authorized evidence, including the synthetic private values used to prove a leak. The verdict helper returns only a classification; it does not replace redaction, retention or customer evidence storage. Those production controls remain unfinished.

## Owned demo

The original Go fixture has shared records with owner-only `private_note` values. It is a controlled demonstration, not a second independently developed application or a real customer assessment. It offers four bounded variants:

1. `property-vulnerable`: the peer receives the private note.
2. `property-filtered`: the peer receives the public record with the private note omitted.
3. `property-ambiguous`: the peer receives a redacted field; the validator keeps this inconclusive under the approved omission rule.
4. `property-stop`: the peer request is held so Stop can be observed at the target.

Setup generates new opaque record IDs, two distinct lab credentials and unique private values, then fills the exact assertions and hashes. This removes hand-written demo bindings; it is **not** general customer discovery/onboarding. Tokens are broker-held and remain valid until this lab setup rotates them; they are not production short-lived run credentials.

On the already provisioned dedicated gVisor/Cilium lab:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make tools-deploy CONTEXT=kind-tsunami-lab IMAGE=tsunami:property-lab TOOL_IMAGE=tsunami-tools:property-lab

# Only if the application Secret mount/observation role is not configured yet:
make stop-lab-setup CONTEXT=kind-tsunami-lab

make property-lab-setup CONTEXT=kind-tsunami-lab
make property-verify CONTEXT=kind-tsunami-lab

# Optional local toolkit-schema checks for the fresh campaign:
python3 scripts/verify-journey.py --context kind-tsunami-lab --profile property --toolkit ../vapt-toolkit
```

Provisioning refuses active/pending workers. It uses a separate restricted namespace for the target, with no service-account token or outbound access and ingress limited to trusted lab control/verifier roles. It preserves other tenant/application configurations; a resource-version check refuses a concurrent registry overwrite. Local generated JSON is written through private temporary files and atomic replacement, and submitted Secret data goes through stdin rather than command arguments. The stored configuration still uses lab Kubernetes Secrets, not an implemented secret-manager integration.

The existing [operator CLI](operator-cli.md) lists these applications. Select an ID, review the plan (including the omitted-field list), approve its digest, start, watch, Stop and export. There is no new natural-language prompt or arbitrary target URL. The real Mecatl loop and fixed `curl-http/v1` adapter execute the plan under gVisor; live-model application execution stays disabled.

The shared qualification driver now accepts `--profile property`. It records actual target request counters, outcomes, rejected unapproved/wrong-digest/stopped launches, protected exports and Stop after a read-only watch timeout. The held worker gets an actual CRI runtime inspection. Fast workers do not acquire a fabricated runtime-exit proof from API object absence.

## Review, compatibility and remaining work

The CLI leaves supported leaks as unreviewed candidates. A reviewer still decides impact/severity through the existing evidence-bound review API. The property finding projection uses the corresponding property-authorization category and CWE-213 instead of the object-ID authorization classification; a native test exercises this with an explicit `automated_lab_test` review. The demo itself does not submit reviews or invent CVSS scores.

Native tests cover vulnerable/filtered/ambiguous results through Mecatl, denied/expired/missing baselines, strict JSON, copied private values, unsupported policies and review classification. The full gate also checks historical artifact integrity and old live-trial hashes. This is not a claim of complete OWASP/API3 or checklist coverage.

See [the recorded campaign](../examples/property-access/README.md) for actual builds, counters and limits. Scope/Stop controls are reused; broader broker failures, partitions, load and hostile-shell containment are not requalified by this field-level demo. Customer identity, private configured applications, general onboarding, useful broader procedures, secret/evidence lifecycle and independent security review still remain.
