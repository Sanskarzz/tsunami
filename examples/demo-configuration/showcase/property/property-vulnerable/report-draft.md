# Tsunami engagement draft

Web/API authorization assessment

Engagement: `8dc64aff4ee7267ee32368c0a35b3bf0`; state: stopped. Synthetic targets only.

Selected procedures: 1; completed: 1; skipped: 0; pending: 0; running: 0; failed: 0; stopped: 0; incomplete evidence: 0.

Completion means the selected procedure captured its prerequisites and probe. It does not mean the application is secure.

| Procedure | Progress | Verdict |
|---|---|---|
| property-vulnerable/property-access/v1 | completed | confirmed |

## Finding candidates

Candidates come from broker-captured evidence on owned lab data. Explicit review records are separate from the observation.

### F-895fd579360a1174c1d868a0c8e1bc93

Owner-only note exposed through a shared record

Status: needs_human_review; severity: unrated.

The peer&\#39;s permitted object read included a known owner-only property value; identities, legitimate private views and follow-up access were verified.

Remediation: Authorize each response property for the requesting principal. Use an explicit public response projection and omit the protected fields from the peer&\#39;s view.

Evidence IDs: E-8eee2d38df2c90dff019a6e0109281d785630c844f5bc28c9842bd045cbe8d5b, E-c1d244f3d5e95d1c0ec0b5222c20c45dedcdb271f0fca131ca6eb6bd3614cf77, E-f062c0c178f6f7bebd420bc35389bbefa7cce0d8a8c4ca2fe02bb74263939986, E-a698316751e0a3c07b9abeb84540c1b9f501e274dfb55435152984c77f4a01ad, E-c70a203cd9f50aa9ebcc2072d0b50afba584317dd21e8ae8dbcda5f7994b55f0, E-edfcb1d7677004cccb039ac15826a08d7b7b16e6cd55778c9ba74f4d0b377a2e, E-04dabf94c2bf215685dfc4582f7bbd7d6363146aaa8f49b3c2c471d638968f6a

## Evidence index

| Evidence | Run | Artifact path |
|---|---|---|
| E-f2ae716d0f60122c3136c80f68da43b3dc72837a27b9795b8967f89ee36ba0f7 | 7eb18102db9b13afa578d5220f05e835 | evidence/f2ae716d0f60122c3136c80f68da43b3dc72837a27b9795b8967f89ee36ba0f7.json |
| E-8eee2d38df2c90dff019a6e0109281d785630c844f5bc28c9842bd045cbe8d5b | 7eb18102db9b13afa578d5220f05e835 | evidence/8eee2d38df2c90dff019a6e0109281d785630c844f5bc28c9842bd045cbe8d5b.json |
| E-c1d244f3d5e95d1c0ec0b5222c20c45dedcdb271f0fca131ca6eb6bd3614cf77 | 7eb18102db9b13afa578d5220f05e835 | evidence/c1d244f3d5e95d1c0ec0b5222c20c45dedcdb271f0fca131ca6eb6bd3614cf77.json |
| E-f062c0c178f6f7bebd420bc35389bbefa7cce0d8a8c4ca2fe02bb74263939986 | 7eb18102db9b13afa578d5220f05e835 | evidence/f062c0c178f6f7bebd420bc35389bbefa7cce0d8a8c4ca2fe02bb74263939986.json |
| E-a698316751e0a3c07b9abeb84540c1b9f501e274dfb55435152984c77f4a01ad | 7eb18102db9b13afa578d5220f05e835 | evidence/a698316751e0a3c07b9abeb84540c1b9f501e274dfb55435152984c77f4a01ad.json |
| E-c70a203cd9f50aa9ebcc2072d0b50afba584317dd21e8ae8dbcda5f7994b55f0 | 7eb18102db9b13afa578d5220f05e835 | evidence/c70a203cd9f50aa9ebcc2072d0b50afba584317dd21e8ae8dbcda5f7994b55f0.json |
| E-edfcb1d7677004cccb039ac15826a08d7b7b16e6cd55778c9ba74f4d0b377a2e | 7eb18102db9b13afa578d5220f05e835 | evidence/edfcb1d7677004cccb039ac15826a08d7b7b16e6cd55778c9ba74f4d0b377a2e.json |
| E-04dabf94c2bf215685dfc4582f7bbd7d6363146aaa8f49b3c2c471d638968f6a | 7eb18102db9b13afa578d5220f05e835 | evidence/04dabf94c2bf215685dfc4582f7bbd7d6363146aaa8f49b3c2c471d638968f6a.json |

This is a draft for human review. Export hashes describe bytes, not independent evidence authenticity.
