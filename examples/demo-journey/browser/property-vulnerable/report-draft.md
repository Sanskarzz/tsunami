# Tsunami engagement draft

UI qualification: property-vulnerable

Engagement: `4c24a6e009f6a2e266d8f7b2b5359815`; state: active. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| property-vulnerable/property-access/v1 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-cb12f214879fc8a0665cd6a869971bdb

Owner-only note exposed through a shared record

Status: needs_human_review; severity: unrated.

The peer&\#39;s permitted object read included a known owner-only property value; identities, legitimate private views and follow-up access were verified.

Remediation: Authorize each response property for the requesting principal. Use an explicit public response projection and omit the protected fields from the peer&\#39;s view.

Evidence IDs: E-f296200e27ad56fb5d4d3333f552ccc6bc32e3a659f2fdfb4e26da78742647a3, E-d482eed43530eb721a0e07c43024e87102604f534a3d5666c31f659d074c5adb, E-8a8cd0c10cd14306db69be07346d0fa1d940e455dd4258221bbc047543605a23, E-5e8a3c2727fb0e34bec3af838bde474abcb13b8c2b329772737b41d2b8c28b04, E-eda10b3ede512da367e901cee3936dbdc1c46fba976cfe1b72b1694f04710299, E-806056d9c0f47582262a8b29b3474cbb0c4e5dc1b6bf8e2c64bca6228e58d072, E-88fcfc379e192917956ca2a726ea63050b0bd227b47cdc062c7125f6d8e796c1

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-92cd158f8a5b47033f7e93641475703ac4cf62214109882d068b45cee9fd9c4d | e950815f8abdffdf65535af82ec9d105 | evidence/92cd158f8a5b47033f7e93641475703ac4cf62214109882d068b45cee9fd9c4d.json |
| E-f296200e27ad56fb5d4d3333f552ccc6bc32e3a659f2fdfb4e26da78742647a3 | e950815f8abdffdf65535af82ec9d105 | evidence/f296200e27ad56fb5d4d3333f552ccc6bc32e3a659f2fdfb4e26da78742647a3.json |
| E-d482eed43530eb721a0e07c43024e87102604f534a3d5666c31f659d074c5adb | e950815f8abdffdf65535af82ec9d105 | evidence/d482eed43530eb721a0e07c43024e87102604f534a3d5666c31f659d074c5adb.json |
| E-8a8cd0c10cd14306db69be07346d0fa1d940e455dd4258221bbc047543605a23 | e950815f8abdffdf65535af82ec9d105 | evidence/8a8cd0c10cd14306db69be07346d0fa1d940e455dd4258221bbc047543605a23.json |
| E-5e8a3c2727fb0e34bec3af838bde474abcb13b8c2b329772737b41d2b8c28b04 | e950815f8abdffdf65535af82ec9d105 | evidence/5e8a3c2727fb0e34bec3af838bde474abcb13b8c2b329772737b41d2b8c28b04.json |
| E-eda10b3ede512da367e901cee3936dbdc1c46fba976cfe1b72b1694f04710299 | e950815f8abdffdf65535af82ec9d105 | evidence/eda10b3ede512da367e901cee3936dbdc1c46fba976cfe1b72b1694f04710299.json |
| E-806056d9c0f47582262a8b29b3474cbb0c4e5dc1b6bf8e2c64bca6228e58d072 | e950815f8abdffdf65535af82ec9d105 | evidence/806056d9c0f47582262a8b29b3474cbb0c4e5dc1b6bf8e2c64bca6228e58d072.json |
| E-88fcfc379e192917956ca2a726ea63050b0bd227b47cdc062c7125f6d8e796c1 | e950815f8abdffdf65535af82ec9d105 | evidence/88fcfc379e192917956ca2a726ea63050b0bd227b47cdc062c7125f6d8e796c1.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
