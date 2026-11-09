# Mecatl: current integration and bounded reuse

Reviewed against Tsunami's pinned `github.com/stacklok/mecatl/engine v0.18.1` and `provider/openai v0.3.0`, not an assumption about the latest upstream branch. The default configured-application demo uses the real engine with a scripted `mockllm` provider. It makes real broker requests; it does not autonomously discover vulnerabilities.

## What Mecatl supplies today

[worker.go](../internal/worker/worker.go) constructs an `agent.Engine`, typed tool catalog and explicit permission policy. Exactly `http_request` and `finish_assessment` are registered and allowed; unexpected permission asks are denied. It uses a run-ID session, in-memory ledger/workspace, host-owned layered prompt, cancellation context and limits of 20 tool calls, three consecutive failures and 20 scripted turns (eight turns on the optional live route). Worker logs retain only event kind and turn, not raw tool results.

The ledger/workspace are transient engine infrastructure. Durable assessment records live in Tsunami's control plane; engine conversation state is not the authoritative finding or Stop record. No shell, filesystem tool, MCP tool, subagent or general network tool is registered.

The optional live route substitutes `remoteModel`, a provider adapter that calls the trusted broker's model gateway. The control plane selects provider/model and owns provider credentials, bounded calls/output and usage accounting. Configured application assessments reject live execution; this route belongs to separately qualified legacy/catalog experiments.

## What Tsunami wraps around the engine

| Tsunami responsibility | Implementation |
|---|---|
| Tenant/operator authorization, immutable approved scope, child-run reservation and linked retest | [engagement.go](../internal/platform/engagement.go), [scope.go](../internal/platform/scope.go), [store.go](../internal/platform/store.go). |
| Checked onboarding, protected origin-bound access, encrypted setup state and separately approved four-read preflight | [onboarding](../internal/onboarding/model.go), [setup persistence](../internal/platform/onboarding_store.go), [preflight](../internal/platform/onboarding_validate.go). |
| Run-authenticated HTTP intents; target credentials remain trusted-side; origin/address/account/path/method/budget/redirect/DNS/TLS checks | [broker](../internal/platform/server.go), [application plans](../internal/application/plan.go). |
| Durable Stop gates, cancellation of admitted work, restart uncertainty and separate exit/cleanup observations | [lifecycle](../internal/platform/lifecycle.go), [store](../internal/platform/store.go), [Stop contract](stop-lifecycle.md). |
| Kubernetes Jobs, per-run worker access, runtime/security settings, reconciliation and network enforcement | [scheduler](../internal/kube/scheduler.go), [sandbox lab](sandbox-lab.md), [deployment](../deploy/sandbox/). |
| Reviewed fixed curl adapter through a private local broker bridge; no arbitrary shell command interface | [tool runner](../internal/toolrunner/runner.go), [execution contract](tool-execution.md). |
| Broker evidence capture, artifact hashes, deterministic prerequisite/verdict validation, safe finding views and exports | [assessment](../internal/platform/assessment.go), [finding view](../internal/platform/finding_view.go), [export](../internal/platform/engagement_export.go). |
| Bounded model gateway, provider restrictions, usage/error handling and diagnostics | [model admission](../internal/platform/inference.go), [provider](../internal/inference/provider.go). |
| Persistent sanitized activity, UI/CLI approval and evidence navigation, recovery and regression/qualification campaigns | [activity](../internal/platform/activity.go), [UI](local-ui.md), [CLI](operator-cli.md), [demo guide](demo-walkthrough.md). |

These are product enforcement and evidence responsibilities. Mecatl permissions are an additional restriction; the broker must independently reauthorize every action. A worker hook or a prompt cannot replace that boundary.

## Where to reuse more, if a demonstrated need justifies it

1. **Improve the harness regression corpus first.** Extend the existing scripted-provider tests with bounded malformed tool calls, untrusted target instructions, missing prerequisites, Stop races and budget exhaustion. Assert broker observations and deterministic outcomes rather than matching model prose. Reuse the engine loop and provider seam; do not build a second execution loop or consume selected-cluster history for every test.
2. **Use existing engine observability seams selectively.** The pinned engine exposes `ToolCallRecorder`, `Clock`, `Diagnostics` and lifecycle hooks in [`agent.Deps`](https://github.com/stacklok/mecatl/blob/engine/v0.18.1/engine/agent/loop.go). If harness timing is needed, adapt only allowlisted event kind, turn, duration and bounded code fields. Never forward generic events/arguments/output: they may contain evidence or secrets. Worker-reported events must be labeled worker observations and cannot assert broker admission, trusted validation or durable revocation. This integration is proposed, not present today.
3. **Evaluate bounded model choice only after reviewer feedback.** A separate offline campaign could compare scripted versus model-selected ordering of the same approved reads. Measure complete prerequisite coverage, valid verdicts, denials, calls/tokens and latency under a fixed budget. Use the existing broker-owned provider route; configured-application live execution is currently unsupported and would require an explicit contract change and qualification. Model text must never assign the trusted verdict.

Keep engine/session turn bookkeeping in Mecatl. Keep approvals, tenant-bound credentials, scope, request admission, durable Stop, evidence and verdicts in Tsunami. Do not adopt distributed sessions, generic hosted execution, automatic resume of uncertain security actions, arbitrary shell/MCP, multi-agent orchestration or additional sandboxes just because the upstream harness offers broader facilities. They do not solve a demonstrated gap in this five-minute slice.
