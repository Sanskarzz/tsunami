# Getting Started

Choose **Path A** to inspect and test the implementation without Kubernetes. Choose **Path B** only to provision an owned synthetic lab in an isolated environment. Commands run from the repository root, in Bash. No default path requires a provider API key or real customer target.

## Prerequisites

| Requirement | Source verification | Kubernetes demo |
|---|---|---|
| Platform | Tested Ubuntu 25.10, Linux amd64; other OS/architectures unqualified | Linux x86_64; the helper accepts only x86_64 and assumes Linux Docker/Kind behavior; complete bootstrap remains unqualified |
| Go | `go.mod` requires Go 1.27.4; older patch versions are refused | Same |
| Build tools | Git, make, C compiler/CGO for race tests, `/bin/sh`, `/usr/bin/curl` | Same, plus tar, zstd, curl and SHA-512 utilities |
| Python | Tested 3.13.7; venv/pip; install `scripts/requirements-test.txt` | Same; target image separately uses Python 3.11 |
| Node | Tested 22.23.3; no npm installation needed | Node 22 for automated browser qualification |
| Docker/Kind | Not needed | Observed Docker 29.1.3 / Kind 0.30.0; functioning Docker daemon and permission to manage it |
| kubectl/Helm | Not needed | Observed kubectl 1.35.3 / Helm 4.0.4; these are observed versions, not a compatibility matrix |
| Runtime/network | Not needed; local tests do not prove process isolation | Script pins Kubernetes 1.34.0 node digest, Cilium 1.20.4 and gVisor release-20261026.0 with an archive checksum |
| Browser | Chrome only for optional browser tests | Any suitable local UI browser; automation specifically calls `google-chrome` |
| Network | Initial Go/Python dependency downloads | Also image registries, Helm chart, gVisor archive and pinned VAmPI source |

Install tools using their upstream instructions: [Go](https://go.dev/doc/install), [Kind](https://kind.sigs.k8s.io/docs/user/quick-start/), [Docker](https://docs.docker.com/engine/install/), [kubectl](https://kubernetes.io/docs/tasks/tools/), [Helm](https://helm.sh/docs/intro/install/). The checked-in [setup script](../scripts/setup-sandbox.sh) supplies the lab Cilium/gVisor configuration; do not install or replace the host's runtime/CNI to follow this guide.

Minimum RAM/CPU/disk has not been measured. The inspected host had about 15 GiB RAM; its idle lab node used about 1.5 GiB at one observation. This is not a minimum or peak-load estimate. Builds, race tests, browser tests and image downloads need additional resources.

## Path A — verify the source locally

### Clone and install test dependencies

```sh
git clone https://github.com/Sanskarzz/tsunami.git
cd tsunami
python3 -m venv .local/test-venv
. .local/test-venv/bin/activate
python3 -m pip install -r scripts/requirements-test.txt
```

The two pinned Python validator requirements are needed by the test suite. The optional external toolkit is not needed. `.local/` is ignored; never copy another workstation's credentials, databases or kubeconfig into a clone to make source checks pass.

### Test and build

```sh
make verify
make verify-ui-retest
make build
```

Expected success:

- `make verify`: fresh Go race tests, vet, 34 Python tests and evidence checks for 38 materialized bundles, 286 artifacts and 20 historical trial hashes.
- `make verify-ui-retest`: 38 frontend tests pass. These isolated test fixtures never appear in the product UI.
- `make build`: `bin/tsunami` and its host CA bundle are created. The release record identifies the tested build; counts may change in later versions.

Tests use temporary local state and owned loopback targets. They neither provision Kubernetes nor call an external model. Go packages without their own test files are exercised by integration tests; the default Go gate skips opt-in Chrome campaigns.

### Optional real-browser checks

With `node` and `google-chrome` on PATH, run sequentially:

```sh
make verify-ui-activity-browser
make verify-ui-onboarding-browser
make verify-ui-findings-browser
make verify-ui-retest-browser
```

These run the real local control plane, broker, validator and Mecatl against disposable loopback targets. They save private synthetic records under ignored `artifacts/`. They do not qualify Kubernetes/gVisor/Cilium or use persistent lab engagement slots. See [verification](verification.md).

## Path B — run the controlled Kubernetes demo

**Fresh bootstrap qualification: NOT RUN.** The sequence below is derived from maintained scripts and targets. Individual components have historical lab evidence; the complete clean-machine sequence has not been executed in this release preparation. Do not interpret it as a tested one-command installation.

### Prepare a genuinely separate environment

Use a fresh Linux machine/VM or explicitly isolated Docker environment. A new Git checkout on a machine with an existing `tsunami-lab` Docker node is not a fresh lab. The scripts use fixed cluster/node names; do not rename them piecemeal, import someone else's kubeconfig or delete an existing cluster to satisfy this guide.

Complete Path A first. Check tools and refuse an existing lab:

```sh
for tool in docker kind kubectl helm curl sha512sum zstd tar; do
  command -v "$tool" >/dev/null || { echo "Missing prerequisite: $tool" >&2; exit 1; }
done
docker info >/dev/null
if docker inspect tsunami-lab-control-plane >/dev/null 2>&1; then
  echo 'Existing tsunami-lab node found. Stop: use a separate environment.' >&2
  exit 1
fi
if [ -e .local/sandbox.kubeconfig ]; then
  echo 'Existing lab configuration found. Stop and inspect its ownership.' >&2
  exit 1
fi
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make sandbox-cluster
```

The helper creates a dedicated single-node cluster with default CNI disabled, installs pinned Cilium and runsc inside that node, and writes a private kubeconfig. It does not change the normal kubeconfig context or restart the host Docker daemon. Bootstrap now refuses an existing dedicated kubeconfig or Kind node before downloads or writes. Use the explicit maintenance helpers for an existing owned lab; do not bypass bootstrap refusal.

Inspect prerequisites before deployment:

```sh
kubectl --context kind-tsunami-lab get nodes
kubectl --context kind-tsunami-lab get runtimeclass gvisor
kubectl --context kind-tsunami-lab -n kube-system get configmap cilium-config \
  -o jsonpath='{.data.enable-policy}'
```

Require a Ready `tsunami-lab-control-plane`, RuntimeClass handler `runsc` and policy value `always`. These configuration checks alone do not prove runtime/network containment. The shared [deployment guard](../scripts/deployment_guard.py) makes read-only prerequisite checks; it is an operator mistake guard, not authorization, an ongoing maintenance lock or fresh runtime qualification. Source `make build`, `make image` and `make tools-image` remain standalone local build commands.

### Build, deploy and seed synthetic targets

The following commands mutate only the dedicated owned lab. Do not run them against an existing lab as an implicit refresh. Bare `make deploy` is refused before image builds. Deployment targets and direct setup helpers require the absolute private repository kubeconfig, context `kind-tsunami-lab` and runtime `gvisor`. The guard checks the local TLS configuration, Docker Kind ownership/published API port, Ready node, RuntimeClass `runsc`, and ready Cilium in `always` mode before admitting builds or mutations. It refuses merged/ambient configs, TLS bypass/proxy/auth plugins and identity mismatches. Specify the selection as shown.

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make deploy CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make tools-deploy CONTEXT=kind-tsunami-lab
make application-setup CONTEXT=kind-tsunami-lab
make stop-lab-setup CONTEXT=kind-tsunami-lab
make property-lab-setup CONTEXT=kind-tsunami-lab
make ui-deploy
make showcase-check
```

Base deployment creates the control plane, operator Secret, state PVC and legacy fixtures. Tools deployment enables the reviewed curl route. Application setup fetches pinned VAmPI into `.local/external/` and seeds only disposable vulnerable/fixed databases. Stop/property setup creates separate owned fixtures. UI deployment selects matching native/tools images and creates the onboarding encryption key once; it preserves an existing key. No external model is configured by this fresh path.

Setup generates application credentials/proofs and exact Service-address bindings. Do not commit them. Re-running setup rotates fixture access/data and invalidates old bindings. Do it only as explicit idle-lab maintenance. Preserve database and matching onboarding key together; key replacement is not an acceptable recovery shortcut.

`showcase-check` should report PASS prerequisites and no target requests. It inspects configuration, capacity, idle permissions and unsigned JWT expiry hints. It does not authenticate accounts or establish network isolation. Require the onboarding route as well: after connecting, route status should show it ready; UI deployment must have succeeded.

### Qualify the runtime and selected network paths

These create bounded diagnostic workloads but no engagements. Run only in your explicitly owned lab and sequentially:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make isolation-check CONTEXT=kind-tsunami-lab
make tools-network-check CONTEXT=kind-tsunami-lab
make showcase-check
```

Require actual CRI runsc observations and successful positive-before/runner-denial/positive-after checks. A missing destination or failed positive control fails qualification. Do not proceed to assessments to obtain a nicer result. Read [sandbox limits](sandbox-lab.md) and [curl qualification](tool-execution.md).

### Access the UI privately

In one terminal:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
kubectl --context kind-tsunami-lab -n tsunami port-forward \
  --address=127.0.0.1 service/tsunami-control 8080:8080
```

In another terminal at the same repository root, save the tenant operator credential without printing it:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
mkdir -p .local
chmod 700 .local
umask 077
kubectl --context kind-tsunami-lab -n tsunami get secret tsunami-operator \
  -o jsonpath='{.data.admin-a}' | base64 --decode > .local/operator-token
chmod 600 .local/operator-token
test -s .local/operator-token
```

Open `http://127.0.0.1:8080/ui/` and privately paste the token into the masked Connect field using a trusted local editor/clipboard. The token is not a target account credential. Do not put it in frontend configuration, URLs, recordings, command arguments or Git. The UI has no public ingress/customer sign-in; its loopback Host check does not replace API authentication.

### Run the example assessment

1. Add application; select the existing reviewed `vampi-vulnerable` integration and confirm disposable synthetic data. Supply a name/environment and authorization reference. The shown registered API target stays authoritative; a frontend URL does not broaden it.
2. Review and explicitly approve the four legitimate prerequisite reads. Validate them; failed identities or unsupported contracts stay unavailable.
3. Prepare the assessment draft. Review the exact target, addresses, account/GET/path matrix, intended deny rule, proof fields, budget and expiry. Check the review box and approve. Approval starts no worker.
4. Explicitly Start. The keyed action reserves one run; reconciliation dispatches its Job. Observe actual broker events and separate worker observations.
5. Open Evidence. A vulnerable control should produce an unreviewed `confirmed` candidate. Inspect owner/probe captures explicitly; their bytes/provenance must match. Read remediation without treating it as a generated/deployed patch.
6. Create a linked retest, explicitly selecting `vampi-fixed` with a fresh authorization reference. Review changed scope, approve and Start separately. A completed `no_issue_observed` covers only this configured rule; original finding/evidence remain intact.

The paired workflow consumes two engagement slots. A Stop demonstration needs a third. Property controls use separate applications/approvals and are not part of a fictional combined target. See [the exact lab contract](demo-configuration.md).

### Inspect rejection and Stop

**Show scope rejection** locates a real persisted rejected intent. The broker rejected it before contacting the unapproved destination. It is a fixed safety test, not model discovery.

For active-request Stop, create/approve a separate `stop-lab-run` assessment. Explain Stop before Start; its fifth approved request is held. Click Stop while the request is active and before the broker timeout. Read acknowledgement, durable revocation, local drain, worker observations and cleanup separately. Ordinary UI status is not the fixture's independent cancellation counter or CRI proof; qualification tooling collects those observations separately.

Closing the page, pausing observation, watch timeout or Ctrl+C on port-forward does not revoke permission. Use explicit assessment Stop. Stop cannot undo a remotely accepted action or prove process exit from missing API objects alone.

## Recovery and safe cleanup

| Situation | Action |
|---|---|
| Python validator import failure | Activate the venv and install `scripts/requirements-test.txt`; do not skip validators to obtain PASS. |
| Missing C compiler/curl | Install source prerequisites; race/adapter tests need them. |
| Existing cluster/node/config found during fresh setup | Stop; inspect ownership or use an isolated environment. Do not adopt or delete it automatically. |
| Failed runtime/network control | Keep the failed record. Do not fall back to runc or broaden policy. |
| Expired VAmPI credentials | They last one hour. Between assessments, explicitly authorized core maintenance can use `python3 -B scripts/showcase.py prepare`, then `make showcase-check`. This rebuilds/deploys and refreshes owned VAmPI/Stop data, so qualification must match the replacement build. |
| Full property refresh | `make showcase-prepare` is explicit broader maintenance and requires seven free slots; it also resets property fixture data. |
| Start response lost | Read saved child IDs or explicitly Recover Start with the original key/request. Never generate a replacement key automatically. |
| Stop acknowledgement lost | Show uncertainty; inspect saved permission separately. A connection error is not successful revocation/cleanup. |
| Missing onboarding key | Restore the matching key; never regenerate it to decrypt existing state. |
| Insufficient history capacity | Preserve/export history and plan separate maintenance. Do not wipe state, raise limits, clear gates or rotate tenants to make tests pass. |
| API unavailable | Normal Stop may be unavailable. Inspect trusted platform controls; no independent external egress cutoff is supplied. |

Readiness requires idle work and no pending diagnostics. Core checks/preparation require three free slots; full selection requires seven. Guards do not reserve capacity against another operator. Run this as a single-operator lab. `make demo-rehearse-core` creates three assessments; `make demo-rehearse` creates six and requires seven free slots. Both need explicit operator authorization and prepared matching builds. They are not part of Path A.

For routine shutdown, explicitly Stop your active assessments, inspect permission/drain/cleanup, preserve needed exports, then end the port-forward. Disconnecting only ends observation. Keep state, keys, approvals and Stop gates. Do not delete the PVC, namespaces or cluster as routine cleanup: cluster teardown destroys the local evidence volume. Any destructive teardown needs a separate backup/retention decision.

Raw artifacts can contain sensitive synthetic bodies. Review selected exports before sharing; never publish whole `.local/` or `artifacts/` trees. See [verification](verification.md), [security](security.md), [CLI reference](operator-cli.md) and the [prepared presentation script](demo-walkthrough.md).
