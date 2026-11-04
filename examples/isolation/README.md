# Recorded lab boundary checks

Captured on 2026-11-01. These files contain synthetic local-cluster diagnostics, not customer traffic or credentials.

- [Original cluster runner](kind-kind/runc/runner.log): direct target/DNS access at startup and persistent node/API access. The corresponding `network.json` records a failed runner check and successful positive controls.
- [Dedicated cluster, runc runner](kind-tsunami-lab/runc/runner.log): tested paths time out at startup and after settling.
- [Dedicated cluster, gVisor runner](kind-tsunami-lab/gvisor/runner.log): the same paths time out; broker HTTP remains reachable.
- [Runtime inspection](kind-tsunami-lab/runtime.json) and [container checks](kind-tsunami-lab/runtime.log): actual CRI handler/type and raw-socket/filesystem/token smoke checks.

Each network directory includes trusted positive-control logs before and after its runner probe, plus a JSON record with context, destinations, Job names, runtime selection, and image IDs. Every negative network check is compared against live positive controls.

Only the listed IPv4 TCP and UDP DNS paths are covered. A pass is not a sandbox escape-resistance claim or permission to enable arbitrary shell execution. See [the method and limitations](../../docs/sandbox-lab.md).
