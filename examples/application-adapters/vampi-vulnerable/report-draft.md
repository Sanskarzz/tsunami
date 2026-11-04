# Tsunami engagement draft

Owned application access-control test

Engagement: `0e729e9376c480369ebe22505083388f`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| vampi-vulnerable/object-access/v2 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-8f9c4d794bb1a5fd573c41a62a048781

Cross-user book secret access

Status: reviewed_confirmed; severity: Medium (see review provenance below).

The authenticated peer received the owner&\#39;s approved proof data; account identities and owner access were independently checked.

Remediation: Authorize every object read against the authenticated principal and intended sharing rules on the server.

Evidence IDs: E-1b89ddf2403124ba00c6b7ebdb07f33069ca6ccced2602f40c6afcc803fe52b7, E-fea32037048fd1017c4b6fd72ab98c288c7860586f0a6fed60969aedcd1febbe, E-44005e2419b5396e8591a28cbf1aa8fe18c90f39ef73cc788ab6fdca7e42951e, E-30b91ad88f41db245f24aec5cfd5899979cf62a27a866133c1a5b40e7385565a, E-b931da81f6d0ae6ad580327fd60bb920e3d954d6f3155913ec34e9478935baed, E-a10fbdb45b145646e029c6ed14c08ef0bf05f3d200f5fb82d0e894d9ac841a05, E-56a1be1a67f690ba6f2e2ee6c00be59e6728dab8b31fb4c65eb370c42decf3d9

## Reviewed findings

### F-8f9c4d794bb1a5fd573c41a62a048781: Cross-user book secret access

Severity: Medium; CVSS 3.1 base score: 6.5; vector: `CVSS:3.1/AV:N/AC:L/PR:L/UI:N/S:U/C:H/I:N/A:N`.

Review purpose: owned-lab, automated_lab_test. An automated lab review is a test, not human signoff.

Impact supplied by reviewer: An authenticated synthetic peer retrieved another synthetic test account&\#39;s seeded private object proof.

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-dcdd32282e468e332b496caefc48756b2576b0ae6abbbf794c3a7e3e1ea8931a | 6be923366136c9afdeac856c3f0537bc | evidence/dcdd32282e468e332b496caefc48756b2576b0ae6abbbf794c3a7e3e1ea8931a.json |
| E-1b89ddf2403124ba00c6b7ebdb07f33069ca6ccced2602f40c6afcc803fe52b7 | 6be923366136c9afdeac856c3f0537bc | evidence/1b89ddf2403124ba00c6b7ebdb07f33069ca6ccced2602f40c6afcc803fe52b7.json |
| E-fea32037048fd1017c4b6fd72ab98c288c7860586f0a6fed60969aedcd1febbe | 6be923366136c9afdeac856c3f0537bc | evidence/fea32037048fd1017c4b6fd72ab98c288c7860586f0a6fed60969aedcd1febbe.json |
| E-44005e2419b5396e8591a28cbf1aa8fe18c90f39ef73cc788ab6fdca7e42951e | 6be923366136c9afdeac856c3f0537bc | evidence/44005e2419b5396e8591a28cbf1aa8fe18c90f39ef73cc788ab6fdca7e42951e.json |
| E-30b91ad88f41db245f24aec5cfd5899979cf62a27a866133c1a5b40e7385565a | 6be923366136c9afdeac856c3f0537bc | evidence/30b91ad88f41db245f24aec5cfd5899979cf62a27a866133c1a5b40e7385565a.json |
| E-b931da81f6d0ae6ad580327fd60bb920e3d954d6f3155913ec34e9478935baed | 6be923366136c9afdeac856c3f0537bc | evidence/b931da81f6d0ae6ad580327fd60bb920e3d954d6f3155913ec34e9478935baed.json |
| E-a10fbdb45b145646e029c6ed14c08ef0bf05f3d200f5fb82d0e894d9ac841a05 | 6be923366136c9afdeac856c3f0537bc | evidence/a10fbdb45b145646e029c6ed14c08ef0bf05f3d200f5fb82d0e894d9ac841a05.json |
| E-56a1be1a67f690ba6f2e2ee6c00be59e6728dab8b31fb4c65eb370c42decf3d9 | 6be923366136c9afdeac856c3f0537bc | evidence/56a1be1a67f690ba6f2e2ee6c00be59e6728dab8b31fb4c65eb370c42decf3d9.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
