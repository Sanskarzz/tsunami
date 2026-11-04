# Review follow-up

Updated: 2026-11-05 (local time). This implements the first deliverable in [the review report](review-report.md): fix its five P2 defects and create a repeatable source-verification gate. The original review and historical lab records retain their meaning.

## Resolved findings

| Finding | Change | Regression evidence |
|---|---|---|
| Operator credential redirects | The Go operator client rejects redirects, including with an injected client. Its default transport ignores ambient proxies; client injection does not mutate the caller. Kubernetes authenticated calls also reject redirects. | A real same-host, different-port HTTP redirect never reaches the receiver. |
| Rejected cleanup recorded as accepted | Kubernetes response handling is method-specific. DELETE conflicts/errors remain retryable; 404 means already absent. Cleanup attempts both Job and Secret even if one fails. | A running reconciler observes conflicts, keeps `deletion_requested_at` empty, preserves revoked permission, retries, and records acceptance only after both deletions are accepted/absent. |
| Application history breaks collection | Default collection selects the five legacy scenarios from paged metadata before fetching reports. Exact-ID collection supports application reports and never downloads history. Selections are validated before file writes. | Mixed application/legacy history across pages selects the latest expected legacy reports; exact application collection bypasses listing. |
| Valid exports exceed the client limit | Individual responses have a separate 16 MiB ceiling; inline engagement exports have a 192 MiB lab ceiling with explicit oversize errors. The Go CLI, Python collector and local schema checker use the same export ceiling. Export requests get a separate 30-second default client timeout. | A legal 12-child engagement above 4 MiB is decoded with all 72 artifacts intact. Oversized bodies and trailing documents are rejected explicitly. |
| Selected build is not actually deployed | The lab renderer binds native manifests and `TSUNAMI_IMAGE` to `IMAGE`. Verifiers, probes, model fixtures, private components and application tripwires select the installed control build. Tool builds/configuration support `TOOL_IMAGE` independently. Private/tools deployment helpers preserve other configuration when selecting the native build. | Renderer tests and Make dry runs assert custom native/tool tags, worker configuration, preserved third-party images and rejection of inconsistent control/worker configuration. |

The stale current engagement-demo message about missing finding-schema export was also corrected. Archived example messages were not rewritten.

## API compatibility change

`GET /v1/runs` now returns a tenant-bound metadata page:

```json
{
  "items": [
    {
      "id": "<run-id>",
      "tenant": "<authenticated-tenant>",
      "scenario": "application",
      "state": "completed",
      "execution": "scripted",
      "created_at": "<timestamp>"
    }
  ],
  "next_cursor": "<last-returned-run-id>"
}
```

Use `?limit=50&cursor=...`; limits are 1–100, default 50. `next_cursor` is omitted when there are no more matching records. Ordering is by stored run ID, not creation time. Listing is not a snapshot under concurrent inserts; exact campaign IDs remain the reliable collection path. Full captures/results stay on authenticated `GET /v1/runs/{id}` and engagement exports.

This intentionally replaces the prototype's unbounded array of full runs. The in-repo collector was migrated. Any external client using the old array must migrate, and a newly built collector requires the updated server. The controller's internal full store scan has not yet been redesigned.

## Export bounds and remaining storage work

The 192 MiB ceiling accommodates current lab allowances and JSON/base64 expansion: 20 children, current maximum 12 forwarded 64 KiB bodies per child, bounded denied captures and metadata. It is a bounded client contract, not a measured memory/throughput guarantee at that maximum. Raising procedure or capture allowances requires reviewing this bound.

This change does not solve the memory cost of building large inline exports, the verifier's resource limits at worst-case sizes, or reconciliation's full-history scan. Separate authorized artifacts and durable metadata remain the next storage design work. The regression establishes the reported legal export above 4 MiB; it is not a maximum-size load qualification.

## Verification

From the repository root, on Linux with Go, make, Python 3 and `/usr/bin/curl` available:

```sh
# If the validator dependencies are not installed, use an ignored local venv.
python3 -m venv .local/test-venv
. .local/test-venv/bin/activate
python3 -m pip install -r scripts/requirements-test.txt

make verify

# Optional: needs the separately obtained local reference toolkit.
make verify-toolkit TOOLKIT_PATH=../vapt-toolkit
```

`make verify` runs fresh Go race tests, vet, Python tests and checks four materialized example bundles, their 32 artifact files and the 20 archived live-trial hashes. It does not download toolkit content or call a model/customer target. `verify-toolkit` additionally checks the three actual local toolkit schemas with remote reference retrieval disabled.

During this follow-up, the native gate and actual local toolkit-schema checks passed. All ten Python tests passed, including the socket-based redirect regression that was restricted in the other session. No fresh Kubernetes deployment, active curl Stop campaign, partition/load qualification or external inference was performed. Checked-in earlier Kubernetes results remain historical evidence for their recorded builds.

## Next implementation order

1. Make permission, dispatch/execution, active activity, cleanup and assessment outcomes explicit. Cover graceful shutdown/store lifetime and keep authenticated revocation available when an optional connector/model route is unhealthy.
2. Hold an actual request behind a gVisor curl worker, revoke run/engagement permission and record target cancellation, last admitted work and cleanup separately. Repeat relevant outages/restart scenarios before making a production Stop claim.
3. Connect a thin CLI/UI to the owned-application APIs. Preserve scripted/model labels, missing prerequisites and uncertainty. The static templates remain disconnected design artifacts.
4. Add a useful reviewed HTTP procedure or another independent owned application, with positive/negative/ambiguous controls and less manual prerequisite binding.
5. Complete the selected HTTP pilot's identity, isolation, secret/evidence lifecycle, TLS, historical-report compatibility and independent boundary checks.
6. Add VM/MCP, mobile, network and specialists only when a useful capability justifies their separate enforcement and qualification work. Interface discovery can continue as design work.

This changes development priority, not the long-term product goal. The repository remains an owned-lab prototype rather than a customer-ready service.
