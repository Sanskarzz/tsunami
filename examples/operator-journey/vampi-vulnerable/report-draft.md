# Tsunami engagement draft

Web/API authorization assessment

Engagement: `f57d946ec7817fdbff8900935718f273`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-2da4e093fe5ca7e5d6a4f72b555e4114

Cross-user book secret access

Status: needs_human_review; severity: unrated.

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-07e484eba4bf8aee777899b6557565e0a8d13536e72949d548364dae9af7b0cb, E-532b9a9ace2edbced98c2f3514cfb1321ac51d5519ea25e404df92c2d0c151f7, E-3a8621752578c50f22c330f1ada432855f88309859d377976119a9994af4c28e, E-7fa41e3c2dd730e9e2c37c1cb9e94da2aaa9a4535cb9f06bca74d1ad0f0f9c5e, E-e3f72d7b3ccf988f0c2b3fb5cfa270fff0fc769c68a3b126ea8c4a775437fa7e, E-0bd0e47f1aa1be7e910fd7f3578b137212cd5e880a907819fdc119a5024cbfc0, E-26dcfac5e834526b8fb4f5cea01209f26d28d16eaa0cf38994ae338bc8ded44f

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-27287a594bf2b6338f49f2a4b5f6a26ff4ef5a2c42e17dfc53429279916701ce | 3ac07c43f1cce1a46659613d90568398 | evidence/27287a594bf2b6338f49f2a4b5f6a26ff4ef5a2c42e17dfc53429279916701ce.json |
| E-07e484eba4bf8aee777899b6557565e0a8d13536e72949d548364dae9af7b0cb | 3ac07c43f1cce1a46659613d90568398 | evidence/07e484eba4bf8aee777899b6557565e0a8d13536e72949d548364dae9af7b0cb.json |
| E-532b9a9ace2edbced98c2f3514cfb1321ac51d5519ea25e404df92c2d0c151f7 | 3ac07c43f1cce1a46659613d90568398 | evidence/532b9a9ace2edbced98c2f3514cfb1321ac51d5519ea25e404df92c2d0c151f7.json |
| E-3a8621752578c50f22c330f1ada432855f88309859d377976119a9994af4c28e | 3ac07c43f1cce1a46659613d90568398 | evidence/3a8621752578c50f22c330f1ada432855f88309859d377976119a9994af4c28e.json |
| E-7fa41e3c2dd730e9e2c37c1cb9e94da2aaa9a4535cb9f06bca74d1ad0f0f9c5e | 3ac07c43f1cce1a46659613d90568398 | evidence/7fa41e3c2dd730e9e2c37c1cb9e94da2aaa9a4535cb9f06bca74d1ad0f0f9c5e.json |
| E-e3f72d7b3ccf988f0c2b3fb5cfa270fff0fc769c68a3b126ea8c4a775437fa7e | 3ac07c43f1cce1a46659613d90568398 | evidence/e3f72d7b3ccf988f0c2b3fb5cfa270fff0fc769c68a3b126ea8c4a775437fa7e.json |
| E-0bd0e47f1aa1be7e910fd7f3578b137212cd5e880a907819fdc119a5024cbfc0 | 3ac07c43f1cce1a46659613d90568398 | evidence/0bd0e47f1aa1be7e910fd7f3578b137212cd5e880a907819fdc119a5024cbfc0.json |
| E-26dcfac5e834526b8fb4f5cea01209f26d28d16eaa0cf38994ae338bc8ded44f | 3ac07c43f1cce1a46659613d90568398 | evidence/26dcfac5e834526b8fb4f5cea01209f26d28d16eaa0cf38994ae338bc8ded44f.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
