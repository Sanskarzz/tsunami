# Tsunami UI direction

A separate [running local operator UI](../local-ui.md) now connects the existing lab APIs. The 14 templates in this directory remain fictional future-product previews; they have not been turned into a customer portal.

Open [index.html](index.html) in a browser. It links all 14 product screens, and **Start the walkthrough** follows the intended journey. Everything is local HTML/CSS/JavaScript; there is no build step, dependency installation or external asset service.

These are design templates, not an implemented frontend. They make no API calls, start no agents, contact no targets and perform no authentication. All application names, domains, credentials references, counts, elapsed times, findings and reviews are fictional. They are separate from the repository's recorded lab results.

## Suggested walkthrough

1. [Choose a workspace](workspace.html).
2. [Browse applications](applications.html).
3. [Add an application](application-new.html), then [connect test access](application-access.html).
4. [Review the application](application-detail.html).
5. [Choose a built-in testing profile](assessment-new.html).
6. [Review scope and the proposed plan](assessment-review.html). Check the approval box to enable the preview launch button.
7. [Follow an assessment](assessment-run.html). Explore Procedures, Activity and Approved scope.
8. Press **Stop assessment**, then **Preview stopped state**. [The stopped screen](assessment-stopped.html) separates permission, request draining and cleanup.
9. [Triage candidates](findings.html), [inspect evidence and review](finding-detail.html), then [preview the report](report.html).

[Assessment history](assessments.html) and [report history](reports.html) show the main list views. The navigation and flow map let you jump between any screens.

## Local interactions

- Sidebar, cards and primary actions navigate between local files.
- Search filters the sample rows on list pages.
- Selecting private connectivity reveals a mock connector selector.
- Scope approval enables the simulated launch action.
- Assessment tabs switch local panels.
- Stop opens a confirmation dialog and links to the fixed stopped-state example.
- Review choices either open the fixed reviewed report or show a simulated disposition message.
- Print uses the browser's print dialog with a report-specific layout; exports are not connected.

Forms do not save input. Subsequent pages use the same fixed Books API example even if fields are edited. No input is stored in local/session storage or sent anywhere. Use only fictional credential references; there are no fields for real passwords/provider keys. Status does not advance automatically or poll a backend.

The running, stopped, candidate and reviewed-report pages show different illustrative moments of a journey. Their data is not a shared live state. The report's example CVSS score is static. It is not a calculator or a real reviewer decision.

## What exists in the backend versus these screens

| Preview area | Current implementation boundary |
|---|---|
| Workspace/sign-in | Demo tenant operator tokens exist. Customer login, sessions and roles are planned. |
| Application onboarding | Applications are registered through trusted startup configuration. This form and customer secret-reference onboarding are planned. |
| Profile selector and automatic plan | The current configured-app path runs one versioned object-access plan. Expanded profile selection and automatic prerequisite binding are planned. |
| Approval, run status and Stop | Corresponding backend routes exist. These pages are not connected to them; distributed failure/Stop qualification remains incomplete. |
| Evidence and review | Captures, review revisions, CVSS-vector scoring and exports exist. Individual reviewer identity and broader remediation/retest workflows are planned. The mock “needs more information” choice is not an existing review API decision. |
| Credential health / private setup | States are illustrative. General customer authentication, refresh and private configurable-app connectivity remain work. |
| Usage and capacity | Counts are fictional. Distributed budgeting and measured production capacity are not implemented. |

The three-included-test sample profile is an intended UI direction, not proof that the current VAmPI configured application executes all three procedures. Current VAmPI records exercise the configured seven-read object-access test. The synthetic catalog's other procedures are a separate existing path.

## Files and visual direction

Each screen is a readable `.html` file. Shared presentation lives in [assets/styles.css](assets/styles.css) and small local behaviors in [assets/preview.js](assets/preview.js). JavaScript also inserts common navigation, icons and the prototype banner; enable it for the complete preview. There is no frontend framework or API client.

The direction uses a dark workspace sidebar, light content area, restrained teal actions, visible approval/Stop controls and evidence-focused pages. Amber marks attention or incomplete work; it is not automatically a vulnerability severity. Layouts adapt to narrow screens. All icons are inline SVG, and the system font works offline.

The interface keeps scope, test procedure, evidence, review and coverage separate. It avoids a chat-first workflow that would require customers to write attack prompts or treat model statements as authority.

Refer to [the full design](../design.md), [the project goal](../project-goal.md), [implementation tracking](../track.md) and [actual lab results](../../examples/tool-execution/README.md) for the source of truth outside these mockups.
