# How the VAPT toolkit informs Tsunami

The friend's toolkit supplies testing knowledge. Tsunami supplies a constrained execution and evidence path. We have not installed the toolkit, copied its scripts/prompts, executed its commands, or integrated its VM/MCP environment.

The initial catalog uses these ideas from the locally reviewed toolkit:

| Toolkit material | Tsunami adaptation |
|---|---|
| API playbook: test object access with a second valid account | Opaque object ownership procedure with two legitimate baselines and a cross-user probe |
| API playbook: rejecting a fabricated ID does not prove access control | Missing-object procedure always remains inconclusive |
| API playbook: include an unauthenticated session | Explicit anonymous procedure that omits the target Authorization header |
| API playbook: distinguish normal users from administrators | Read-only admin-summary procedure with an administrator baseline and a member probe |
| Methodology: authentication and authorization are different | A valid session does not justify private-object or administrator access |
| Methodology/reporting: prove the finding and preserve reproducible evidence | Broker-owned captures, fixture-specific validation, request IDs, reproduction steps, and remediation |

The shared-object positive control is our additional guard against a simplistic rule that every cross-owner HTTP 200 is a bug. Each application's intended access rules still need a trustworthy source.

The five JSON procedures are original, narrow definitions, not a machine translation of all toolkit playbooks. Their review status explicitly says security-engineer review is pending. The evaluator and validator use known synthetic ownership/role rules; they do not understand arbitrary customer applications.

The engagement foundation now produces a projection that can be validated against the local engagement schema. A read-only inventory hashes 44 scripts, 26 checklists, three schemas and four prompt files; the role document describes eight roles. It records checklist identifiers without executing test instructions or copying their content. All script entries remain marked `adapter_not_implemented`. The toolkit has no detected license file, so redistribution terms are still pending.

The configured-application milestone adds review-backed finding/evidence schema projections and original CSV/Markdown exporters. The local checker validates all three actual schemas and materialized reviewed evidence without executing the toolkit's utilities. A local mapping adapter links imported checklist IDs to sampled procedure results; it deliberately does not mark a broad standard item fully completed from one probe. See [application contracts and limits](application-adapters.md).

The first original [shell/curl adapter](tool-execution.md) executes the configured read procedure through the broker. It does not run or grant compatibility to an existing toolkit script; those inventory entries remain unimplemented. Its output still needs broker validation rather than trusting a script's suggested severity.

General recon/scanner adapters, full reviewed checklist execution, the existing reporting utilities themselves, and specialist-agent coordination remain work. Scanner or MCP integration needs enforcement at its actual execution boundary. Running a network-capable script inside a container would not automatically give it scope enforcement. Raw network and dynamic mobile work need separately qualified profiles, not just the current HTTP broker. See [engagement implementation](engagements.md) and [the full product plan](product-roadmap.md).

Next useful input from the security engineer is a review of these procedures: prerequisites, intended sharing rules, what constitutes evidence, ambiguous outcomes, and a sanitized real assessment example when one becomes available. This should guide additional procedures before expanding the tool catalog.
