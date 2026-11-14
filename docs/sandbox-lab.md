# Testing the execution boundary

The dedicated `tsunami-lab` Kind cluster adds a real gVisor runtime and Cilium networking. It is separate from the original `kind-kind` cluster. General agent-controlled shell execution remains disabled. The fixed curl adapter has its own [reviewed boundary](tool-execution.md); the native checks here cover selected paths, not complete hostile-process containment. [Historical selected-build qualification](../examples/demo-configuration/README.md) records both native and curl checks on the selected demo build.

## Why a separate cluster

The original runner network test failed. Fresh runner pods reached the direct target at startup. After a delay, target and DNS TCP connections were blocked, but the node API and Kubernetes API Service stayed reachable. The API Service points at this single node's control plane.

This is consistent with a pod-policy startup window and a local-node traffic exception in the installed networking implementation. The probes establish the behavior; they do not establish the exact implementation cause of every packet decision. Waiting a few seconds before running an agent would hide the startup gap rather than fix it.

I kept that result and did not replace networking in the existing cluster. The new development cluster uses Cilium with `policyEnforcementMode: always`. This selects default deny even while endpoint identity is being established. Explicit lab policies allow trusted services and the positive-control verifier; the runner policy allows only the broker on TCP 8080. These broad trusted-service rules are for synthetic development, not a customer deployment template.

## Reproduce

Tested on Linux x86_64 with Docker, Kind, kubectl, Helm, Go, Python 3, curl and zstd.

```sh
make sandbox-cluster
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
make deploy CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make isolation-check CONTEXT=kind-tsunami-lab
make verify-k8s CONTEXT=kind-tsunami-lab
make collect-k8s CONTEXT=kind-tsunami-lab
```

The setup pins the Kind node image digest, Cilium 1.20.4, and gVisor `release-20261026.0`. The runtime archive is checked against a pinned SHA-512. Runtime files are installed inside the dedicated Kind node, including the accompanying `gvisor-bin` directory. Nothing is installed into the laptop's runtime directories, and the host Docker daemon is not restarted. A separate ignored kubeconfig keeps the user's normal context unchanged.

The trusted control plane and fixtures use ordinary containers. The deployment renderer puts the chosen worker runtime into the control plane's manifest before creation. Agent Jobs specify `runtimeClassName: gvisor`; this maps to containerd's `runsc` handler using the systrap platform. A configured runtime failure is not retried under runc.

Setup adds another local cluster and uses memory and disk. Re-running it with the pinned runtime already installed leaves that node's containerd running. To remove this disposable lab, use `kind delete cluster --name tsunami-lab --kubeconfig .local/sandbox.kubeconfig`. Removing it also removes its local evidence volume; export reports first.

## What the check does

1. Start a trusted diagnostic Job with `runtimeClassName: gvisor`.
2. Inspect the running pod's actual sandbox through CRI on the dedicated node. Require handler `runsc` and runtime type `io.containerd.runsc.v1`.
3. Check that the pod has no API token, raw socket creation is denied, and root filesystem writes fail. The pod also prints its kernel version as supporting evidence.
4. Run a trusted positive-control probe that can reach all test destinations.
5. Run a runner-labelled probe. At process startup, then after two short delays, try fresh direct connections to the tripwire Service and pod IP, Kubernetes API Service, node API port, and TCP DNS. Also try a UDP DNS lookup of the cluster's own Kubernetes Service. Those paths must time out, while broker HTTP must succeed.
6. Repeat the positive control afterward, so a dead target cannot make a denial look successful.

The diagnostic sends no application test requests directly to the tripwire. TCP handshakes do not increment its assessment counter. DNS queries use an internal cluster name. There are no external assessment targets.

`make network-check CONTEXT=kind-tsunami-lab RUNTIME_CLASS=` repeats network checks with an ordinary container. This helps separate CNI behavior from the runtime boundary. These diagnostic executables are not registered as Mecatl tools.

The scripts save logs and JSON under `artifacts/isolation/<context>/`. A refusal, missing target, unexpected error, or failed positive control is inconclusive and fails qualification. A successful broker request by itself never proves containment.

## What this still does not prove

- No sandbox escape testing or promise that gVisor has no vulnerabilities.
- No IPv6, metadata-route, arbitrary UDP, DNS tunneling, ICMP, alternate interface, or node-failure qualification yet.
- No cross-node or production multi-tenant validation. The lab has one node and one application namespace.
- No measured runtime overhead or production capacity claim; resource requests need profiling before scale estimates.
- No arbitrary shell or external scanner qualification. The fixed curl adapter is assessed separately; later opt-in live-model trials use the same constrained broker tools. The original isolation experiment itself used scripted execution.
- No guarantee that target-side work already accepted before cancellation has stopped.
- No distributed permission lease or independent emergency cutoff yet.

Additional shell/scanner capabilities are deferred by the [current limitations](current-status.md#unimplemented-or-unqualified). Any later selection needs its own reviewed adapter and containment/bypass tests. A proxy environment variable is insufficient. Existing live-model paths use constrained HTTP tools and do not require unrestricted shell access.

## Sources used for setup

- [Cilium on Kind](https://docs.cilium.io/en/stable/installation/kind/) and [policy enforcement modes](https://docs.cilium.io/en/stable/security/policy/intro/).
- [gVisor installation](https://gvisor.dev/docs/user_guide/install/) and [containerd configuration](https://gvisor.dev/docs/user_guide/containerd/configuration/).

The checked-in versions and recorded local results are the reference for this experiment; those upstream pages can change.
