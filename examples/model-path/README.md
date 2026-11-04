# Model-path fixture results

These came from real Mecatl workers using gVisor Jobs in `kind-tsunami-lab` on 2026-11-01. The Responses adapter and broker ran normally, but the upstream server was a deterministic API fixture, **not a live LLM**.

- [Vulnerable report](vulnerable.md): both ownership baselines succeeded, and Bob received Alice's known private document.
- [Fixed report](fixed.md): the cross-user read was forbidden.
- [Evaluation Job log](verification.log): expected/actual results and exact run IDs.
- [Recorded counters](record.json): 5 model-gateway calls and 5 upstream attempts per run. The usage values are synthetic fixture data.
- [Runner network log](network/runner.log): direct model-service access times out while broker access succeeds. Matching before/after positive controls and a JSON record are in the same directory.

Each assessment also attempted an unapproved origin. The broker denied it. The model did not decide the finding verdict; the trusted validator used the captured target responses.

Reports were exported by their exact run IDs and given readable scenario filenames here. They contain only synthetic documents and account aliases. No provider, operator, or run credential is exported.

These samples verify integration and enforcement. They do not demonstrate live-model quality, general vulnerability discovery, complete container isolation, or a production stop guarantee. See [how to reproduce and what remains](../../docs/model-path.md).
