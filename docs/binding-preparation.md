# Prepare a plan from supplied facts

`tsunami prepare-plan` replaces hand-written assertion maps, literal object paths and private-value hashes with a small binding description and four inline baseline documents. It supports the existing object-access and property-access profiles. It makes no HTTP requests and does not register, approve or execute anything.

This is still synthetic-lab tooling. An operator chooses the actual application, exact destination addresses, intended sharing rule, endpoints and JSON selectors. The tool derives mechanical bindings; it does not discover business rules, prove legal ownership or authenticate the supplied documents.

## Inputs and output

The input contains:

- Application ID/title, HTTP origin and explicit approved IP addresses.
- `http-object-access/v2` with `allow`/`deny`, or `http-property-access/v1` with `filtered`.
- The shared identity endpoint/pointer, object path template with one `{id}`, object ID/owner pointers and one to six protected pointers.
- Each account's supplied identity response and legitimate own-object response, including their declared status codes. All four must be HTTP 200.
- Explicit `synthetic_data_only: true`. Customer data is not supported by this preparation workflow.

The compiler checks distinct identities and records, verifies that each supplied object owner matches its identity, and extracts the object IDs. IDs must be safe literal string path segments; numeric IDs, escaped paths and arbitrary route discovery are unsupported. Protected values must be distinct synthetic strings of 24–256 bytes, rather than common/empty proof values. Property-access retains its top-level protected-field restriction.

The whole input is capped at 64 KiB and JSON depth eight, including its envelope. Duplicate keys, trailing documents, unknown envelope fields, missing/wrong-shaped values and incompatible profile rules are refused. Response documents are inline, so there are no file references, executable expressions, OpenAPI imports or remote `$ref` fetches.

Run the included fictional template without contacting its reserved example address:

```sh
make build
mkdir -p .local
./bin/tsunami prepare-plan \
  --input examples/binding-preparation/input.template.json \
  --out .local/my-prepared-plan
```

The output is a new private directory containing:

| File | Meaning |
|---|---|
| `plan.json` | Registration-plan shape with extracted public identity/ownership facts and hashed protected values. The credential fingerprint is deliberately empty. |
| `manifest.json` | Hashes of the input and exact plan bytes, required credential aliases and `pending_operator_registration_and_scope_approval`. Target requests remain zero. |

Both files use mode 0600 in a mode-0700 directory. Existing output directories are refused. The Linux command accepts a bounded regular input file, refuses a final symlink/FIFO/device, and never interprets a path from document content. Errors do not echo baseline values. A write failure can leave an incomplete new directory; it must not be installed as a completed package.

Only the explicitly selected public fields are copied into the plan. Protected proof values are hashed, but this is not a general redactor: operators must choose nonsensitive public identifiers and keep the original baseline input private. Other fields in the supplied bodies are not copied into the prepared plan.

## Preparation is not permission

A trusted operator must register the prepared plan through the existing startup configuration, with separately supplied credentials for aliases `owner` and `peer`. The server computes the actual credential-set fingerprint; the package's local hashes are not an approval digest or proof of source authenticity.

After registration, use the normal [operator flow](operator-cli.md): select the application, create a draft, review the server's actual scope, approve that digest, then start a bounded attempt. Native tests verify that preparation cannot create an unapproved child run. No new application-registration API or customer sign-in is introduced.

The live assessment still performs its identity/object checks and owner/session rechecks. Supplied documents may be stale, fabricated or bound to the wrong credentials; compiling them does not remove these runtime checks or turn them into successful evidence. Configuration changes require the normal new-plan/approval path. No agent is given the authority to expand scope from response content.

## Current owned-lab integration

Both setup helpers now use this compiler rather than constructing assertion/hash maps themselves:

- VAmPI setup creates disposable accounts/objects as before, then captures four legitimate identity/own-object GETs per variant. It derives the bindings from those actual supplied responses. These setup reads are separate from the seven reads in an approved assessment and do not probe cross-user access.
- Property-fixture setup supplies documents from its own generated synthetic seed data. That is not a live observation or an independent target measurement. The later broker-mediated run verifies the actual fixture behavior.

Raw inputs and prepared packages stay under ignored `.local/application-bindings/` directories with private permissions. The helper sends filenames, not response bodies or credentials, in subprocess arguments. Credentials remain separate in the existing lab configuration; this does not implement Vault/ESO or production secret handling.

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
# Existing dedicated gVisor/Cilium cluster and configured application mounts:
make tools-deploy CONTEXT=kind-tsunami-lab IMAGE=tsunami:binding-lab TOOL_IMAGE=tsunami-tools:binding-lab
make application-setup CONTEXT=kind-tsunami-lab
make property-lab-setup CONTEXT=kind-tsunami-lab

make journey-verify CONTEXT=kind-tsunami-lab
make property-verify CONTEXT=kind-tsunami-lab
```

The setup targets now build the local compiler first. Direct helper invocation also checks for a compatible compiler before provisioning. Setup still resets only the owned disposable targets and preserves unrelated tenant/application plans. Preparation itself has no network client; actual target access belongs to the trusted setup driver or the separately approved broker path.

See [the retained preparation/runtime records](../examples/binding-preparation/README.md). This reduces manual binding work; it does not turn a URL into a complete assessment, remove the need for application facts, supply general customer onboarding or establish customer readiness.
