// Local-only, same-origin operator client. No cookies, browser storage, target
// login, retries of mutations, or authority beyond the existing operator API.
import {createOnboardingUI} from './onboarding.js';
const $ = id => document.getElementById(id);
const state = {token: '', epoch: 0, generation: 0, apps: [], view: null, app: null, run: '', status: null,
  bundle: null, busy: false, stopping: false, polling: false, timer: null, polls: 0, trace: [], selectedTrace: null,
  uncertainStarts: new Set(), uncertainDraft: false};
const idPattern = /^[a-f0-9]{32}$/;
const pages = {
  setup: 'POST /v1/onboarding saves a setup. Its answers, protected access, DNS permission and four-read validation are separate bounded actions. Prepare creates only an unapproved assessment draft.',
  applications: 'GET /v1/applications and GET /v1/engagements list configured plans and saved assessments. Selecting an application makes no target request.',
  plan: 'POST /v1/engagements saves a draft. GET retrieves its immutable scope. POST …/approve records approval; no agent runs yet. Digest entry is a client review guard.',
  run: 'POST …/runs creates one attempt. GET /v1/runs/{id}/status observes permission, execution, activity and cleanup. POST …/stop revokes the engagement.',
  evidence: 'GET …/export retrieves an explicit evidence snapshot and draft report. The browser does not invent findings, severity, or a human review.'
};

function element(tag, text, className) {
  const node = document.createElement(tag);
  if (text !== undefined) node.textContent = String(text);
  if (className) node.className = className;
  return node;
}
function message(text, error = false) {
  $('message').textContent = text;
  $('message').className = 'message' + (error ? ' error' : '');
  $('message').hidden = !text;
}
function page(name) {
  for (const panel of document.querySelectorAll('[data-panel]')) panel.hidden = panel.dataset.panel !== name;
  for (const button of document.querySelectorAll('[data-page]')) button.classList.toggle('selected', button.dataset.page === name);
  $('step-explanation').textContent = pages[name];
}
function pretty(value) { return JSON.stringify(value, null, 2); }
function time(value) { return value ? new Date(value).toLocaleString() : 'Not recorded'; }
function details(node, pairs) {
  node.replaceChildren();
  for (const [key, value] of pairs) node.append(element('dt', key), element('dd', value ?? 'Not recorded'));
}
function showTrace(entry) {
  state.selectedTrace = entry;
  $('api-detail').textContent = pretty({request: {method: entry.method, path: entry.path, body: entry.body, authorization: '[omitted]'},
    response: {status: entry.status, body: entry.response}, at: entry.at, duration_ms: entry.duration}).slice(0, 18000);
}
function trace(entry) {
  state.trace.unshift(entry);
  state.trace.length = Math.min(state.trace.length, 60);
  $('api-log').replaceChildren();
  for (const call of state.trace) {
    const button = element('button', undefined, 'api-entry' + (call.status >= 300 || !call.status ? ' failed' : ''));
    button.type = 'button';
    button.append(element('span', call.method + ' · ' + (call.status || 'not confirmed'), 'api-method'), element('span', call.path, 'api-path'), element('small', time(call.at) + ' · ' + call.duration + ' ms'));
    button.addEventListener('click', () => showTrace(call));
    $('api-log').append(button);
  }
  if (!state.selectedTrace || entry.method === 'POST') showTrace(entry);
}
async function api(method, path, body) {
  if (!state.token) throw new Error('Connect with your operator token first.');
  if (!/^\/v1\/(applications|status|onboarding(?:\/[a-f0-9]{32}(?:\/(answers|resolve|access|details|validate|prepare|stop))?)?|engagements(?:\/[a-f0-9]{32}(?:\/(approve|runs|stop|export))?)?|runs\/[a-f0-9]{32}\/status)$/.test(path)) throw new Error('Unsupported operator API path.');
  const epoch = state.epoch, started = performance.now();
  const entry = {method, path, body: path.startsWith('/v1/onboarding') ? '[setup input omitted; includes protected fields]' : body, at: new Date().toISOString(), status: 0, response: 'No response confirmed.'};
  const controller = new AbortController();
  const timeout = setTimeout(() => controller.abort(), 35000);
  try {
    const response = await fetch(path, {method, credentials: 'omit', cache: 'no-store', redirect: 'error', signal: controller.signal,
      headers: {Accept: 'application/json', Authorization: 'Bearer ' + state.token, ...(body === undefined ? {} : {'Content-Type': 'application/json'})},
      body: body === undefined ? undefined : JSON.stringify(body)});
    entry.status = response.status;
    // Match the existing inline lab export ceiling. Large exports can still
    // consume substantial browser memory; this is not scalable object storage.
    const limit = path.endsWith('/export') ? 192 * 1024 * 1024 : 2 * 1024 * 1024;
    const reader = response.body.getReader(), chunks = [];
    let size = 0;
    for (;;) {
      const {done, value} = await reader.read();
      if (done) break;
      size += value.length;
      if (size > limit) { await reader.cancel(); throw new Error('Response exceeds the local interface limit. Use the CLI and inspect the server contract.'); }
      chunks.push(value);
    }
    const bytes = new Uint8Array(size);
    let offset = 0;
    for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.length; }
    const data = JSON.parse(new TextDecoder().decode(bytes));
    entry.response = path.endsWith('/export') && response.ok ? {version: data.version, synthetic_only: data.synthetic_only,
      candidates: data.finding_candidates?.length, reviewed_findings: data.toolkit_findings?.length, artifacts: data.artifacts?.length,
      note: 'Full raw evidence is shown only in the evidence panel/download.'} : data;
    if (!response.ok) throw new Error('API ' + response.status + ': ' + (data.error || 'Request refused.'));
    if (epoch !== state.epoch) throw new Error('Connection changed; inspect saved state before retrying work.');
    return data;
  } catch (error) {
    if (error.name === 'AbortError') throw new Error('API response timed out. A mutation may already be committed; inspect saved assessments before retrying.');
    throw error;
  } finally {
    clearTimeout(timeout);
    entry.duration = Math.round(performance.now() - started);
    if (epoch === state.epoch) trace(entry);
  }
}
function stopObservation() {
  clearTimeout(state.timer); state.timer = null; state.polling = false;
  $('pause').textContent = 'Resume observation';
}
function resetEvidence() {
  state.bundle = null;
  $('candidates').replaceChildren();
  $('captures').textContent = 'Load a snapshot to inspect captures.';
  $('capture-detail').hidden = true; $('capture-detail').textContent = '';
  $('evidence-summary').textContent = 'No evidence snapshot loaded.';
  $('draft-report').textContent = 'No report loaded.';
}
function controls() {
  const connected = !!state.token, e = state.view?.engagement, locked = state.busy || state.stopping;
  $('connect').disabled = locked; $('disconnect').disabled = !connected || locked;
  $('reload').disabled = !connected || locked;
  $('create-draft').disabled = !connected || !state.app || locked || state.uncertainDraft;
  const current = e && new Date(e.expires_at).getTime() > Date.now();
  $('approve').disabled = !connected || locked || !current || e.state !== 'draft' || !$('scope-reviewed').checked || $('reviewed-digest').value.trim() !== e.scope_digest;
  $('go-run').disabled = !e;
  $('start').disabled = !connected || locked || !current || e.state !== 'active' || e.approved_digest !== e.scope_digest || !!e.run_ids?.length || state.uncertainStarts.has(e.id);
  // Stop stays independent of observation/export/start requests.
  $('stop').disabled = !connected || !e || state.stopping;
  $('refresh-run').disabled = !connected || !e || locked;
  $('pause').disabled = !connected || !state.run;
  $('go-evidence').disabled = !e;
  $('load-evidence').disabled = !connected || !e || locked;
  $('download-bundle').disabled = !state.bundle;
  $('download-report').disabled = !state.bundle;
  $('run-picker').disabled = locked;
  for (const b of document.querySelectorAll('.choose-app,.open-engagement')) b.disabled = locked;
  setupUI.lock(connected,locked);
}
function selectedApplication() {
  const app = state.app;
  $('selected-application').replaceChildren();
  $('draft-form').hidden = !app || !!state.view;
  if (!app) { $('selected-application').textContent = 'Choose a configured application first.'; return; }
  $('selected-application').append(element('p', 'REGISTERED APPLICATION', 'eyebrow'), element('h2', app.id), element('p', app.title));
  const info = element('dl', undefined, 'details');
  details(info, [['Origin', app.origin], ['Profile', app.version], ['Destination addresses', app.approved_addresses.join('\n')], ['Expected access', app.expected_access], ['Test identities', app.identity.map(r => r.account).join(', ')]]);
  $('selected-application').append(info);
}
function renderPlan() {
  const e = state.view?.engagement;
  $('plan-card').hidden = !e;
  if (!e) {
    details($('run-context'), [['Assessment', 'No assessment loaded.']]);
    $('events').textContent = 'Load an assessment to read its stored events.';
    $('run-picker-wrap').hidden = true;
    controls(); return;
  }
  $('engagement-state').textContent = e.state;
  details($('scope-details'), [['Assessment', e.id], ['Application', e.application.id], ['Origin', e.target], ['Allowed addresses', e.application.approved_addresses.join('\n')],
    ['Authorization ref', e.authorization_ref], ['Executor', e.executor || 'native'], ['Execution', 'Scripted Mecatl; no external model calls'],
    ['Limits', '8 target reservations/run · 90-second run deadline · ' + e.max_runs + ' child runs maximum · ' + e.max_concurrent + ' concurrent maximum'],
    ['Per-run rate', e.application.min_interval_ms ? 'At least ' + e.application.min_interval_ms + ' ms between target reservations; no cross-run aggregate rate guarantee' : 'Existing bounded lab profile; no configured request interval'],
    ['Engagement expiry', time(e.expires_at)], ['Scope digest', e.scope_digest], ['Expected access', e.application.expected_access]]);
  const p = e.application;
  $('scope-json').textContent = pretty(e);
  const reads = [...p.identity, ...p.baselines, p.probe, p.baselines[0], p.identity[1]];
  const labels = ['Verify owner identity', 'Verify peer identity', 'Establish owner baseline', 'Establish peer baseline',
    p.version === 'http-property-access/v1' ? 'Check shared response field filtering' : 'Check cross-user object access', 'Recheck owner access', 'Recheck peer identity'];
  $('read-sequence').replaceChildren();
  for (const [i, r] of reads.entries()) {
    const li = element('li'), text = element('span'); text.append(element('code', 'GET ' + r.path), element('small', ' · ' + labels[i]));
    li.append(element('strong', r.account), text); $('read-sequence').append(li);
  }
  $('approval-form').hidden = e.state !== 'draft';
  const picker = $('run-picker'); picker.replaceChildren();
  for (const id of e.run_ids || []) { const option = element('option', id); option.value = id; picker.append(option); }
  if (state.run) picker.value = state.run;
  $('run-picker-wrap').hidden = !(e.run_ids?.length > 1);
  details($('run-context'), [['Assessment', e.id], ['Application', e.application.id], ['Run', state.run || 'Not started'], ['Engagement state', e.state], ['Scope expiry', time(e.expires_at)]]);
  $('events').replaceChildren();
  for (const ev of [...(e.events || [])].reverse().slice(0, 15)) {
    const row = element('div', undefined, 'event');
    row.append(element('time', time(ev.at)), element('strong', ev.kind || ev.type || 'Recorded event'), element('div', ev.detail || ev.message || ''));
    $('events').append(row);
  }
  if (!e.events?.length) $('events').textContent = 'No engagement events recorded.';
  controls();
}
function renderStatus() {
  const s = state.status;
  if (!s) $('observed-at').textContent = 'No status response received. Values shown later are observations, not continuous process proof.';
  for (const key of ['permission', 'execution', 'activity', 'cleanup']) $('status-' + key).textContent = s?.[key] || 'Not observed';
  $('verdict').textContent = s?.assessment?.verdict || 'Not assessed';
  $('verdict').className = 'badge' + (s?.assessment?.verdict === 'confirmed' ? ' attention' : '');
  $('result-reason').textContent = s?.assessment?.reason || (s?.permission === 'revoked' ? 'Permission was revoked. Stop is not a successful assessment; inspect any incomplete evidence.' : 'The trusted validator decides from broker-captured evidence. No supported result has been observed here.');
  const v = s?.assessment?.validation;
  $('validation-counts').textContent = v ? `${v.baselines_verified}/${v.baselines_required} baselines/rechecks verified · probe captured: ${v.probe_captured} · target reservations: ${s.target_requests_reserved}` : (s ? 'Target reservations: ' + s.target_requests_reserved + ' · uncertain operations: ' + s.uncertain_operations : '');
}
async function lists() {
  const epoch = state.epoch;
  const apps = await api('GET', '/v1/applications');
  const engagements = await api('GET', '/v1/engagements');
  const setups = await api('GET', '/v1/onboarding');
  if (epoch !== state.epoch) return;
  state.apps = apps.sort((a, b) => a.id.localeCompare(b.id));
  setupUI.lists(setups,state.apps);
  $('application-list').replaceChildren();
  for (const app of state.apps) {
    const card = element('article', undefined, 'card app-card');
    card.append(element('span', '↗', 'app-icon'), element('h2', app.id), element('span', app.version, 'badge'), element('code', app.origin));
    const b = element('button', 'Review this application →', 'button choose-app'); b.type = 'button'; b.dataset.application = app.id;
    b.addEventListener('click', () => choose(app)); card.append(b); $('application-list').append(card);
  }
  if (!apps.length) $('application-list').append(element('div', 'No applications are registered for this operator. Trusted configuration is required.', 'card empty'));
  $('engagement-list').replaceChildren();
  for (const e of engagements.sort((a, b) => b.created_at.localeCompare(a.created_at))) {
    const button = element('button', undefined, 'history-row open-engagement'); button.type = 'button'; button.dataset.engagement = e.id;
    const text = element('span', e.name); text.append(element('small', (e.application?.id || 'Legacy lab') + ' · ' + e.id));
    button.append(text, element('span', e.state, 'badge'));
    button.addEventListener('click', () => act(() => openEngagement(e.id))); $('engagement-list').append(button);
  }
  if (!engagements.length) $('engagement-list').textContent = 'No saved assessments for this operator.';
  state.uncertainDraft = false;
  controls();
}
function choose(app) {
  if (state.busy || state.stopping) return;
  stopObservation(); state.generation++; state.view = null; state.app = app; state.run = ''; state.status = null;
  $('scope-reviewed').checked = false; $('reviewed-digest').value = '';
  resetEvidence(); selectedApplication(); renderPlan(); renderStatus(); page('plan'); message('');
}
async function openEngagement(id) {
  if (!idPattern.test(id)) throw new Error('Invalid engagement ID.');
  stopObservation(); const generation = ++state.generation;
  const view = await api('GET', '/v1/engagements/' + id);
  if (generation !== state.generation) return;
  if (!view.engagement.application) throw new Error('This interface supports configured application engagements. Use the CLI for this legacy lab record.');
  state.view = view; state.app = view.engagement.application; state.run = view.engagement.run_ids?.at(-1) || ''; state.status = null;
  resetEvidence(); selectedApplication(); renderPlan(); renderStatus();
  $('scope-reviewed').checked = false; $('reviewed-digest').value = '';
  page(state.run ? 'run' : 'plan');
  if (state.run) await refresh();
  controls();
}
async function refresh() {
  const id = state.view?.engagement.id, generation = state.generation;
  if (!id) return;
  const view = await api('GET', '/v1/engagements/' + id);
  if (generation !== state.generation || state.view?.engagement.id !== id) return;
  state.view = view;
  if (!state.run && view.engagement.run_ids?.length) state.run = view.engagement.run_ids.at(-1);
  const selectedRun = state.run;
  if (selectedRun) {
    const status = await api('GET', '/v1/runs/' + selectedRun + '/status');
    if (generation !== state.generation || selectedRun !== state.run) return;
    state.status = status;
    $('observed-at').textContent = 'Last successful status response: ' + new Date().toLocaleTimeString() + '. Container observations have their own timestamps; API object absence alone is not process-exit proof.';
  }
  renderPlan(); renderStatus();
}
async function poll() {
  if (!state.polling) return;
  const generation = state.generation;
  try {
    await refresh();
    if (generation !== state.generation || !state.polling) return;
    if (state.status?.cleanup === 'api_resources_absent' || ++state.polls >= 120) {
      stopObservation();
      if (state.polls >= 120) message('Observation window ended. This did not Stop the run; inspect state or explicitly Stop.');
      return;
    }
    state.timer = setTimeout(poll, 1500);
  } catch (error) { stopObservation(); message('Observation failed; last displayed values may be stale. Stop is a separate action. ' + error.message, true); }
}
function observe() { stopObservation(); state.polls = 0; state.polling = true; $('pause').textContent = 'Pause observation'; void poll(); }
async function act(fn) {
  if (state.busy || state.stopping) return;
  state.busy = true; controls(); message('');
  try { await fn(); } catch (error) { message(error.message, true); }
  finally { state.busy = false; controls(); }
}
function download(name, data, type) {
  const url = URL.createObjectURL(new Blob([data], {type}));
  const link = element('a'); link.href = url; link.download = name; document.body.append(link); link.click(); link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
function renderEvidence(bundle) {
  const candidates = bundle.finding_candidates || [], artifacts = bundle.artifacts || [];
  $('evidence-summary').textContent = `Snapshot: ${time(new Date().toISOString())} · ${candidates.length} candidates · ${bundle.toolkit_findings?.length || 0} reviewed findings · ${artifacts.length} captures. Scope: selected procedures only.`;
  $('candidates').replaceChildren();
  for (const c of candidates) {
    const card = element('article', undefined, 'card candidate'); card.dataset.candidate = c.id;
    card.append(element('span', c.status + (c.review ? ' · review recorded; see bundle' : ' · severity unrated'), 'badge attention'), element('h2', c.title), element('p', c.reason), element('p', 'Remediation: ' + c.remediation), element('code', 'Evidence: ' + c.evidence_ids.join(', ')));
    $('candidates').append(card);
  }
  if (!candidates.length) $('candidates').append(element('div', 'No supported finding candidates in this snapshot. Check procedure progress and inconclusive/stopped results; this does not establish that the application is secure.', 'card section-card'));
  $('captures').replaceChildren();
  for (const artifact of artifacts) {
    // Decode for display only. Downloaded hashes are checked by the existing
    // export verifier, not authenticated by this viewer.
    let decoded;
    try { decoded = JSON.parse(new TextDecoder().decode(Uint8Array.from(atob(artifact.data_base64), c => c.charCodeAt(0)))); }
    catch { throw new Error('Malformed evidence artifact; retain the bundle and inspect it with the verifier.'); }
    const capture = decoded.capture;
    const button = element('button', undefined, 'capture-row' + (capture.blocked ? ' blocked' : '')); button.type = 'button';
    button.dataset.capture = capture.id;
    const text = element('span', (capture.blocked ? 'BLOCKED intent' : 'HTTP ' + (capture.status || 'error')) + ' · ' + capture.account);
    text.append(element('small', capture.url), element('small', 'SHA-256 ' + artifact.sha256));
    button.append(text);
    button.addEventListener('click', () => { $('capture-detail').hidden = false; $('capture-detail').textContent = pretty(decoded); });
    $('captures').append(button);
  }
  if (!artifacts.length) $('captures').textContent = 'No captures in this snapshot. Missing evidence is not a clean result.';
  $('draft-report').textContent = bundle.report_markdown || 'No report included.';
  controls();
}

const setupUI=createOnboardingUI({api,act,message,page,openEngagement,element,details,time,pretty});
for (const b of document.querySelectorAll('[data-page]')) b.addEventListener('click', () => page(b.dataset.page));
$('connect-form').addEventListener('submit', event => { event.preventDefault(); void act(async () => {
  const token = $('operator-token').value.trim(); $('operator-token').value = '';
  if (token.length < 16 || /\s/.test(token)) throw new Error('Enter a valid tenant operator token, not application credentials.');
  state.token = token; state.epoch++;
  try {
    await lists();
    const status = await api('GET', '/v1/status');
    $('route-note').textContent = 'Startup route configuration: ' + Object.entries(status.execution_routes || {}).map(([name, value]) => name + '=' + value.state).join(' · ') + '. Not a live dependency probe.';
    $('connect-panel').hidden = true; $('connection-state').textContent = 'Operator connected'; $('connection-state').className = 'badge good';
    message('Connected. Select a configured application or recover a saved assessment. No target requests have been made by connecting.');
  } catch (error) { state.token = ''; throw error; }
}); });
function disconnect() {
  stopObservation(); state.epoch++; state.generation++; state.token = ''; state.view = null; state.app = null; state.run = ''; state.status = null;
  state.apps = []; state.trace = []; state.selectedTrace = null; state.busy = false; state.stopping = false;
  setupUI.reset();
  $('application-list').replaceChildren(element('div', 'Disconnected. Reconnect to read saved state.', 'card empty'));
  $('engagement-list').textContent = 'Not connected.'; $('api-log').replaceChildren(); $('api-detail').textContent = 'Disconnected.';
  $('connect-panel').hidden = false; $('connection-state').textContent = 'Disconnected'; $('connection-state').className = 'badge';
  $('route-note').textContent = ''; $('operator-token').value = '';
  resetEvidence(); selectedApplication(); renderPlan(); renderStatus(); page('applications');
  message('Disconnected from the viewer. This did not Stop any assessment. Reconnect to inspect or revoke work.');
}
$('disconnect').addEventListener('click', disconnect);
$('reload').addEventListener('click', () => void act(lists));
$('draft-form').addEventListener('submit', event => { event.preventDefault(); void act(async () => {
  if (!state.app || state.uncertainDraft) throw new Error('Select an application and inspect saved assessments first.');
  let created;
  try { created = await api('POST', '/v1/engagements', {application: state.app.id, name: $('assessment-name').value.trim(), authorization_ref: $('authorization-ref').value.trim(), executor: $('executor').value}); }
  catch (error) { state.uncertainDraft = true; throw new Error('Draft creation not confirmed. Refresh saved assessments before creating another. ' + error.message); }
  await openEngagement(created.id);
  message('Draft saved. No target requests. Review the exact scope before approval.');
}); });
for (const id of ['reviewed-digest', 'scope-reviewed']) $(id).addEventListener('input', controls);
$('approval-form').addEventListener('submit', event => { event.preventDefault(); void act(async () => {
  const e = state.view.engagement;
  if (e.state !== 'draft' || !$('scope-reviewed').checked || $('reviewed-digest').value.trim() !== e.scope_digest) throw new Error('Review the saved draft and enter its exact digest.');
  state.view = await api('POST', '/v1/engagements/' + e.id + '/approve', {});
  renderPlan(); message('Approval recorded. Start is a separate action; no agent was launched by approval.');
}); });
$('go-run').addEventListener('click', () => page('run'));
$('start').addEventListener('click', () => void act(async () => {
  const e = state.view.engagement;
  if (e.state !== 'active' || e.approved_digest !== e.scope_digest || e.run_ids?.length || state.uncertainStarts.has(e.id)) throw new Error('Start requires an approved plan without an existing or uncertain attempt in this viewer.');
  state.uncertainStarts.add(e.id);
  let run;
  try { run = await api('POST', '/v1/engagements/' + e.id + '/runs', {assessment: e.items[0].procedure, execution: 'scripted'}); }
  catch (error) { throw new Error('Start not confirmed; no retry was made. Refresh the assessment to recover child IDs. ' + error.message); }
  state.uncertainStarts.delete(e.id); state.run = run.id; resetEvidence();
  await refresh(); observe();
  message('Run recorded. Reconciliation creates/observes its Kubernetes Job. The browser does not send target requests.');
}));
$('stop').addEventListener('click', async () => {
  const id = state.view?.engagement.id;
  if (!id || state.stopping) return;
  state.stopping = true; controls();
  try {
    await api('POST', '/v1/engagements/' + id + '/stop', {});
    message('Stop acknowledged: engagement permission revoked. Request drain, worker execution and cleanup still require separate observations. Target-side effects cannot be undone.');
    try { await refresh(); if (state.run) observe(); }
    catch { message('Stop acknowledged, but follow-up observation failed. Do not assume drain or cleanup; refresh when the API returns.', true); }
  } catch (error) { message('Stop was NOT acknowledged. Do not assume permission was revoked or work ended. ' + error.message, true); }
  finally { state.stopping = false; controls(); }
});
$('refresh-run').addEventListener('click', () => void act(refresh));
$('pause').addEventListener('click', () => {
  if (state.polling) { stopObservation(); message('Observation paused. This did not Stop the run.'); } else observe();
});
$('run-picker').addEventListener('change', () => void act(async () => {
  stopObservation(); state.generation++; state.run = $('run-picker').value; state.status = null; renderStatus(); await refresh();
}));
$('go-evidence').addEventListener('click', () => page('evidence'));
$('load-evidence').addEventListener('click', () => void act(async () => {
  const bundle = await api('GET', '/v1/engagements/' + state.view.engagement.id + '/export');
  state.bundle = bundle; renderEvidence(bundle);
}));
$('download-bundle').addEventListener('click', () => { if (state.bundle) download('tsunami-' + state.view.engagement.id + '-bundle.json', pretty(state.bundle), 'application/json'); });
$('download-report').addEventListener('click', () => { if (state.bundle) download('tsunami-' + state.view.engagement.id + '-draft.md', state.bundle.report_markdown, 'text/markdown'); });
window.addEventListener('pagehide', disconnect);
controls();
