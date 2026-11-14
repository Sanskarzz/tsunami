# Free-model trial

The model path now works with a real provider in the dedicated lab. It still tests only our synthetic document API. Scripted execution remains the default; configuring a key does not start inference.

## What ran

On 2026-11-01, `nvidia/nemotron-3.5-lightning:free` passed the vulnerable/fixed pair through real Mecatl workers in gVisor Jobs. Both runs used 5 model calls and 4 approved target requests. The model tested both directions of cross-user access. The trusted validator used the known-owner baselines and captured responses to decide the result.

| Check | Observation |
|---|---|
| Vulnerable API | `confirmed`; Bob received Alice's private fixture document |
| Fixed API | `no_issue_observed`; the tested cross-user reads returned 403 |
| Stop during inference | Provider HTTP 200 stream headers observed; broker drained in 29 ms; no counter growth over the next second |
| OpenRouter account metadata | Usage and daily usage reported $0 after 12 free requests |
| Direct model access from worker | One resolved public IPv4 TCP/443 address timed out in all three probe phases; trusted before/after probes connected |

The 12 requests cover one connection check, 10 paired-assessment calls, and one stopped inference. Completed assessment usage was 5,679 input / 228 output tokens for vulnerable and 5,537 / 216 for fixed. These are provider-reported values. The stopped call returned no usable usage and stays marked uncertain.

The stop sample measures our operator call through broker drain, including polling. It does not prove when the vendor stopped computing. Already accepted application work cannot be undone. One paired trial is not a quality benchmark; the prompt supplies accounts, paths, and ownership rules. See [exact reports, logs, and sanitized account snapshots](../examples/free-model/README.md).

## Free-only controls

I'd rather fail the run than silently try a paid route. The `openrouter-free` profile applies these checks outside the worker:

- Only the official HTTPS API base and an explicit `vendor/model:free` ID are accepted. Generic Responses mode rejects OpenRouter routing and OpenRouter credentials.
- Immediately before an upstream attempt, the transport verifies the model and exactly our two function tools. It rejects fallback model lists, plugins, routing overrides, service tiers, and hosted tools.
- It sends `allow_fallbacks: false`, `require_parameters: true`, and zero prompt/completion/request price ceilings. Reasoning is disabled for this small trial; output is capped at 1,024 tokens.
- SDK retries are disabled. Each call and actual wire attempt reserves one of the run's 8 slots before dispatch. Failed attempts keep their reservation. There is no automatic switch to another model.
- `free-eval` and `free-stop-check` require the `openrouter-free/v1` policy when creating a run, as well as checking operator configuration. This prevents a configuration race from silently creating a paid run.

These controls trust OpenRouter's routing and pricing enforcement. The sanitized usage snapshot is a point-in-time vendor report, not an independent billing audit. The preflight catalogue check is also a snapshot; the last-hop zero ceilings still apply if pricing changes afterward. A free route can be busy or unavailable; the assessment may fail or be inconclusive. The observed account allowance was 50 free requests/day, with 38 remaining after this trial. That is not a hardcoded platform limit.

OpenRouter documents [provider price ceilings and fallback controls](https://openrouter.ai/docs/guides/routing/provider-selection), its [Responses interface](https://openrouter.ai/docs/api_reference/responses/overview), and [account usage/limits](https://openrouter.ai/docs/api_reference/limits). Check them again before changing provider behavior.

## Reproduce

Use the dedicated local cluster. Put a valid key in the ignored `.local/openrouter-api-key` file with mode `0600`; do not put it in a command argument, report, image, or Git. Rotate any key previously shared in chat.

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor

# Metadata GETs only; verifies tool support and zero listed token/request prices.
python3 scripts/openrouter-preflight.py \
  --model nvidia/nemotron-3.5-lightning:free \
  --out artifacts/openrouter/before.json

python3 scripts/configure-model.py \
  --context kind-tsunami-lab --provider openrouter-free \
  --model nvidia/nemotron-3.5-lightning:free \
  --key-file .local/openrouter-api-key

# Explicitly requests real free inference; neither target allows a paid fallback.
make free-eval CONTEXT=kind-tsunami-lab
make free-stop-check CONTEXT=kind-tsunami-lab

python3 scripts/openrouter-preflight.py \
  --model nvidia/nemotron-3.5-lightning:free \
  --out artifacts/openrouter/after.json

# TCP reachability only to a resolved address of this fixed model dependency;
# no API credential or assessment request is sent by the diagnostic probe.
PROBE_OPENROUTER=true make isolation-check CONTEXT=kind-tsunami-lab
```

Model configuration creates a Secret mounted only in the control-plane pod and restarts that Deployment. Rotate/reconfigure between assessments. Workers still receive only a run credential and cannot directly contact the provider on the tested network path. Metadata checks do not start inference. Free evaluations consume the vendor's free request allowance.

Use the Stop API for active assessments. Afterward, `make disable-model CONTEXT=kind-tsunami-lab` disables new live runs and removes the provider Secret. A provider restart or configuration change is not a substitute for durable Stop.

## Still missing

Repeated and less guided evaluations, private connectivity, customer evidence review, general monetary budgets, distributed concurrency, partitions, and an independent emergency cutoff. The worker has no shell/scanner tools. The public probe covers one IPv4 address, not every provider address, IPv6, metadata, alternate interfaces, or sandbox escape. Existing offline tests cover scope denials and cancellation races; this live model did not attempt an out-of-scope request.
