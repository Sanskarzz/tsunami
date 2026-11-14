# Tsunami operator console redesign

## Audit and direction

The existing embedded HTML/CSS/JavaScript already implements the trusted APIs and important race/recovery guards. The UX buries that work: repeated target cards compete with one another, setup presents unrelated fields together, navigation opens unselected contexts, the activity journal reads like internal event names, and finding proof competes with technical metadata. The existing 9–14px typography and muted labels make a recorded walkthrough difficult to read.

The journey is **Application → Baseline access → Scope approval → Assessment → Findings and retest**. Registration, baseline permission, final approval and Start remain separate. A registered application's scope can be reviewed directly; onboarding provides the separately authorized baseline path. Navigation must explain missing selection and provide the next valid action.

## Skill review

Read the locally installed `ui-ux-pro-max/SKILL.md` and its Python CLI, then searched cybersecurity assessment consoles, enterprise security SaaS, multi-step validation, keyboard navigation, tables, status feedback, permission confirmation, evidence disclosure, typography and timeline presentation. The generated system recommended Minimalism & Swiss Style, which fits the console. Its neon/black palette, operations landing page, external fonts and GSAP reveal animation do not fit this product and were rejected. Permission-specific search had no matching record; the narrower confirmation search supports explicit feedback, while the existing backend contracts determine permission behavior. Evidence/timeline searches were weak matches: their chart suggestions do not fit a persisted event journal. Use native disclosures and a textual timeline, drawing on the skill's explicit progressive-disclosure and accessible status guidance.

Adopt visible labels, 16px body type, 44px controls, 4/8px spacing, visible keyboard focus, current navigation semantics, textual statuses, bounded tables, long-token wrapping, explicit empty/error/stale states and reduced motion. Use local system fonts; no asset/package/CDN imports.

## Visual system

- Forest brand `#146b59`, stronger hover `#0f5144`, dark navigation `#142c27`.
- Canvas `#f4f6f3`, white surface, ink `#17332d`, secondary text `#52645d`.
- Success `#246443` on `#eaf5ee`; warning `#80510f` on `#fff5df`; danger `#a52d2d` on `#fff1ef`; information `#285979` on `#edf5fa`.
- Type: system sans; 16px body, 14px supporting text, 12px eyebrows, 20px section titles, responsive 28–36px page titles. System monospace for exact references only.
- Spacing: 4, 8, 12, 16, 24, 32, 48px. 10px panel radius, 6px controls, thin neutral borders, minimal shadow.
- One primary action per stage. Secondary actions outlined, permission revocation red and independently accessible. Focus ring dark teal with a white separation ring. No animation beyond brief hover feedback.

## Page design and implementation

1. Applications: searchable/filterable rows retaining distinct IDs/scopes; exact registration source, procedure, local access availability and actual latest saved attempt. Resume links recover saved IDs. A reviewed VAmPI integration shortcut is offered only when returned by the server. Saved history remains intact.
2. Setup: application information, integration choice, server-checked resources/questions, explicit four-read baseline review, then scope handoff. Required frontend metadata is secondary and explained; backend still requires it. New public HTTPS bearer/JSON setup and exact registered lab reuse remain distinct.
3. Scope: target, addresses, identities, action matrix, rule, budgets and expiry; allowed/blocked comparison; digest and assertions remain expandable. Checkbox binds the saved digest. Approval never starts work.
4. Assessment: contextual identity, architecture strip, independent permission/execution/activity/cleanup facts, recorded allowance and friendly timeline titles with exact event kind/source/reference disclosures. Uncertain Start keeps explicit same-key recovery.
5. Findings: authoritative verdict and expected/observed comparison first; selected baseline/probe proof references and check results; remediation and review state; raw artifacts and export remain explicit. Retest creates a separate draft and compares actual results.

Primary files: `internal/webui/static/index.html`, `styles.css`, `app.js`, `onboarding.js`. Keep handler and API contracts unchanged. Extend frontend and disposable browser tests where interactions change. No persistent state, images, deployment, network policy, credentials or Stop gates are changed.

## State/test risks

Preserve DOM IDs, event hooks, test selectors, raw metadata keys and exact scope/action matrix. Avoid re-rendering focused search controls. Setup stages are presentation only; revision/digest invalidation and backend checks remain authoritative. Navigation must not clear a selected assessment or infer completion. Stop bypasses busy reads/mutations. Generations/abort controllers continue to reject stale reads. No automatic mutation retries, credential storage, untrusted HTML, raw evidence polling or hardcoded result/allowance fallback.

## Verification boundary

Use source checks and sequential disposable loopback browser campaigns. Their real native Mecatl/broker/validator observations do not qualify Kubernetes, Cilium, gVisor, a patch deployment or production readiness. Screenshots under ignored `artifacts/` contain only disposable synthetic fixtures. Existing Kind history is outside this refactor.
