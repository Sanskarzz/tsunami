# First reviewed web/API command adapter

`curl-http/v1` runs a real shell script and curl inside a gVisor worker. It exercises the same seven-read configured object-access plan as the native worker. It is an original adapter informed by the toolkit's testing workflow, not an installation or execution of its scripts.

The tools image adds `/bin/sh` and curl. Mecatl still exposes only `http_request` and `finish_assessment`; it has no general shell command tool. Configured application runs remain scripted, with synthetic data and no external model calls.

This first adapter proves the command-to-broker boundary; it adds no new vulnerability class or general curl compatibility. The native Go path is simpler for this particular test. External commands add process overhead and a larger image, so later adapters should justify that cost with useful reviewed tool behavior rather than wrapping every Go request in curl.

## Request path

```text
Mecatl http_request(url, account)
  -> fixed reviewed /bin/sh script
  -> curl POST of JSON through a private Unix socket
  -> Go bridge, bound to this run and broker
  -> authenticated broker scope/quota/Stop checks
  -> broker attaches the selected target credential and makes an approved GET
```

Curl never opens the target connection. The local POST submits a read intent; it does not grant upstream POST permission. The broker keeps the approved hostname, port, destination addresses, account/path pairs, TLS verification and redirect policy. Redirects are not followed automatically. Raw HTTP, CONNECT tunnels, caller-supplied methods/headers/bodies, arbitrary URLs and scripts are unsupported.

The script passes JSON on stdin. Target strings never become shell source or curl options. Curl's config file and proxy environment are ignored; it receives a minimal environment, a fixed local URL and a private socket. Target/provider credentials are not supplied in its arguments or environment. Target responses could still echo a credential or contain sensitive data; general redaction is not implemented. The socket is mode 0600 in a mode 0700 directory. The worker checks the script against its embedded bytes; this detects an image mismatch, not a compromised-process security boundary.

A compromised worker can read its own run token and call its bridge or broker directly. That gives it only that run's existing broker permissions. The bridge is an integration point, not a replacement for broker authorization or network enforcement. Installing curl alone, or setting `HTTPS_PROXY`, would not prevent direct sockets.

## Execution and approval

The operator chooses `executor: curl-http/v1` when creating a configured-application engagement. It is included in the approved scope digest and copied into child runs from durable state. A child cannot choose another executor. Unknown profiles, built-in/private procedures and live application inference are refused for this executor.

The startup feature switch is off by default. Enabling it requires a trusted tools image and `gvisor`; the scheduler also checks these before creating a Secret or Job. Native workers and the control plane keep their scratch image. Tool workers retain non-root UID, dropped capabilities, read-only root, no API token and the existing runner egress policy. They add a 16 MiB memory-backed `/tmp` for the socket. Image tags are a lab convenience; qualification records the actual image ID. A production deployment needs immutable image references and a reviewed dependency update process.

Each invocation has a 12-second local timeout and bounded stdout/stderr. The run has at most eight adapter invocations: one deliberate rejected-scope check and seven approved reads. Broker request/action/concurrency limits remain authoritative even if a compromised process bypasses this local counter. There are no automatic retries after uncertain requests. Curl exit output and stderr are not logged as evidence; the broker's target captures remain the source of truth.

Cancellation kills the fixed command's Linux process group, and a disconnected local request cancels its broker call. Durable Stop independently denies subsequent broker requests and cancels active target requests. Ordinary descendants are tested; a malicious program can create another session, so process groups alone do not promise hostile-process cleanup. Pod/cgroup teardown and the network/broker boundary remain necessary. Cancelling an HTTP request cannot undo a target-side effect already performed.

## Reproduce in the owned lab

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
make tools-deploy CONTEXT=kind-tsunami-lab
# Reseeds disposable test databases and replaces their one-hour credentials.
# Run between assessments, not while an agent is active.
make application-setup CONTEXT=kind-tsunami-lab
make tools-network-check CONTEXT=kind-tsunami-lab
make tools-verify CONTEXT=kind-tsunami-lab
```

The dedicated sandbox cluster must already have enforced Cilium policies and gVisor. The tools-deploy helper preserves model/operator/connector configuration and refuses active workers. Building the Debian image installs curl and CA packages; the base digest is pinned, while package versions come from the package repository at build time. Record the built image and curl version rather than claiming reproducible or vulnerability-free dependencies.

## Qualification boundaries

Native tests exercise shell/option injection as JSON data, ignored curl/proxy settings, refused extra/duplicate fields, invocation/output limits, cancellation, ordinary descendant cleanup, broker Stop, upstream TLS rejection and redirect isolation. Existing broker DNS pinning and scope tests still apply; this adapter does not introduce a second resolver for targets.

Kubernetes diagnostics run actual curl in the tools image with direct literal IPs, `--resolve` and `--connect-to`. They check startup, settled and repeat attempts, bracketed by trusted reachable controls. Both sides use gVisor; the harness inspects the actual CRI runtime, not only the RuntimeClass. Credential-free diagnostics deliberately ignore Kubernetes API TLS verification to test connectivity; this flag is absent from the supported adapter and broker. Runtime smoke checks cover no API token, denied raw sockets and denied root writes.

These checks cover selected IPv4/TCP destinations. UDP DNS has its separate Go diagnostic. The later [Stop lifecycle campaign](stop-lifecycle.md) measures active curl/broker/target cancellation under gVisor, including controller API denial and restart after committed revocation. IPv6, metadata, alternate interfaces, CNI failures, cross-node partitions, hostile fork/session behavior and sandbox escapes remain unqualified. GET requests are not inherently harmless. This release uses owned applications and disposable data, not customer data or production systems. There is no production 60-second Stop or capacity guarantee.

The next tool needs its own reviewed intent contract and evidence rules. Browser/authentication flows, arbitrary headers/bodies, scanners, raw network/mobile/VM execution and remote MCP tools remain separate work. Unmodified toolkit scripts usually make direct requests and are still disabled. Toolkit licensing and procedure review are still open.

See [recorded results](../examples/tool-execution/README.md) and [current limitations](current-status.md#unimplemented-or-unqualified).
