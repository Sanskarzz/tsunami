# Tsunami engagement draft

Web/API authorization assessment

Engagement: `a23fa23d907835b382669f374ab6b226`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| property-vulnerable/property-access/v1 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-0ac63c327b029c4418e22a46ba39e517

Owner-only note exposed through a shared record

Status: needs_human_review; severity: unrated.

The peer&\#39;s permitted object read included a known owner-only property value; identities, legitimate private views and follow-up access were verified.

Remediation: Authorize each response property for the requesting principal. Use an explicit public response projection and omit the protected fields from the peer&\#39;s view.

Evidence IDs: E-8b70d96c24d89d1bd35f9ea5ae1cc6bc3b85ef87abf63d642dda39a459b14419, E-ddf92811ef5d99e7267460bb162da97a3f338eed56068179da54820d7cb7e3f0, E-f777432740aecf5b091c74e52231a879ebb5bafcee09f7f74bf897f5d5ec42ea, E-89574b994f6f315e89eb71a387fd2e723e5301002cd4835892c0dc5a019e0ffa, E-8c34993116cfbf265deee144f692f4e93d93d15a8195ca30eededd56f346f945, E-ed0ae69a0ea5dcba6824c4ad38529e085248411f419934ce59736d60864b0705, E-9cf463f9dd7e26848cb3897503a0cad21c32f31ece3d0c834ade1ef39061eb72

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-ff6a77c09c67f70954658b2867067b4474ff13c5837d0206d74084ad097d4c45 | d11958ef026f4bd36f0675df85bdedd6 | evidence/ff6a77c09c67f70954658b2867067b4474ff13c5837d0206d74084ad097d4c45.json |
| E-8b70d96c24d89d1bd35f9ea5ae1cc6bc3b85ef87abf63d642dda39a459b14419 | d11958ef026f4bd36f0675df85bdedd6 | evidence/8b70d96c24d89d1bd35f9ea5ae1cc6bc3b85ef87abf63d642dda39a459b14419.json |
| E-ddf92811ef5d99e7267460bb162da97a3f338eed56068179da54820d7cb7e3f0 | d11958ef026f4bd36f0675df85bdedd6 | evidence/ddf92811ef5d99e7267460bb162da97a3f338eed56068179da54820d7cb7e3f0.json |
| E-f777432740aecf5b091c74e52231a879ebb5bafcee09f7f74bf897f5d5ec42ea | d11958ef026f4bd36f0675df85bdedd6 | evidence/f777432740aecf5b091c74e52231a879ebb5bafcee09f7f74bf897f5d5ec42ea.json |
| E-89574b994f6f315e89eb71a387fd2e723e5301002cd4835892c0dc5a019e0ffa | d11958ef026f4bd36f0675df85bdedd6 | evidence/89574b994f6f315e89eb71a387fd2e723e5301002cd4835892c0dc5a019e0ffa.json |
| E-8c34993116cfbf265deee144f692f4e93d93d15a8195ca30eededd56f346f945 | d11958ef026f4bd36f0675df85bdedd6 | evidence/8c34993116cfbf265deee144f692f4e93d93d15a8195ca30eededd56f346f945.json |
| E-ed0ae69a0ea5dcba6824c4ad38529e085248411f419934ce59736d60864b0705 | d11958ef026f4bd36f0675df85bdedd6 | evidence/ed0ae69a0ea5dcba6824c4ad38529e085248411f419934ce59736d60864b0705.json |
| E-9cf463f9dd7e26848cb3897503a0cad21c32f31ece3d0c834ade1ef39061eb72 | d11958ef026f4bd36f0675df85bdedd6 | evidence/9cf463f9dd7e26848cb3897503a0cad21c32f31ece3d0c834ade1ef39061eb72.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
