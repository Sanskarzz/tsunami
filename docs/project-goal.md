# Tsunami: project goal and purpose

Updated: 2026-11-04.

This is my understanding of the project based on our discussions, the original take-home and the work in this repository. It describes the intended product, not a claim that the complete product exists today. [track.md](track.md) records implementation status; [design.md](design.md) explains the architecture and user journey.

## 1. The main goal

**Build a platform where a customer can authorize an assessment, and agents can perform useful security tests within enforceable boundaries, producing evidence that a person can inspect and trust under stated assumptions.**

The customer should provide the application, approved access, relevant business rules and permission. Tsunami should supply the testing knowledge, execution workflow, controls and reporting.

The first useful experience should be understandable: add an application, connect test accounts, choose a built-in profile, approve the plan, run it, inspect findings and stop when needed. A customer should not need to write prompts describing individual attacks or understand Kubernetes internals to use it.

The project started from the “agents that attack, safely” take-home. Its two central concerns remain: an agent acting outside approved targets, and one customer's credentials/evidence reaching another customer. The implementation must treat these as enforcement problems, not instructions that a model might follow.

## 2. The problem we are trying to solve

Security assessments involve more than running a scanner. Someone establishes scope, obtains appropriate access, chooses relevant tests, interprets responses, distinguishes normal behavior from bugs, captures evidence, validates impact and writes a usable report.

Agents may help with that work, but they can misinterpret application behavior, follow hostile instructions in content, request an unapproved tool, consume resources or claim findings without enough proof. A script can also make unsafe requests without involving an LLM at all.

Tsunami's purpose is to coordinate useful assessment work while controlling the authority each participant receives. The testing method, infrastructure boundary and evidence rules must fit together.

An “authorized” sentence is an input to a permission workflow. It is not permission for arbitrary network access. A “vulnerability found” sentence is an observation to investigate. It is not a verified result by itself.

## 3. The customer experience we want

Imagine a customer wants their staging books application tested. They should be able to:

1. Register the application and identify approved/excluded areas.
2. Provide dedicated test identities through a protected flow and explain relevant rules, such as which records are private or intentionally shared.
3. Choose a built-in profile such as web/API authorization testing, with recommended defaults and clear prerequisites.
4. Review what Tsunami proposes to do, including permissions, limits, unavailable tests and data/model handling.
5. Approve and start the assessment without manually sending every request.
6. Follow progress, usage, denials, failures and incomplete work, and stop the assessment when necessary.
7. Inspect evidence-backed candidates, review actual impact, export a report and request a retest after remediation.

Tsunami supplies the cross-user access test. The customer does not type “try reading Alice's book as Bob.” It should obtain suitable test facts through bounded discovery or guided setup where possible. It must ask for help or report uncertainty when it cannot establish legitimate identities, objects or the intended access rule.

This does not promise useful assessment from a URL alone for every application. Authentication, business rules, private connectivity and suitable test data can require customer input. The product should make that input understandable instead of exposing manual proof hashes and implementation details as the normal experience.

## 4. What agents should do, and what the platform must own

| Agents and reviewed tools may do | The trusted platform must own |
|---|---|
| Interpret permitted application information | Tenant identity and approved scope |
| Propose applicable tests or choose the next allowed action | Which profiles/tools are supported and eligible |
| Execute bounded tool operations | Authorization, credential delivery and actual network enforcement |
| Analyze returned observations and suggest explanations | Durable reservations, deadlines and cost/resource limits |
| Draft finding descriptions and remediation ideas | Evidence provenance, validation and review history |
| Request completion or assistance | Stop/revocation and workload/executor lifecycle |

An agent may be wrong or compromised. It must not be able to approve its own scope, enable new tools, read another customer's artifacts, mint privileged credentials or confirm its own unsupported claim.

The model is an aid to reasoning and coordination. It is not the security boundary. Scripted procedures remain useful for repeatability and infrastructure testing; their success must not be presented as autonomous discovery quality.

The intended onboarding is form-first: the user supplies initial application/access details, then a setup agent asks only for missing or conflicting information, such as the backend API behind a frontend URL. The user does not need to specify attack steps or mechanical proof bindings. The backend checks procedure readiness and permissions, and a final explicit confirmation starts testing. Credentials use protected fields, not chat. See [the planned onboarding/readiness flow](onboarding-readiness.md); the [first checked workflow](application-onboarding.md) is implemented; a live-model setup planner, general login and automatic discovery remain work.

## 5. The role of the VAPT toolkit

The friend's toolkit provides practical assessment knowledge: workflows, test ideas, expected inputs, evidence/reporting conventions, checklists and specialist-role concepts. It helps connect infrastructure design to actual security work.

The goal is to support useful toolkit-inspired capabilities through reviewed adapters and appropriate execution profiles. It is not enough to put every script into an image and let an agent run it freely. Existing scripts may need permission changes, safer defaults, bounded inputs, cancellation, evidence validation or replacement.

The full ambition includes web/API, network, mobile, cloud/crypto where appropriate, reporting, VM/MCP execution and specialist coordination. These modes do not all fit one sandbox or gateway. Each needs its own scope, identity, resources, evidence rules and Stop behavior at the actual execution point.

Current use is narrower: original authorization procedures, local inventory, partial checklist references, schema-compatible exports and an original curl adapter. Original toolkit scripts are not executed or copied. Its redistribution terms and the friend's actual VM/MCP contracts remain unresolved inputs. Reference material is not the same as a supported execution integration.

## 6. The role of Mecatl and Kubernetes

Mecatl provides the agent loop and tool-call coordination. Kubernetes provides workload scheduling and lifecycle. Tsunami adds the assessment-specific control plane: applications, scope approval, profiles/plans, execution permissions, broker/connector enforcement, budgets, evidence, review and reporting.

The current real Mecatl integration is not an implementation of every Mecatl feature. Durable assessment state and the Job reconciler in this repository should be described by what they actually do. Exploring another project's architecture must not be presented as a personal implementation.

Containers can support many HTTP and offline analysis tasks, provided the runtime and network boundary are qualified. Some VM, raw-network and mobile/device work needs a different executor. Kubernetes can coordinate those executors without pretending that everything runs safely inside the current gVisor worker.

## 7. The outcomes that matter

| Outcome | Evidence we should require |
|---|---|
| Useful findings | Reproducible observations, legitimate baselines, intended-policy context, appropriate review and fixed/negative controls |
| Fewer misleading conclusions | Visible false positives, misses, incomplete prerequisites and inconclusive outcomes; no invented coverage |
| Enforced scope | Deliberate bypass tests, including direct clients, redirects, DNS changes and indirect/remote requests for each profile |
| Customer separation | Adversarial checks across identities, workloads, networks, credentials, storage and exports |
| Effective stopping | Recorded revocation, last accepted requests, cancellation/drain and executor cleanup under stated failures and load |
| Bounded cost and resources | Reservations before work, explicit ceilings, uncertain usage tracking, fairness and tested failure behavior |
| Understandable operation | A usable launch/status/stop/review/report journey with observable failure states |

“No issue observed” is limited to the checks completed. A schema-valid report does not prove a finding is correct. A hash does not prove evidence authenticity against a compromised trusted component. A pod deletion acknowledgement does not prove every process or target-side operation has stopped.

Safety here means specific controls under stated assumptions and tests. It cannot mean that every possible approved request is harmless or that all infrastructure compromise is prevented.

## 8. Where the project is now

There is a working owned-lab prototype: configured HTTP authorization tests, a real Mecatl loop, gVisor/Cilium experiments, a trusted broker, durable Stop, synthetic private connectivity, review/export contracts and a fixed command-to-broker adapter.

The VAmPI vulnerable/fixed example supplies a useful concrete result. It uses disposable accounts, known object/rule assertions and scripted execution. The curl adapter tests a controlled command path; it does not add broad discovery or unmodified script compatibility. Live-model experiments are separate and have mixed recorded outcomes.

A local UI and narrow form/conversation/bearer-validation onboarding path now work. General onboarding/login, model-assisted profile planning, general application-agent behavior, broader executors and customer-grade infrastructure are still work to do. Customer data handling, general redaction, production identity/key lifecycle, distributed budgets and measured 1,000-agent capacity are not complete. Production 60-second Stop is not verified.

The [HTML templates](templates/index.html) visualize the intended product only. Their fictional findings, reviews, metrics and statuses are not additional test results.

## 9. What a useful first delivery looks like

The first delivery should provide one complete assessment experience: an owned or explicitly approved application, a useful built-in test, clear setup/approval, controlled execution, visible progress, active-request Stop and an inspectable report. Include vulnerable, fixed and ambiguous controls.

Build the small UI/CLI on the same backend APIs. Keep repeated scripted qualification separate from live-agent evaluation. Add agent choice and additional procedures only with permissions and evidence rules that the platform can enforce.

A limited customer pilot requires completed controls for that profile: customer identity, appropriate secrets/data protection, operational procedures, independent review and measured failure behavior. It need not wait for every mobile/network capability, but neither should a narrow pilot imply the full toolkit-inspired vision is complete.

## 10. Why this project exists beyond the interview

The immediate opportunity is to show Axel a working extension of the assignment, with real code, reproducible results, trade-offs and honest limits. His interest is a reason to prepare a clear demonstration, not a reason to overstate readiness or predict a hiring outcome.

The longer-term purpose is a personal open-source/portfolio project at the intersection of Go, Kubernetes infrastructure, MCP/agent execution and practical security assessments. Its value should continue independently of this job opportunity.

The intended result is software that a security engineer can use and challenge, and that a customer can understand and control. More scripts, more agents or a polished UI are useful only when they improve that assessment workflow without silently weakening its boundaries.
