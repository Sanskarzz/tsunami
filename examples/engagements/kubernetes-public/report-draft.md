# Tsunami engagement draft

Owned lab authorization assessment

Engagement: `7f9682e803fa66ebdd1d41e105c03094`; state: stopped. Synthetic targets only.

Selected procedures: 5; completed: 5; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| anonymous-access/v1 | completed | confirmed |
| missing-object/v1 | completed | inconclusive |
| opaque-ownership/v1 | completed | confirmed |
| role-boundary/v1 | completed | confirmed |
| shared-access/v1 | completed | no_issue_observed |

## Finding candidates

Candidates are observations from the synthetic validator. Impact, severity and CVSS need human review.

### F-4a5f76cdb813b1d3a47f34265b6d9d00

Unauthenticated access to a known private object

Status: needs_human_review; severity: unrated.

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

Remediation: Require authentication and object authorization before returning private data.

Evidence IDs: E-066ad3f32ed85558439dda009bed1db4d39f67d74442ffa2f2a90942139470da, E-c74f6e4e993b9e04b0eac1151b4c2270280292d348ae4bd6f085fcfd53201e44, E-6d000a5345a3f8205e4bd757cc48ea98e3aed925ff24830084d55fc70335d797

### F-a3273d9dba6ffdf71961c9380c80636c

Member access to an administrator summary

Status: needs_human_review; severity: unrated.

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

Remediation: Enforce the administrator role on the server before returning the summary. A valid member session is insufficient.

Evidence IDs: E-176bd9f17aa03b350d4141c9c843e0811dd8386273b0947e751343257b59bc35, E-01d45daea8df6c27ea0fbc0a2c20f074b5b85203974095f34e9857a3c106a7c4, E-7c26239012ac7d33f7f094171093ae03ad110c90b8770271513f25a6f58c88f9

### F-b51b1d65e467a208725685992fc7e3a8

Cross-user access to an opaque object ID

Status: needs_human_review; severity: unrated.

The probe returned the exact protected fixture object despite the reviewed deny rule; legitimate baselines also succeeded.

Remediation: Check object ownership against the authenticated account on every object read. Opaque IDs alone do not authorize access.

Evidence IDs: E-11da58db90588ff74a5c464e79ba57e9f6a207430f4731fb041185c7e4fbf2cc, E-b2a59da79528af9e3847579d4b660d34a67aa9280b12aeb526c4ee614977fd80, E-b5b5ee11082e76864527a83a819c06ce65b3bf792d426f0193810778e7ddfb01

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-bd199840c9d4b34fa9a97f0122542209333e1a42f468ad1e0360c47f87e6e266 | 93a8cde6d62ff763a98a70f27c8f1cbd | evidence/bd199840c9d4b34fa9a97f0122542209333e1a42f468ad1e0360c47f87e6e266.json |
| E-066ad3f32ed85558439dda009bed1db4d39f67d74442ffa2f2a90942139470da | 93a8cde6d62ff763a98a70f27c8f1cbd | evidence/066ad3f32ed85558439dda009bed1db4d39f67d74442ffa2f2a90942139470da.json |
| E-c74f6e4e993b9e04b0eac1151b4c2270280292d348ae4bd6f085fcfd53201e44 | 93a8cde6d62ff763a98a70f27c8f1cbd | evidence/c74f6e4e993b9e04b0eac1151b4c2270280292d348ae4bd6f085fcfd53201e44.json |
| E-6d000a5345a3f8205e4bd757cc48ea98e3aed925ff24830084d55fc70335d797 | 93a8cde6d62ff763a98a70f27c8f1cbd | evidence/6d000a5345a3f8205e4bd757cc48ea98e3aed925ff24830084d55fc70335d797.json |
| E-bf6df6daa72d83e9029d0017733a7c94e952d2a9d2742ae81b0b27a6e6f66e4c | 17eda9695d957f4c1122161c3137d5f0 | evidence/bf6df6daa72d83e9029d0017733a7c94e952d2a9d2742ae81b0b27a6e6f66e4c.json |
| E-6c5b9233a6bfe1d8543e42d887ee6e6a9a8ef0180b66f8dc37ab01e9ddbedf46 | 17eda9695d957f4c1122161c3137d5f0 | evidence/6c5b9233a6bfe1d8543e42d887ee6e6a9a8ef0180b66f8dc37ab01e9ddbedf46.json |
| E-8e3e5bfd60be7d268ea414e923268b626ea869771f7e5219e59850ea9cfde97e | 17eda9695d957f4c1122161c3137d5f0 | evidence/8e3e5bfd60be7d268ea414e923268b626ea869771f7e5219e59850ea9cfde97e.json |
| E-eecc8ed9be9d1179605432c3e39ae9e30a51f8cc4f4b3b216c98cf42f9dec765 | 17eda9695d957f4c1122161c3137d5f0 | evidence/eecc8ed9be9d1179605432c3e39ae9e30a51f8cc4f4b3b216c98cf42f9dec765.json |
| E-5822f05a07daefa17c513f03696d171cba4e7063b6ce1309a7096ef3a2f6ff0d | 36be34f6d285e8899fd27476491987bb | evidence/5822f05a07daefa17c513f03696d171cba4e7063b6ce1309a7096ef3a2f6ff0d.json |
| E-11da58db90588ff74a5c464e79ba57e9f6a207430f4731fb041185c7e4fbf2cc | 36be34f6d285e8899fd27476491987bb | evidence/11da58db90588ff74a5c464e79ba57e9f6a207430f4731fb041185c7e4fbf2cc.json |
| E-b2a59da79528af9e3847579d4b660d34a67aa9280b12aeb526c4ee614977fd80 | 36be34f6d285e8899fd27476491987bb | evidence/b2a59da79528af9e3847579d4b660d34a67aa9280b12aeb526c4ee614977fd80.json |
| E-b5b5ee11082e76864527a83a819c06ce65b3bf792d426f0193810778e7ddfb01 | 36be34f6d285e8899fd27476491987bb | evidence/b5b5ee11082e76864527a83a819c06ce65b3bf792d426f0193810778e7ddfb01.json |
| E-fc64d02f5ddce285f4960dfdaf515890b59497db6715e7e7c43fc2f3fd6776be | 9c19738f6eb9eb9497d0b7792ec4291b | evidence/fc64d02f5ddce285f4960dfdaf515890b59497db6715e7e7c43fc2f3fd6776be.json |
| E-176bd9f17aa03b350d4141c9c843e0811dd8386273b0947e751343257b59bc35 | 9c19738f6eb9eb9497d0b7792ec4291b | evidence/176bd9f17aa03b350d4141c9c843e0811dd8386273b0947e751343257b59bc35.json |
| E-01d45daea8df6c27ea0fbc0a2c20f074b5b85203974095f34e9857a3c106a7c4 | 9c19738f6eb9eb9497d0b7792ec4291b | evidence/01d45daea8df6c27ea0fbc0a2c20f074b5b85203974095f34e9857a3c106a7c4.json |
| E-7c26239012ac7d33f7f094171093ae03ad110c90b8770271513f25a6f58c88f9 | 9c19738f6eb9eb9497d0b7792ec4291b | evidence/7c26239012ac7d33f7f094171093ae03ad110c90b8770271513f25a6f58c88f9.json |
| E-51051959f72c7f5392fbd2c1679e08f2721574394492a92d9a93d8d181a803da | 1a8963118796a4cb417431476ced4dbe | evidence/51051959f72c7f5392fbd2c1679e08f2721574394492a92d9a93d8d181a803da.json |
| E-43e3b6924aedebbaf0c09a1fe86fc9996355224cf2fda114acf5e4bd3ee23749 | 1a8963118796a4cb417431476ced4dbe | evidence/43e3b6924aedebbaf0c09a1fe86fc9996355224cf2fda114acf5e4bd3ee23749.json |
| E-7295aa6a9e1e789ac704d2caa7815c80f85029e3cd5b7b890d470da01a8bc67f | 1a8963118796a4cb417431476ced4dbe | evidence/7295aa6a9e1e789ac704d2caa7815c80f85029e3cd5b7b890d470da01a8bc67f.json |
| E-6ca15d4df842a49bea6f005528f322545da664c7013b7192de7e59e3452aa709 | 1a8963118796a4cb417431476ced4dbe | evidence/6ca15d4df842a49bea6f005528f322545da664c7013b7192de7e59e3452aa709.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
