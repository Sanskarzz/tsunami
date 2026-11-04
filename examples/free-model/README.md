# Actual free-model trial

These records came from the dedicated `kind-tsunami-lab` cluster on 2026-11-01. Unlike the older model-path samples, the upstream service was a real model: `nvidia/nemotron-3.5-lightning:free` through the pinned OpenRouter free-only profile. The target applications and all account data were synthetic.

- [Vulnerable report](vulnerable.md) and [JSON evidence](vulnerable.json): confirmed cross-user document read.
- [Fixed report](fixed.md) and [JSON evidence](fixed.json): no issue observed in the tested reads.
- [Stopped run](stop.md) and [JSON state](stop.json): stopped during the first inference, before any target request. Missing usage remains uncertain.
- [Pair log](verification.log), [stop log](stop-verification.log), and [recorded counters](record.json): exact run IDs, evaluation image, and the 29 ms broker-drain observation.
- [Account before](account-before.json), [after the connection check](account-after-smoke.json), and [after the complete trial](account-after.json): only pricing and numeric usage/allowance fields; final usage and daily usage were $0, free requests used rose from 0 to 12.
- [Connection check](connection-check.json): one real adapter request completed before the Kubernetes trial.
- [Network record](network/network.json), [runner log](network/runner.log), [before](network/positive-before.log) and [after](network/positive-after.log) controls: the worker could reach the broker, but direct target/API/DNS/model paths timed out. The public-model probe used one resolved IPv4 address on TCP/443 without credentials or HTTP requests.

Both assessments reserved 5 calls and 5 upstream attempts, and captured 4 target responses. They tested legitimate reads and both cross-user directions. The validator uses fixture ownership; the model cannot declare the result confirmed.

The stop harness waited for HTTP 200 provider stream headers during a currently active inference, submitted durable Stop, waited for all broker operations to drain, and checked for no counter growth for one second. Vendor compute/billing cancellation is unverified. The live run used 1 attempt and recorded no completed turn.

The assessment/stop image is recorded in `record.json`. A later image added only the opt-in public-network diagnostic; its ID is in `network/network.json`. Reports retain original run IDs and timestamps. They contain synthetic evidence and aliases, with no provider, operator, or run credentials.

This is one guided integration trial, not a blind discovery benchmark, broad containment qualification, or a production stop guarantee. See [limits and reproduction](../../docs/openrouter-free.md).
