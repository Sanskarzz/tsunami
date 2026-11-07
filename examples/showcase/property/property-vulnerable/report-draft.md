# Tsunami engagement draft

Web/API authorization assessment

Engagement: `736406c56111342e745d5afc4484e37f`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| property-vulnerable/property-access/v1 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-e7012ec17ee8d91cd01ce5d532669561

Owner-only note exposed through a shared record

Status: needs_human_review; severity: unrated.

The peer&\#39;s permitted object read included a known owner-only property value; identities, legitimate private views and follow-up access were verified.

Remediation: Authorize each response property for the requesting principal. Use an explicit public response projection and omit the protected fields from the peer&\#39;s view.

Evidence IDs: E-ebcb533860f83f3c7107fdfbc2babeaf8455d58c7437f21e73b082a71f453851, E-bd789ef01e2031a2b3e56cf567cb403f8fc956bcdcd41ae814042083cd991e21, E-9feb60628845aca38aa913bd832597b1a9417e737f7db1e14cf7c165164a6d4a, E-f1b8f0c07d441c07ce39b46909a0e1ca3980ca6bbf93871dedda31249e6070e7, E-73b4f4e8673ae2db21d79ce7b076f207e5d6b3cad78a23ea4a95d9a459d8eb2e, E-ee678cb960f8c1b790474306efb54bb0da14167e427a1a03d4d40e59a46972e8, E-7020a327904ddcef62ed0b17c9ed6036ad98b95557a394708410efd06b70daf6

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-1c825538aa111a082c2564fbb4de891f2e0b6ab96bc81f9064de797f3f81dae2 | ec33560af96b9117eaabe4c4ffd74051 | evidence/1c825538aa111a082c2564fbb4de891f2e0b6ab96bc81f9064de797f3f81dae2.json |
| E-ebcb533860f83f3c7107fdfbc2babeaf8455d58c7437f21e73b082a71f453851 | ec33560af96b9117eaabe4c4ffd74051 | evidence/ebcb533860f83f3c7107fdfbc2babeaf8455d58c7437f21e73b082a71f453851.json |
| E-bd789ef01e2031a2b3e56cf567cb403f8fc956bcdcd41ae814042083cd991e21 | ec33560af96b9117eaabe4c4ffd74051 | evidence/bd789ef01e2031a2b3e56cf567cb403f8fc956bcdcd41ae814042083cd991e21.json |
| E-9feb60628845aca38aa913bd832597b1a9417e737f7db1e14cf7c165164a6d4a | ec33560af96b9117eaabe4c4ffd74051 | evidence/9feb60628845aca38aa913bd832597b1a9417e737f7db1e14cf7c165164a6d4a.json |
| E-f1b8f0c07d441c07ce39b46909a0e1ca3980ca6bbf93871dedda31249e6070e7 | ec33560af96b9117eaabe4c4ffd74051 | evidence/f1b8f0c07d441c07ce39b46909a0e1ca3980ca6bbf93871dedda31249e6070e7.json |
| E-73b4f4e8673ae2db21d79ce7b076f207e5d6b3cad78a23ea4a95d9a459d8eb2e | ec33560af96b9117eaabe4c4ffd74051 | evidence/73b4f4e8673ae2db21d79ce7b076f207e5d6b3cad78a23ea4a95d9a459d8eb2e.json |
| E-ee678cb960f8c1b790474306efb54bb0da14167e427a1a03d4d40e59a46972e8 | ec33560af96b9117eaabe4c4ffd74051 | evidence/ee678cb960f8c1b790474306efb54bb0da14167e427a1a03d4d40e59a46972e8.json |
| E-7020a327904ddcef62ed0b17c9ed6036ad98b95557a394708410efd06b70daf6 | ec33560af96b9117eaabe4c4ffd74051 | evidence/7020a327904ddcef62ed0b17c9ed6036ad98b95557a394708410efd06b70daf6.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
