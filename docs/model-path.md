# Optional model path

The default workflow still uses Mecatl's scripted adapter. This milestone adds a separate, opt-in model path through the trusted broker. The first supported wire protocol is the Responses API, using Mecatl's published `provider/openai` v0.3.0. This does not imply support for every service described as OpenAI-compatible; a Chat Completions-only endpoint will not work.

Offline checks use a local Responses fixture with deterministic tool calls and synthetic token counts. A separate opt-in trial now uses a real OpenRouter free model; the recorded account usage remained $0. See [the free-only profile and trial](openrouter-free.md). Fixture and real-model records stay separate.

## How it works

1. An authenticated operator creates a run with `execution: live`. Omitting that field selects scripted execution, even when a model is configured.
2. The gVisor worker uses Mecatl's loop with the same two tools: `http_request` and `finish_assessment`. There is no shell, browser, external MCP server, or scanner.
3. The worker sends conversation data to `/v1/runs/<id>/model` with its run credential. It cannot choose the provider, model, API endpoint, system prompt, or tool definitions.
4. The trusted server builds those settings from operator configuration and the durable run policy. Only control-plane mode loads the API key and constructs the upstream provider client.
5. Each model call and each upstream HTTP attempt reserve quota before dispatch. Stop revokes permission and cancels the tracked model context as well as target HTTP contexts.
6. The broker releases a completed model turn to the worker. An interrupted or malformed stream releases no partial tool calls.
7. `finish_assessment` asks the existing known-owner validator to evaluate broker evidence. Model text cannot set a verdict. After validation, the worker emits a fixed completion message locally rather than paying for another model call.

Worker networking is unchanged: only the broker is reachable. The model API key is mounted only in the trusted control-plane pod. Workers receive their own run credential, not a provider credential. A compromised worker can spend its run's remaining allowance, but cannot use that allowance for a different run or change the configured model route.

The request format carries text conversation and opaque reasoning replay data. Remote image/file/resource references are rejected. Only the two function tools are advertised upstream; built-in browsing, hosted MCP, and other remote execution features are not enabled. Target content remains untrusted regardless of what the prompt says.

## Run without a model key

Locally:

```sh
make model-fixture-demo
```

This runs the actual Responses adapter, broker, validator, and Mecatl loop against a local API fixture. It writes a vulnerable/fixed pair to `artifacts/model-fixture/`. It does not claim process isolation.

In the dedicated gVisor/Cilium cluster:

```sh
export KUBECONFIG="$PWD/.local/sandbox.kubeconfig"
export CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make deploy CONTEXT=kind-tsunami-lab RUNTIME_CLASS=gvisor
make configure-model-fixture CONTEXT=kind-tsunami-lab
make model-fixture-eval CONTEXT=kind-tsunami-lab
```

The fixture deliberately asks for an unapproved origin first. The broker denies it. It then emits the own-document baselines, cross-user read, and validation call. The evaluator compares the actual trusted result against each fixture's known behavior.

Reports are collected by the exact run IDs printed by that evaluation Job, rather than whichever run happens to be newest. Failures and inconclusive results are retained. `execution: live` identifies use of the model gateway; `provider: fixture` and the report's mode explicitly identify that no live model was involved.

`make verify-k8s CONTEXT=kind-tsunami-lab` remains the original five-case scripted suite. It never opts into model calls.

## Configure a real provider

For OpenRouter, use the dedicated [free-only setup](openrouter-free.md). The generic `responses` profile rejects OpenRouter endpoints and credentials to prevent accidental paid routing. Other provider configuration below can incur costs and is not used for the free trial.

This is an operator action, not something an agent can request. Select an explicit model ID that supports this adapter and place its API key in a protected file outside Git, for example under `.local/`. Do not paste the key into a shell argument or chat.

```sh
python3 scripts/configure-model.py \
  --provider responses \
  --model YOUR_EXPLICIT_MODEL_ID \
  --base-url https://api.openai.com/v1/ \
  --key-file .local/model-api-key

# This is a separate, explicit action and can incur provider costs.
make live-eval CONTEXT=kind-tsunami-lab ALLOW_MODEL_CALLS=true
```

Setup creates a Kubernetes Secret through stdin and mounts it only into the control plane. It restarts that Deployment to load the key, including when only the Secret changed. Configure/rotate between assessments: this restart can interrupt active work. The run pins the model route and policy fingerprint; changing that configuration does not silently move an existing run to another model.

The image contains the build host's public CA bundle so the scratch image can verify HTTPS. The build host and its trust store are therefore trusted inputs. Real provider mode requires HTTPS; redirects and environment proxies are refused. Tests check rejection of an untrusted certificate and prevent credential forwarding on redirects.

The fixture's HTTP exception always substitutes a fixed synthetic key and accepts only a local lab endpoint. It cannot send a real model credential over cleartext HTTP.

To disable the model path and remove its provider Secret:

```sh
make disable-model CONTEXT=kind-tsunami-lab
```

Use the durable Stop API to stop active assessments. Disabling or reconfiguring a provider is a maintenance operation, not a replacement for Stop. An explicitly configured provider can persist across application redeploys until disabled.

## Current limits

| Control | Implemented limit |
|---|---|
| Calls reserved per run | 8 |
| Upstream HTTP attempts per run | 8, including adapter retries |
| Concurrent inference | 1 per run; 2 in this server process |
| Model HTTP body readers | 2; Stop does not wait for those slots |
| Conversation request | 64 KiB; at most 48 messages |
| Completed reply | 128 KiB; at most 512 chunks |
| Model output request | 1,024 tokens |
| Model call duration | 20 seconds or the earlier run deadline |
| Assessment duration | Existing 90-second run deadline |

SDK retries are disabled. If the adapter performs a protocol recovery request, that request must pass the same transport reservation. Failed or interrupted attempts keep their reservation. Missing usage stays uncertain, including across a restart; later success does not erase that uncertainty.

New calls also persist closed error categories and gateway timing metadata. A pending record after a crash stays unknown. The adapter can emit zero counts when upstream usage is missing; those counts do not prove zero usage. See [failure diagnostics and the offline harness](inference-diagnostics.md). Earlier generic `model_error` records cannot be assigned a more precise cause retroactively.

HTTP bodies have a five-second read deadline and responses have a thirty-second write deadline. These bound slow clients; they are not a general multi-tenant denial-of-service guarantee.

These are call, byte, time, and requested-output limits. They are **not a dollar spending guarantee** or an exact input-token reservation. Reported totals cover usable completed turns; failed or truncated turns can be billed without usable accounting. Aggregate customer/account spend reservations, price schedules, billing reconciliation, and distributed concurrency limits remain future work. An authorized operator can create additional runs; a compromised worker cannot create them with its run credential.

The 1,024-token cap and 20-second timeout may be too small for some reasoning models. One free-model trial passed under these limits with reasoning disabled. Other models and repeated runs still need evaluation. No default paid model is selected.

Stop closes this system's in-flight provider request and prevents new authorized attempts. It does not prove the provider immediately stopped computing or billing. Requests already accepted by a target cannot be undone either. A production stop objective still needs failure and partition testing.

The workflow is deliberately guided: it supplies synthetic accounts, object ownership, approved paths, and a versioned testing procedure (`object-ownership/v1`). It does not discover an arbitrary application's ownership rules. The original model-path reports use an offline fixture; the separate free-model reports record one real paired trial. Neither forms a discovery benchmark.

Only synthetic evidence is supported here. `store: false` requests stateless Responses behavior; it is not a claim about every provider's data retention. Customer evidence handling needs a separate review before real assessments.

## References

- [OpenAI Responses API](https://developers.openai.com/api/reference/go/resources/responses) and [function calling](https://developers.openai.com/api/docs/guides/function-calling).
- [Mecatl provider adapter](https://github.com/stacklok/mecatl/tree/provider/openai/v0.3.0/provider/openai).

The checked-in versions and local tests are the reference for this prototype. A successful live sample is not a provider availability or production behavior guarantee.
