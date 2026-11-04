# Inference failures and evaluation outcomes

The catalog's two follow-up model failures only recorded `model_error`. HTTP 200 headers were observed, but that does not prove a complete or usable model turn arrived. The old records do not establish whether the provider stalled, hit an output limit, or failed in another way. I have kept those causes unknown.

This milestone adds diagnostics for future calls and a local failure harness. It does not increase timeouts, output limits, target scope, call quotas, or retries. It makes no external model calls.

## What a new call records

The gateway persists a pending call record with its quota reservation, then updates it after the adapter returns. Fields include the sequence, start/deadline, whether the run or call supplied that deadline, observed HTTP status, header and first adapter-chunk timings, wire attempts, buffered/released tool counts, usage status, and a closed outcome code. Reports include a small call table.

Examples of codes are `upstream_http_429`, `upstream_event_http_429`, `model_deadline_exceeded`, `model_cancelled`, `model_output_token_limit`, and `model_stream_truncated`. Unknown failures remain generic. `model_cancelled` means the gateway observed context cancellation; the code alone does not identify who caused it. An observed deadline does not identify why the provider took that long.

The record contains no prompt, provider response text, tool arguments, authorization headers, raw error message, or provider request ID. That restriction applies to these diagnostics; assessment evidence and the conversation sent to the provider have their own data boundaries.

Times are measured at the gateway. The first adapter chunk is not necessarily the first network byte or token. Header timing is the last observed attempt's response, measured from the logical call's start. SDK retries stay disabled, and any adapter recovery request still consumes an upstream reservation.

Only the initial reservation and HTTP 200 header observation are persisted during the call. Chunk progress is saved when the call finishes. A process crash can therefore leave a pending record with an unknown outcome. Restart does not replay that call or invent its result.

## Partial replies and usage

Tool calls are buffered until a bounded, completed turn passes the gateway checks. Failure, Stop, invalid usage, or an incomplete stream returns no reply to the worker. A released tool count describes the gateway's completed reply, not proof that the worker executed a tool. The broker checks every later target request independently.

The pinned adapter emits a zero-valued usage chunk even when the wire response omitted usage. A zero-input/zero-output count stays uncertain for our nonempty requests. `usage_chunk_observed` means an adapter chunk was seen; `usage_accepted` means a successful turn included nonzero counts within our bounds. Neither is an independent billing check. Failed calls retain reservations and uncertain usage, even if some usage arrived. A later success cannot erase that uncertainty.

## Reproduce without a model key

```sh
make inference-diagnostics

go run ./cmd/tsunami catalog-analyze \
  --source examples/assessment-catalog/initial-live \
  --out artifacts/inference-diagnostics/initial-analysis

go run ./cmd/tsunami catalog-analyze \
  --source examples/assessment-catalog/feedback-live \
  --out artifacts/inference-diagnostics/followup-analysis
```

The diagnostic command uses the real Mecatl Responses adapter and gateway against loopback fixtures with a fixed synthetic key. Fourteen cases cover normal completion, omitted usage, HTTP 401/429/503, no headers, headers without a turn, Stop after headers, truncated/partial tool calls, output limit, unknown incomplete response, an in-stream rate limit, invalid usage, and an oversized reply. Each fixture receives one request. No agent tool or target request runs. Only this trusted harness shortens its synthetic run deadlines to test stalls quickly; deployed limits are unchanged.

This is a native integration test, not a real-provider reliability test, Kubernetes partition test, or production Stop measurement. The fixture includes private-data canaries to check that diagnostics do not expose upstream text.

## Keep quality and operational failures visible

New catalog summaries add outcome categories and separate counts for operational failures, incomplete assessments, assessment misses, and stopped runs. A failed last call is a model failure; a pending call is unfinished. A later worker failure after a successful call is not blamed on an older recovered model error. These are observed categories, not a full root-cause analysis.

The offline analyzer reads stored run verdicts and validation details. It accepts only the supported procedure versions/digests and writes a separate projection outside the archive. It does not revalidate evidence or edit original records. Generic historical errors remain `model_failure_unclassified`.

| Archived sample | Original matched | Raw missed planted cases | Operational failures | Assessment misses | Incomplete assessments |
|---|---|---|---|---|---|
| Initial free trial | 3/4 | 1 | 0 | 1 | 1 |
| Feedback free trial | 2/4 | 1 | 2 | 0 | 0 |

The follow-up still failed to deliver two expected results, including one planted finding. Separating that operational failure does not improve the end-to-end score. No extra live trial was run to obtain a better score. See [recorded diagnostics and analysis](../examples/inference-diagnostics/README.md).

Next I would get the procedures reviewed by the security engineer and build a separate, less guided dataset. More live trials should have explicit limits and retain failures. Automated retries, longer timeouts, private model execution, and general customer assessments are not part of this milestone.
