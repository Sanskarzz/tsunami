# Owned gVisor/curl Stop campaign

Recorded on 2026-11-05 at 07:21 UTC in `kind-tsunami-lab`. These are four single samples, not latency percentiles or a production guarantee. No external targets or model providers were called.

The worker uses the real Mecatl scripted loop and fixed shell/curl adapter. The owned synthetic API holds the cross-user read open; its own counters record cancellation. Runtime inspection confirmed `runsc` / `io.containerd.runsc.v1` for each worker.

| Case | Stop acknowledgement | Target cancellation observed | CRI exit observed | Result |
|---|---:|---:|---:|---|
| One run | 6.36 ms | 10.52 ms | 278.92 ms | Revoked; no further target requests; cleanup completed. |
| Engagement | 4.07 ms | 6.19 ms | 410.44 ms | Revoked; another child refused; cleanup completed. |
| Controller API access denied | 6.18 ms | 10.48 ms | 435.44 ms | Request cancelled while API observations failed; cleanup pending until recovery. |
| API denial, Stop, then control-plane restart | 4.18 ms | 6.02 ms | 439.09 ms | Revocation survived restart with API still blocked; cleanup completed after recovery. |

Times are measured from the verifier's Stop call using its monotonic clock, including client and polling overhead. Target cancellation was also timestamped independently inside the target. In all four cases, only the first five approved requests reached the target, the held probe was cancelled rather than completed, and counters stayed unchanged through cleanup/restart. At Stop, held-request age was about 0.3 seconds normally and 4.5 seconds under the fault, comfortably before the broker's ten-second request deadline.

The controller's status stayed honest during the API denial: permission revoked, local activity drained, cleanup pending, executor observation unavailable. Host-side CRI inspection independently observed `CONTAINER_EXITED`. Only after API recovery did the controller observe accepted cleanup and absent Job/Pod resources. The restart case restarts after a committed Stop; it does not test a crash during an uncertain acknowledgement.

Builds used:

- Native image `tsunami:stop-v2`: `sha256:4637812e21d0bf31c9c7c76031f00704deab31cbb979fa2555ea1f1d4d26a999`.
- Tools image `tsunami-tools:stop-v2`: `sha256:22c61e4b02030e088a7cf556ccae0465faa0033b79ca4fb0905b1506a9d7e179`.
- Single-node Kubernetes v1.34.0, Cilium 1.20.4, gVisor `release-20261026.0` with systrap, as provisioned by the dedicated sandbox setup.
- Source: the lifecycle follow-up working tree after baseline `985d57a9a9df3eb0876b3a01c2d56e4b81e2b913`; image IDs identify the actual binaries tested. This is not a claim that the baseline commit alone contains these changes.

The private broker leaf certificate had expired on 2026-11-04. The new control plane started with that optional route unavailable while state/revocation readiness and the configured HTTP route remained available. The route map in the records is **startup configuration**, not a live health check of every dependency. A model route being configured did not result in any model calls.

Files:

- [summary.json](summary.json): raw campaign observations, statuses, timestamps, image IDs and limitations; no credentials or target response bodies.
- [native-verification.txt](native-verification.txt): fresh Go race tests, vet, ten Python tests and historical export integrity checks.
- [toolkit-verification.txt](toolkit-verification.txt): validation against the three separately obtained local toolkit schemas.

The diagnostic fault policy was removed; the normal kubeconfig context stayed `kind-kind`. Private certificates and VAmPI credentials were not renewed by this campaign. See [reproduction and trust boundaries](../../docs/stop-lifecycle.md). Broker death, disk loss, arbitrary node partitions, uncooperative target processing, hostile workers, remote executors and Stop under load remain outside this result.
