# Form first, questions when needed

The first working slice is implemented: a local form, durable typed questions/answers, protected bearer access, explicit four-read validation, inferred supported bindings and atomic application/assessment-draft registration. See [the implementation and usage guide](application-onboarding.md).

The setup assistant currently uses **checked rules, not a live LLM**. The assessment uses the existing scripted Mecatl/Kubernetes path. General password/SSO login, documentation ingestion, frontend discovery and model-assisted planning below remain intended extensions. This document describes the product direction; the implementation guide states the actual supported boundaries.

The customer should not need to know every prerequisite before beginning, write attack prompts, create proof hashes, choose JSON pointers or tell Tsunami which vulnerability to find. Start with a short form. The setup agent uses the supplied facts to identify gaps, asks focused questions when necessary and proposes the supported assessment. Trusted backend checks and explicit user approval control whether testing can start.

## Intended sequence

1. **Add application.** Collect the name/environment, frontend URL, initial approved hosts/actions, authorization reference, exclusions, test window and limits. Offer protected test-account fields and optional API documentation/specification. Unknown API/authentication details can stay unknown; do not require invented values to submit. The initial action saves a draft and reviews information, not an assessment run.
2. **Review what is known.** The setup agent reads non-secret metadata and checks the prerequisites of supported procedures. It records known, missing, conflicting, unsupported and unverified facts separately. An omitted value is not permission to guess.
3. **Ask only when needed.** Show the setup conversation after submission, alongside a small readiness checklist. Enable a response field when a question is pending. Ask one focused question, or a small related group, explain why it matters and retain prior answers. Let the user revise earlier form details. Do not ask again for facts already supplied and still valid.
4. **Validate approved prerequisites.** If permitted, trusted integrations perform bounded login/current-user and legitimate own-resource checks. Record actual observations and failures. Unsupported SSO/MFA, missing data or an unclear sharing rule makes the affected procedure unavailable; the agent must not loop forever or invent a workaround. The user may fix access, narrow the assessment or leave unsupported tests out.
5. **Review the proposed assessment.** Show exact hosts/routes/actions, test identities/roles, applicable procedures, unavailable tests and reasons, data/model handling, limits and expiry. The agent explains its recommendation; the backend evaluates the typed prerequisites and permissions. Readiness is per procedure, not a promise of complete application coverage.
6. **Confirm and start.** Enable Review and start only for a valid plan. The final action records approval of that exact scope/version and starts the supported workflow through the existing engagement/run lifecycle. Saving details, answering a chat question and completing validation must not silently launch security tests. Lost acknowledgements are recovered from stored IDs, without automatic mutation retries.
7. **Observe and inspect.** Show progress, missing evidence, Stop, supported candidates and their captured evidence. A question that arises during execution pauses the affected work at a defined checkpoint. It cannot widen the running plan; changed permissions require a new reviewed scope. Incomplete work and unavailable tests stay visible.

## Example: frontend URL without its API

The user enters an authorized dev frontend URL and supplies test-access references. Tsunami does not assume the frontend is the API host.

The assistant may ask: “I have the frontend URL. Which backend API does this environment use? You can provide its URL or an API specification. I need this to prepare the authorization tests.”

If the user has separately permitted a bounded frontend inspection, it may identify a candidate API host from the observed configuration or requests. That host is a discovered suggestion, not automatically approved scope. Ask the user to include it explicitly, then show it in the final reviewed plan. No credential forwarding or requests to newly discovered hosts before permission.

If an API specification already supplies this information, use it and ask only about unresolved facts. A specification, page, script or tool response is untrusted input: it cannot direct the agent to ignore scope, disclose credentials or call another service.

## What can be checked before testing?

Separate two activities rather than claiming the system has done everything without contacting the target:

| Activity | Permission and behavior |
|---|---|
| Information review | Parse form fields, answers and supplied documents without target requests. Identify supported authentication and missing prerequisites. Reading an uploaded document must not execute it or follow remote references. |
| Connection/prerequisite validation | Requires an explicit limited preflight grant for the relevant host, routes, authentication action, request count and expiry. Login may be a state-changing POST. Own-object reads may expose data. These requests still pass through trusted scope, credential and limit enforcement. |
| Security assessment | Requires approval of the final procedure scope. Includes the cross-user/property probe or other separately implemented test. Preflight access is not a general testing permission. |

Entering credentials alone authorizes none of these requests. A user may approve bounded preflight as part of the form; the UI must show what it permits. If no preflight is approved, the assistant can ask and prepare metadata while keeping observations marked unverified.

## Agent reasoning and backend authority

The setup agent chooses a useful next question and recommends applicable procedures. A typed backend readiness evaluator checks requirements such as:

- The actual API host/routes/actions are approved, with exclusions, limits and a valid time window.
- The installed execution/authentication integration supports this application. An HTTP 200 or successful browser login does not itself establish a usable API identity.
- Required test identities are distinct and their roles are understood. Credentials have usable references and sufficient lifetime for the proposed work.
- Legitimate baseline resources and the intended access/sharing rules are available for each selected procedure. Different owners do not necessarily imply forbidden sharing.
- Required observations were obtained under the appropriate preflight permission and belong to the current application/access configuration.
- The registered plan can be compiled and enforced by the broker, and its scope/data-handling approval matches the current version.

An LLM's “ready” message cannot satisfy missing fields, fake a login observation, approve a new destination, enable an unsupported tool or start a run. Scope, credential and rate/Stop enforcement stay outside the agent. User answers are proposals until parsed and validated; answering in natural language is not a replacement for server-side permission records.

The conversation needs durable tenant-bound questions/answers, typed facts with their source and validation status, revisioned plan state and a pending-question ID. User replies may arrive after refresh/reconnect; an answer must match the pending question/current revision. Duplicate or stale replies must not create duplicate work. Stop/revocation applies to any active preflight or assessment operations, not only the final Job.

The essential states are **draft → reviewing → needs input / validating → ready for approval → approved → running**. An unavailable procedure, failed validation or revoked/expired permission has an explicit reason. “Ready for approval” is a prerequisite outcome, not an authorization grant or a security verdict. Changes that affect scope/access invalidate the relevant readiness observations and approval; the backend must not silently reuse the previous plan.

## Credential and model boundary

The setup conversation receives credential aliases and safe status, not passwords, cookies or raw tokens. If access is missing, it opens protected input fields rather than asking the user to paste a password into chat. A supported trusted login integration obtains a session under the preflight permission; the test worker continues to use aliases through the broker.

This needs a protected credential lifecycle beyond the existing lab's startup Secret. Do not put real customer passwords into bbolt plaintext, chat history, prompts, reports or logs to make onboarding appear complete. Expiry and renewal need explicit behavior; wrong credentials must not trigger unbounded retries or lock accounts.

An external model is optional and requires an approved data-handling policy as well as existing free-only development restrictions. Customer documents, responses and credentials must not be sent to a provider merely because a free model key is configured. Deterministic readiness checks can operate without a model, but the UI must label that mode honestly; a rule-based checklist must not be presented as autonomous agent reasoning. The current scripted assessment demo is not this proposed setup agent.

## Implementation boundary and remaining work

Build this as a guided onboarding path attached to the existing application/engagement lifecycle, not a general-purpose chatbot or arbitrary shell tool. Durable sessions, safe metadata APIs, encrypted bearer configuration, typed readiness, bounded JSON GET preflight and the form/conversation/review UI now exist. The checked question workflow is not model-assisted planning. General login integrations, document/frontend discovery, broader response contracts and customer release controls remain work. Reuse existing broker scope checks, reservations, evidence, revocation and run reconciliation where applicable; do not bypass them for setup traffic.

Qualify the journey first on owned synthetic applications: a missing backend URL, supplied spec, unsupported login, failed/expired identity, conflicting sharing rule, suggested out-of-scope host, late answer, changed scope, lost response, active-preflight Stop and restart/recovery. Only then qualify an authorized external dev application with its actual contract. The AccuKnox integration still needs the requested sanitized API/authentication details and exact testing scope; no login endpoint or expected finding is assumed.
