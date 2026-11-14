// Local-only, same-origin operator client. No cookies, browser storage, target
// login, retries of mutations, or authority beyond the existing operator API.
import {createOnboardingUI} from './onboarding.js';
const $ = id => document.getElementById(id);
const state = {token: '', epoch: 0, generation: 0, apps: [], view: null, app: null, run: '', status: null, statusStale: false,
  bundle: null, busy: false, stopping: false, polling: false, timer: null, polls: 0, trace: [], selectedTrace: null,
  uncertainStarts: new Set(), startRequests: new Map(), uncertainDraft: false, currentPage: 'applications', actionID: 0, pollVersion: 0,
  refreshJob: null, readControllers: new Set(), activity: null, stopReceipt: null, statusStaleReason: '',
  engagements: [], listsAt: null, reviewedScope: '', findings: null, findingsStale: false, findingsJob: null, rawJob: null, rawEvidence: null,
  retest: null, retestJob: null, retestStale: false};
const idPattern = /^[a-f0-9]{32}$/;
const pages = {
  setup: 'POST /v1/onboarding saves a setup. Its answers, protected access, DNS permission and four-read validation are separate bounded actions. Prepare creates only an unapproved assessment draft.',
  applications: 'GET /v1/applications and GET /v1/engagements list configured plans and saved assessments. Selecting an application makes no target request.',
  plan: 'POST /v1/engagements saves a draft. GET retrieves its immutable scope. POST …/approve requires the reviewed scope digest; no agent runs yet.',
  run: 'GET …/status and cursor-based GET …/events show actual stored observations. POST …/stop independently revokes the engagement. Admission, send, response, validation and cleanup are separate facts.',
  evidence: 'GET …/findings reads body-free candidate/check metadata. Individual raw evidence and full exports require separate explicit reads. Validation and recorded review provenance stay separate.'
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
function renderNavigation() {
  for (const name of ['plan','run','evidence']) {
    const available = name === 'plan' ? !!state.app : !!state.view;
    $(name + '-empty').hidden = available;
    $(name + '-workspace').hidden = !available;
  }
}
function page(name) {
  renderNavigation();
  if (state.currentPage === 'run' && name !== 'run') stopObservation();
  if (state.currentPage === 'evidence' && name !== 'evidence') cancelFindingReads();
  if (!['plan','evidence'].includes(name)) cancelRetestRead();
  state.currentPage = name;
  $('setup-safety').hidden = name !== 'setup' || $('setup-current').hidden;
  for (const panel of document.querySelectorAll('[data-panel]')) panel.hidden = panel.dataset.panel !== name;
  for (const button of document.querySelectorAll('[data-page]')) {
    const selected = button.dataset.page === name;
    button.classList.toggle('selected', selected);
    if (selected) button.setAttribute('aria-current', 'page'); else button.removeAttribute('aria-current');
  }
  for (const step of document.querySelectorAll('[data-journey]')) step.classList.toggle('current', step.dataset.journey === name);
  const empty = ['plan','run','evidence'].includes(name) && !$(name + '-empty').hidden;
  const heading = document.querySelector(empty ? `#${name}-empty h2` : `[data-panel="${name}"] h1`);
  if (heading) { heading.tabIndex = -1; heading.focus(); }
  $('step-explanation').textContent = pages[name];
  if (name === 'evidence' && state.token && state.view) void loadFindings().catch(() => {});
  if (['plan','evidence'].includes(name) && state.view?.engagement.source_engagement_id) void loadRetestComparison().catch(() => {});
}
function pretty(value) { return JSON.stringify(value, null, 2); }
function time(value) { return value ? new Date(value).toLocaleString() : 'Not recorded'; }
function accessText(access) {
  if (!access) return 'Availability not returned; session health unknown';
  const availability = {configured: 'Credentials configured locally', unavailable: 'Access unavailable locally', configuration_changed: 'Configuration differs from this saved scope'};
  let text = (availability[access.availability] || 'Availability unknown') + ' · current session health unknown';
  if (access.setup_permission_expires_at) text += ' · setup permission expires ' + time(access.setup_permission_expires_at);
  if (access.baselines_validated_at) text += ' · preflight last passed ' + time(access.baselines_validated_at);
  return text;
}
function executionText(mode) {
  if (mode === 'scripted') return 'Scripted Mecatl; no external model calls';
  return mode || 'Not returned by server';
}
function details(node, pairs) {
  node.replaceChildren();
  for (const [key, value] of pairs) node.append(element('dt', key), element('dd', value ?? 'Not recorded'));
}
function ruleText(rule) {
  return {deny: "The peer must not receive the owner's private object.", filtered: 'The peer may read the shared public object; known private fields must be omitted.', allow: 'The selected object and proof are intentionally shared with the peer.'}[rule] || 'Access rule not available';
}
function actionTable(node, rows) {
  const table = element('table'), head = element('thead'), header = element('tr'), body = element('tbody');
  for (const label of ['Account', 'Action', 'Purpose']) { const th = element('th', label); th.scope = 'col'; header.append(th); }
  head.append(header);
  for (const [account, path, purpose] of rows) {
    const row = element('tr'); row.append(element('td', account), element('td', 'GET ' + path, 'mono'), element('td', purpose)); body.append(row);
  }
  table.append(head, body); node.replaceChildren(table);
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
async function api(method, path, body, signal) {
  if (!state.token) throw new Error('Connect with your operator token first.');
  if (!/^\/v1\/(applications|status|onboarding(?:\/[a-f0-9]{32}(?:\/(answers|resolve|access|details|validate|prepare|stop))?)?|engagements(?:\/[a-f0-9]{32}(?:\/(approve|runs|stop|export|findings|retest|evidence\/E-[a-f0-9]{64}))?)?|runs\/[a-f0-9]{32}\/(status|events(?:\?after=\d{1,16}&limit=\d{1,3})?))$/.test(path)) throw new Error('Unsupported operator API path.');
  const epoch = state.epoch, started = performance.now();
  const entry = {method, path, body: path.startsWith('/v1/onboarding') ? '[setup input omitted; includes protected fields]' : body, at: new Date().toISOString(), status: 0, response: 'No response confirmed.'};
  const controller = new AbortController();
  const abort = () => controller.abort();
  if (signal?.aborted) abort(); else signal?.addEventListener('abort', abort, {once: true});
  if (method === 'GET') state.readControllers.add(controller);
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
    entry.response = path.includes('/evidence/') && response.ok ? {id:data.id,run_id:data.run_id,capture_id:data.capture_id,sha256:data.sha256,note:'Raw bytes omitted from API inspection; explicit capture panel only.'} : path.endsWith('/export') && response.ok ? {version: data.version, synthetic_only: data.synthetic_only,
      candidates: data.finding_candidates?.length, reviewed_findings: data.toolkit_findings?.length, artifacts: data.artifacts?.length,
      note: 'Full raw evidence is shown only in the evidence panel/download.'} : data;
    if (!response.ok) throw new Error('API ' + response.status + ': ' + (data.error || 'Request refused.'));
    if (epoch !== state.epoch) throw new Error('Connection changed; inspect saved state before retrying work.');
    return data;
  } catch (error) {
    if (error.name === 'AbortError' && (signal?.aborted || epoch !== state.epoch)) {
      const cancelled = new Error('Viewer read cancelled; no permission was changed.'); cancelled.name = 'AbortError'; throw cancelled;
    }
    if (error.name === 'AbortError') throw new Error(method === 'GET' ? 'Observation read timed out; current state is unknown.' : 'API response timed out. A mutation may already be committed; inspect saved assessments before retrying.');
    throw error;
  } finally {
    clearTimeout(timeout);
    signal?.removeEventListener('abort', abort);
    state.readControllers.delete(controller);
    entry.duration = Math.round(performance.now() - started);
    if (epoch === state.epoch) trace(entry);
  }
}
function readRunActivity(id, after = 0, limit = 50, signal) {
  if (!idPattern.test(id) || !Number.isSafeInteger(after) || after < 0 || !Number.isInteger(limit) || limit < 1 || limit > 100)
    throw new Error('Invalid activity cursor.');
  return api('GET', '/v1/runs/' + id + '/events?after=' + after + '&limit=' + limit, undefined, signal);
}
function cancelRefresh() {
  const previous = state.refreshJob; state.refreshJob = null;
  previous?.controller.abort();
}
function stopObservation() {
  state.pollVersion++;
  clearTimeout(state.timer); state.timer = null; state.polling = false;
  cancelRefresh();
  $('pause').textContent = 'Resume observation';
  renderActivity();
}
function resetEvidence() {
  cancelFindingReads(); state.findings=null; state.findingsStale=false; state.rawEvidence=null;
  state.bundle = null;
  $('candidates').replaceChildren();
  $('captures').textContent = 'Load a snapshot to inspect captures.';
  $('capture-detail').hidden = true; $('capture-detail').textContent = '';
  $('evidence-summary').textContent = 'No evidence snapshot loaded.';
  $('draft-report').textContent = 'No report loaded.';
  $('finding-detail').hidden=true; $('finding-summary').textContent='No finding metadata snapshot loaded.';
  for(const id of ['attempt-evidence','finding-context','finding-title','finding-observation','finding-remediation','evidence-comparison','evidence-checks','evidence-check-help','finding-expected','finding-observed','finding-review','finding-verdict'])$(id).replaceChildren();
  $('raw-evidence-status').textContent='No raw response loaded.';
}
function controls() {
  const connected = !!state.token, e = state.view?.engagement, locked = state.busy || state.stopping;
  $('prototype-notice').hidden = connected;
  $('connect').disabled = locked; $('disconnect').disabled = !connected;
  $('reload').disabled = !connected || locked;
  $('add-application').disabled = !connected || locked;
  $('create-draft').disabled = !connected || !state.app || locked || state.uncertainDraft;
  const current = e && new Date(e.expires_at).getTime() > Date.now();
  const permission = state.view?.metadata?.permission;
  const supported = state.view?.metadata?.procedures?.length && state.view.metadata.procedures.every(p => p.supported);
  $('approve').disabled = !connected || locked || !current || !supported || e.state !== 'draft' || permission !== 'unapproved' || state.stopReceipt?.id === e.id || !$('scope-reviewed').checked || state.reviewedScope !== e.scope_digest || $('reviewed-digest').value.trim() !== e.scope_digest;
  if (e?.source_engagement_id && (!state.retest || state.retestStale || state.retest.retest.engagement.scope_digest !== e.scope_digest)) $('approve').disabled = true;
  $('go-run').disabled = !e;
  const recovery = e && state.uncertainStarts.has(e.id) && state.startRequests.get(e.id)?.scope === e.scope_digest;
  $('start-recovery').hidden = !recovery || !!state.run;
  $('operation-state').textContent = state.busy ? 'Operator request pending · Stop remains available for selected work' : '';
  $('start').textContent = recovery ? 'Recover Start' : 'Start';
  $('start').disabled = !connected || locked || !current || !supported || e.state !== 'active' || permission !== 'active' || (!recovery && state.view?.metadata?.access?.availability !== 'configured') || e.approved_digest !== e.scope_digest || !!state.run || !!e.run_ids?.length || (state.uncertainStarts.has(e.id) && !recovery) || state.stopReceipt?.id === e.id;
  // Stop stays independent of observation/export/start requests.
  $('stop').disabled = !connected || !e || state.stopping;
  $('refresh-run').disabled = !connected || !e || locked;
  $('pause').disabled = !connected || !e;
  $('go-evidence').disabled = !e;
  $('load-evidence').disabled = !connected || !e || locked;
  $('load-findings').disabled = !connected || !e || !!state.findingsJob;
  $('download-capture').disabled = !state.rawEvidence;
  $('download-bundle').disabled = !state.bundle;
  $('download-report').disabled = !state.bundle;
  $('create-retest').disabled = !connected || !e || locked || state.uncertainDraft || !state.apps.some(app => app.id === $('retest-application').value && app.metadata?.procedure?.supported);
  $('run-picker').disabled = locked;
  for (const b of document.querySelectorAll('.choose-app,.open-engagement')) b.disabled = locked;
  setupUI.lock(connected,locked);
  renderStopReceipt();
  renderNavigation();
  $('use-reviewed-lab').disabled = !connected || locked;
}
function selectedApplication() {
  const app = state.app;
  $('selected-application').replaceChildren();
  $('selected-application').classList?.toggle('empty', !app);
  $('draft-form').hidden = !app || !!state.view;
  if (!app) { $('selected-application').textContent = 'Choose a configured application first.'; return; }
  if (state.view) {
    $('selected-application').append(element('p', 'SAVED APPLICATION SCOPE', 'eyebrow'), element('h2', app.title), element('small', app.id)); return;
  }
  $('selected-application').append(element('p', state.view ? 'SAVED APPLICATION SCOPE' : 'REGISTERED APPLICATION', 'eyebrow'), element('h2', app.id), element('p', app.title));
  const info = element('dl', undefined, 'details');
  const metadata = state.view?.metadata || app.metadata;
  details(info, [['Origin', app.origin], ['Profile', app.version], ['Destination addresses', app.approved_addresses.join('\n')], ['Expected access', app.expected_access], ['Test identities', app.identity.map(r => r.account).join(', ')],
    ['Access availability', accessText(metadata?.access)], ['Metadata observed', time(metadata?.observed_at)]]);
  $('selected-application').append(info);
}
function renderPlan() {
  const e = state.view?.engagement;
  $('plan-card').hidden = !e;
  if (!e) {
    details($('run-context'), [['Assessment', 'No assessment loaded.']]);
    $('run-title').textContent = 'No assessment selected'; $('run-identity').textContent = '';
    $('events').textContent = 'Load an assessment to read its stored events.';
    $('run-picker-wrap').hidden = true;
    controls(); return;
  }
  $('engagement-state').textContent = e.state;
  $('run-title').textContent = e.name || e.application.title;
  $('run-identity').textContent = e.application.id + ' · Run: ' + (state.run || 'Not started');
  const metadata = state.view.metadata;
  const procedure = metadata?.procedures?.find(p => p.id === e.items[0]?.procedure && p.digest === e.items[0]?.definition_digest);
  const limits = procedure?.supported && Number.isInteger(procedure.max_requests) && Number.isInteger(metadata.max_run_duration_seconds)
    ? `${procedure.max_requests} target reservations/run · up to ${metadata.max_run_duration_seconds} seconds/run, capped at scope expiry · ${e.max_runs} child runs maximum · ${e.max_concurrent} concurrent maximum`
    : 'Run allowances not available for this stored procedure';
  const displayed = $('reviewed-digest');
  if (displayed.dataset.scope !== e.scope_digest) {
    displayed.dataset.scope = e.scope_digest; displayed.value = e.scope_digest;
    $('scope-reviewed').checked = false; state.reviewedScope = '';
  }
  details($('scope-summary'), [['Application', e.application.id], ['Target', e.target], ['Test identities', e.application.identity.map(r => r.account).join(', ')], ['Approved addresses', e.application.approved_addresses.join('\n')],
    ['Intended rule', ruleText(e.application.expected_access)], ['Protected proof fields', Object.keys(e.application.baselines?.[0]?.sha256 || {}).join(', ') || 'Not returned'], ['Limits', limits], ['Minimum request interval', !procedure?.supported ? 'Unknown' : procedure.min_interval_ms ? procedure.min_interval_ms + ' ms per run' : 'No configured interval'],
    ['Permission expires', time(e.expires_at)], ['Execution', executionText(metadata?.execution_mode)], ['Scope permission', metadata?.permission || 'Unknown'], ['Access', accessText(metadata?.access)]]);
  $('scope-support').textContent = procedure?.supported ? 'Only the listed account/GET/path pairs may be requested within the allowance. Identity and owner rechecks are required by validation; the broker does not guarantee exactly-once execution or this order.' : 'This saved procedure is unsupported or its metadata is unavailable. Approval and Start are disabled; inspect the historical scope without inferring current permissions.';
  details($('scope-details'), [['Assessment', e.id], ['Application', e.application.id], ['Origin', e.target], ['Allowed addresses', e.application.approved_addresses.join('\n')],
    ['Authorization ref', e.authorization_ref], ['Executor', metadata?.executor || 'Not returned by server'], ['Execution', executionText(metadata?.execution_mode)],
    ['Procedure', procedure?.id || 'Not returned by server'], ['Procedure version', procedure?.version || 'Unknown'], ['Definition digest', procedure?.digest || 'Not returned by server'], ['Limits', limits],
    ['Per-run rate', !procedure?.supported ? 'Not available' : procedure.min_interval_ms ? 'At least ' + procedure.min_interval_ms + ' ms between target reservations; no cross-run aggregate rate guarantee' : 'No configured request interval'],
    ['Engagement expiry', time(e.expires_at)], ['Scope digest', e.scope_digest], ['Expected access', e.application.expected_access],
    ['Scope permission', metadata?.permission || 'Not returned by server'], ['Access availability', accessText(metadata?.access)], ['Metadata observed', time(metadata?.observed_at)]]);
  const p = e.application;
  $('scope-json').textContent = pretty(e);
  const reads = [...p.identity, ...p.baselines, p.probe, p.baselines[0], p.identity[1]];
  const labels = ['Verify owner identity', 'Verify peer identity', 'Establish owner baseline', 'Establish peer baseline',
    p.version === 'http-property-access/v1' ? 'Check shared response field filtering' : 'Check cross-user object access', 'Recheck owner access', 'Recheck peer identity'];
  actionTable($('scope-actions'), reads.map((r, i) => [r.account, r.path, labels[i]]));
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
  const latest = metadata?.latest_attempt;
  details($('run-context'), [['Assessment', e.id], ['Application', e.application.id], ['Run', state.run || 'Not started'], ['Approval record', e.state],
    ['Scope permission', metadata?.permission || 'Not returned by server'], ['Latest attempt', latest ? `${latest.state}${latest.verdict ? ' · ' + latest.verdict : ''}` : (metadata ? 'Not started' : 'Not returned by server')], ['Scope expiry', time(e.expires_at)]]);
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
  $('observed-at').textContent = !s ? 'No status response received. Values shown later are observations, not continuous process proof.'
    : `${state.statusStale ? 'STALE · ' + (state.statusStaleReason || 'latest read failed') + '. ' : ''}Status snapshot: ${time(s.observed_at)} · executor check: ${s.observation_code || 'Not recorded'} at ${time(s.executor_observation_attempted_at)}. A failed check is not a worker observation; API object absence is not process-exit proof.`;
  for (const key of ['permission', 'execution', 'activity', 'cleanup']) $('status-' + key).textContent = s?.[key] || 'Not observed';
  $('verdict').textContent = s?.assessment?.verdict || 'Not assessed';
  $('verdict').className = verdictClass(s?.assessment?.verdict);
  $('result-reason').textContent = s?.assessment?.reason || (s?.permission === 'revoked' ? 'Permission was revoked. Stop is not a successful assessment; inspect any incomplete evidence.' : 'The trusted validator decides from broker-captured evidence. No supported result has been observed here.');
  const v = s?.assessment?.validation;
  const allowance = Number.isInteger(s?.target_request_limit) ? s.target_request_limit : 'unknown';
  $('validation-counts').textContent = v ? `${v.baselines_verified}/${v.baselines_required} baselines/rechecks verified · probe captured: ${v.probe_captured} · target reservations: ${s.target_requests_reserved}/${allowance}` : (s ? 'Target reservations: ' + s.target_requests_reserved + '/' + allowance + ' · uncertain operations: ' + s.uncertain_operations : '');
  const remaining = Number.isInteger(s?.target_request_limit) && Number.isInteger(s?.target_requests_reserved) && s.target_request_limit >= s.target_requests_reserved && s.target_requests_reserved >= 0 ? s.target_request_limit - s.target_requests_reserved : null;
  $('budget-remaining').textContent = (state.statusStale ? 'STALE · ' : '') + (remaining === null ? 'Not observed' : `${remaining} of ${s.target_request_limit} unreserved`);
  $('budget-deadline').textContent = s ? time(s.expires_at) : 'Not observed';
  details($('run-policy'), s ? [['Attempt state', s.state || 'Not returned by server'], ['Execution mode', executionText(s.execution_mode)], ['Executor', s.executor === '' ? 'native' : (s.executor || 'Not returned by server')],
    ['Procedure', s.procedure_id || 'Not returned by server'], ['Procedure version', s.procedure_version || 'Unknown'], ['Definition digest', s.procedure_digest || 'Not recorded'],
    ['Actual run deadline', time(s.expires_at)], ['Target reservations', `${s.target_requests_reserved}/${allowance}`], ['Worker observed', time(s.worker_observed_at)], ['Exit observed', time(s.exit_observed_at)]] : [['Attempt policy', 'Not observed']]);
  details($('lifecycle-observations'), s ? [['Permission revoked', time(s.revoked_at)], ['Last reservation', time(s.last_admitted_at)],
    ['Local drain observed', time(s.local_drain_observed_at)], ['Local operations', s.local_operations ?? 'Unknown'], ['Uncertain operations', s.uncertain_operations ?? 'Unknown'],
    ['Deletion accepted', time(s.delete_accepted_at)], ['API resources absent', time(s.resources_absent_at)]] : [['Lifecycle observations', 'Not recorded']]);
}
function renderStopReceipt() {
  const e = state.view?.engagement, receipt = state.stopReceipt;
  $('assessment-safety').hidden = !state.token || !e;
  $('stop-context').textContent = e ? (e.name || 'Assessment') + ' · ' + e.id : '';
  const node = $('stop-receipt'); node.className = 'hint';
  if (!receipt || receipt.id !== e?.id) {
    node.textContent = e?.state === 'stopped' ? 'Stored assessment permission is revoked. Drain, worker exit and cleanup remain separate observations.' : 'No Stop request acknowledged in this viewer. Closing or pausing this page does not Stop testing.';
    return;
  }
  if (receipt.phase === 'pending') {
    node.textContent = 'Stop request pending. No acknowledgement received; revocation and termination are not confirmed.';
  } else if (receipt.phase === 'acknowledged') {
    node.textContent = 'Stop acknowledged at ' + time(receipt.at) + ' (viewer receipt): assessment permission revoked. Local drain, worker exit and cleanup still require separate observations.';
  } else {
    node.className = 'hint activity-warning';
    node.textContent = receipt.observedAt ? 'Stop response was NOT acknowledged. No automatic retry was made. Drain, worker exit and cleanup still require separate observations.'
      : 'Stop was NOT acknowledged. Do not assume permission was revoked or work ended. No automatic retry was made. Refresh stored state or explicitly request Stop again.';
  }
  if (receipt.observedAt) node.textContent += ' Revocation separately observed in stored assessment at ' + time(receipt.observedAt) + '.';
}
function resetActivity() {
  state.activity = {run: state.run, items: [], cursor: 0, loaded: false, at: null, stale: false, error: '',
    tracked: null, truncated: false, gap: false, displayTruncated: false, hasMore: false};
  renderActivity();
}
function activityTitle(kind) {
  return {'run.created':'Assessment attempt recorded', 'execution.dispatch_requested':'Worker dispatch requested', 'execution.worker_observed':'Running worker observed', 'execution.failed':'Worker failure recorded',
    'request.admitted':'Request admitted by broker', 'request.sent':'HTTP request write observed', 'request.capture_persisted':'Response capture persisted', 'request.rejected':'Out-of-scope intent rejected',
    'validation.completed':'Trusted validation completed', 'permission.revoked':'Permission revoked', 'activity.drained_locally':'Local broker activity drained', 'execution.exit_observed':'Worker exit observed',
    'cleanup.delete_accepted':'Kubernetes deletion accepted', 'cleanup.resources_absent':'Kubernetes resources absent'}[kind] || 'Recorded event';
}
function activityDetail(event) {
  const d = event.data;
  if (d.code === 'details_unavailable') return 'Recorded event; details were unavailable under the event byte limit.';
  switch (event.kind) {
    case 'run.created': return `Run policy recorded · ${d.execution_mode || 'mode unknown'} · ${d.executor || 'executor unknown'} · allowance ${d.request_limit ?? 'unknown'} · deadline ${time(d.deadline)}.`;
    case 'execution.dispatch_requested': return 'Dispatch intent persisted. This does not prove a Job was created or a worker started.';
    case 'execution.worker_observed': return 'The controller observed a running worker.';
    case 'execution.failed': return 'Worker failure recorded; no automatic replay. This is not a validated finding.';
    case 'request.admitted': return `Reservation ${d.reservation ?? 'unknown'} · reviewed read ${d.read_index ?? 'unknown'} · account slot ${d.account_slot ?? 'unknown'}. Admission is not target receipt.`;
    case 'request.sent': return 'Broker observed the HTTP request write. Target processing and response success are not established.';
    case 'request.rejected': return `REJECTED by broker · ${d.stage || 'stage unknown'} · ${d.code || 'reason unknown'}. No approved send is claimed for this rejected intent.`;
    case 'request.capture_persisted': return `Capture persisted · ${d.outcome || 'outcome unknown'}${d.status ? ' · HTTP ' + d.status : ''}${Number.isFinite(d.duration_ms) ? ' · ' + d.duration_ms + ' ms' : ''}. A capture alone does not confirm a finding.`;
    case 'validation.completed': return `Trusted validation: ${d.verdict || 'unknown'}${d.validation ? ' · ' + d.validation.verified + '/' + d.validation.required + ' prerequisites verified · probe captured: ' + d.validation.probe_captured : ''}.`;
    case 'permission.revoked': return 'Permission durably revoked · ' + (d.code || 'reason unknown') + '. This does not prove target cancellation or worker exit.';
    case 'activity.drained_locally': return 'Tracked broker operations drained locally. This is not target-observed cancellation or process-exit proof.';
    case 'execution.exit_observed': return 'The controller observed worker exit.';
    case 'cleanup.delete_accepted': return 'Kubernetes deletion accepted. This is not process-exit proof.';
    case 'cleanup.resources_absent': return 'Kubernetes API resources observed absent. This is not process-exit proof.';
    default: return 'Unrecognized recorded event. No additional execution or security claim is inferred.';
  }
}
function renderActivity() {
  const feed = state.activity, list = $('activity-events');
  const top = list.scrollTop || 0, follow = !list.children.length || list.scrollHeight - top - list.clientHeight < 24;
  const expanded = new Set([...list.children].filter(row => [...row.children].some(node => node.tagName === 'DETAILS' && node.open)).map(row => row.dataset.sequence));
  const focusedSequence = document.activeElement?.tagName === 'SUMMARY' ? document.activeElement.closest('[data-sequence]')?.dataset.sequence : null;
  list.replaceChildren();
  const rejections = feed?.items.filter(event => event.kind === 'request.rejected').length || 0;
  $('activity-rejections').textContent = feed?.loaded ? 'Scope rejections in displayed records: ' + rejections : 'Scope rejections: activity not loaded';
  $('show-rejection').disabled = !rejections;
  const mode = state.polling ? 'Observing' : 'Observation paused';
  $('activity-observed-at').textContent = feed?.at ? `${feed.stale ? 'STALE · latest activity read failed. ' : ''}${mode} · page received ${time(feed.at)} · cursor ${feed.cursor}${feed.hasMore ? ' · more stored pages remain' : ''}.` : `${mode} · no activity page received.`;
  const warnings = [];
  if (feed?.loaded && !feed.tracked) warnings.push('Tracking did not start at run creation. Earlier activity is unknown; no historical events were invented.');
  if (feed?.truncated) warnings.push('Server retention removed earlier events. This is a bounded journal, not a complete audit archive.');
  if (feed?.gap) warnings.push('Cursor gap observed: some events were unavailable between reads.');
  if (feed?.displayTruncated) warnings.push('Only the latest 256 received records are displayed.');
  if (feed?.error) warnings.push(feed.error);
  $('activity-notice').textContent = warnings.join(' '); $('activity-notice').hidden = !warnings.length;
  if (!feed?.items.length) {
    list.textContent = !state.run ? 'No run selected. Planned reads are not activity.' : feed?.loaded ? 'No typed activity events recorded in the returned history. Missing events are not proof that no work occurred.' : 'No activity page received for this run.';
    return;
  }
  for (const event of feed.items) {
    const row = element('li', undefined, 'event' + (event.kind === 'request.rejected' ? ' activity-rejected' : ''));
    row.dataset.sequence = event.sequence; row.dataset.kind = event.kind;
    row.append(element('time', '#' + event.sequence + ' · recorded ' + time(event.recorded_at)), element('strong', activityTitle(event.kind)), element('p', activityDetail(event)));
    const refs = [`Source: ${event.source}`, event.operation_id && 'Operation: ' + event.operation_id, event.capture_id && 'Capture: ' + event.capture_id,
      event.data.observed_at && 'Fact observed: ' + time(event.data.observed_at), event.data.evidence_ids?.length && 'Evidence: ' + event.data.evidence_ids.join(', ')].filter(Boolean);
    const technical = element('details'); technical.open = expanded.has(String(event.sequence)); technical.append(element('summary', 'Source and exact event references'), element('small', event.kind + ' · ' + refs.join(' · '))); row.append(technical); list.append(row);
  }
  list.scrollTop = follow ? list.scrollHeight : top;
  if (focusedSequence) [...list.children].find(row => row.dataset.sequence === focusedSequence)?.querySelector('summary')?.focus({preventScroll:true});
}
function validateActivityPage(data, run, after) {
  const integer = n => Number.isSafeInteger(n) && n >= 0;
  if (!data || !Array.isArray(data.items) || data.items.length > 50 ||
      ![data.next_cursor, data.oldest_sequence, data.latest_sequence].every(integer) || data.next_cursor < after || data.next_cursor > data.latest_sequence || data.oldest_sequence > data.latest_sequence ||
      (data.oldest_sequence === 0) !== (data.latest_sequence === 0) || data.has_more !== (data.next_cursor < data.latest_sequence) ||
      !['has_more','history_truncated','gap','tracked_from_creation'].every(k => typeof data[k] === 'boolean')) throw new Error('Invalid activity page; reopen this assessment to inspect retained history.');
  let previous = after;
  for (const [i, event] of data.items.entries()) {
    const sources = {'run.created':'control_plane','permission.revoked':'control_plane','activity.drained_locally':'control_plane',
      'execution.dispatch_requested':'controller','execution.worker_observed':'controller','execution.exit_observed':'controller','execution.failed':'controller',
      'cleanup.delete_accepted':'controller','cleanup.resources_absent':'controller','request.admitted':'broker','request.sent':'broker',
      'request.rejected':'broker','request.capture_persisted':'broker','validation.completed':'validator'};
    if (event.schema_version !== 1 || event.run_id !== run || (event.engagement_id && event.engagement_id !== state.view.engagement.id) ||
        !integer(event.sequence) || event.sequence <= previous || event.sequence > data.latest_sequence ||
        (event.sequence !== previous + 1 && !(i === 0 && data.gap && event.sequence === data.oldest_sequence)) ||
        typeof event.kind !== 'string' || !['control_plane','controller','broker','validator'].includes(event.source) || (sources[event.kind] && sources[event.kind] !== event.source) ||
        !event.data || typeof event.data !== 'object' || Array.isArray(event.data) || !Number.isFinite(Date.parse(event.recorded_at))) throw new Error('Activity history is inconsistent or unsupported; no records from that page were promoted.');
    const d = event.data;
    if (['reservation','read_index','account_slot','request_limit','duration_ms','status'].some(k => d[k] !== undefined && !integer(d[k])) ||
        ['code','stage','outcome','verdict','execution_mode','executor'].some(k => d[k] !== undefined && (typeof d[k] !== 'string' || d[k].length > 100)) ||
        ['observed_at','deadline'].some(k => d[k] !== undefined && !Number.isFinite(Date.parse(d[k]))) ||
        (d.evidence_ids !== undefined && (!Array.isArray(d.evidence_ids) || d.evidence_ids.length > 24 || !d.evidence_ids.every(id => idPattern.test(id)))) ||
        (d.validation !== undefined && (!d.validation || !integer(d.validation.required) || !integer(d.validation.verified) || typeof d.validation.probe_captured !== 'boolean')))
      throw new Error('Activity details are incompatible; no records from that page were promoted.');
    previous = event.sequence;
  }
  if (data.next_cursor !== previous || (data.has_more && !data.items.length)) throw new Error('Activity cursor did not match the returned records.');
}
async function lists() {
  const epoch = state.epoch;
  try {
    const apps = await api('GET', '/v1/applications');
    const engagements = await api('GET', '/v1/engagements');
    const setups = await api('GET', '/v1/onboarding');
    if (epoch !== state.epoch) return;
    state.apps = apps.sort((a, b) => a.id.localeCompare(b.id)); state.engagements = engagements;
    setupUI.lists(setups,state.apps);
    state.listsAt = new Date().toISOString();
    $('dashboard-observed-at').textContent = 'Lists received ' + time(state.listsAt) + '. Access availability is local configuration; current credential health is unknown. Registration alone grants no assessment permission.';
    renderDashboard(); state.uncertainDraft = false; controls();
  } catch (error) {
    if (epoch === state.epoch) $('dashboard-observed-at').textContent = 'Lists unavailable. ' + (state.listsAt ? 'STALE · retaining lists received ' + time(state.listsAt) + '. ' : 'No list snapshot received. ') + 'Refresh to inspect current state.';
    throw error;
  }
}
function attemptText(e) {
  const attempt = e.latest_attempt;
  if (e.latest_attempt_status === 'recorded' && attempt) return 'Latest attempt: ' + attempt.state + (attempt.verdict ? ' · ' + attempt.verdict : ' · no stored verdict');
  if (e.latest_attempt_status === 'not_started' && e.run_ids?.length === 0) return 'No attempt reserved';
  return 'Latest attempt metadata unavailable';
}
function renderDashboard() {
  const list = $('application-list'); list.replaceChildren(); list.className = 'application-list table-wrap';
  const lab = state.apps.find(app => app.id === 'vampi-vulnerable' && app.metadata?.registration === 'startup');
  $('lab-shortcut').hidden = !lab;
  const query = $('application-search').value.trim().toLowerCase(), filter = $('application-filter').value || 'all';
  const apps = state.apps.filter(app => (filter === 'all' || app.metadata?.registration === filter) &&
    [app.title, app.id, app.origin].some(text => String(text).toLowerCase().includes(query)));
  $('application-count').textContent = state.listsAt ? `${apps.length} of ${state.apps.length} registered applications shown` : 'Application snapshot not loaded.';
  const table = element('table', undefined, 'application-table'), head = element('thead'), headers = element('tr'), body = element('tbody');
  table.setAttribute('aria-label', 'Registered applications and latest saved assessments');
  for (const label of ['Application / exact target', 'Assessment & access', 'Latest saved assessment', 'Next action']) { const th = element('th', label); th.scope = 'col'; headers.append(th); }
  head.append(headers);
  for (const app of apps) {
    const row = element('tr');
    const registration = {startup:'Reviewed lab integration',onboarding:'Added through onboarding'}[app.metadata?.registration] || 'Registration source unknown';
    const readiness = !app.metadata?.procedure?.supported ? 'Procedure support unknown or unavailable' : app.metadata?.access?.availability !== 'configured' ? 'Access unavailable for assessment' : 'Scope review required before testing';
    const identity = element('td'); identity.dataset.label = 'Application';
    identity.append(element('span', registration, 'badge'), element('h2', app.title), element('code', app.id), element('small', app.origin));
    const access = element('td'); access.dataset.label = 'Assessment & access';
    access.append(element('strong', app.version === 'http-property-access/v1' ? 'Private-field exposure' : app.version === 'http-object-access/v2' ? 'Object authorization' : app.version), element('p', {configured:'Credentials configured',unavailable:'Access unavailable',configuration_changed:'Access configuration changed'}[app.metadata?.access?.availability] || 'Access availability unknown', 'hint'));
    const inspection = element('details'); inspection.append(element('summary', 'Access configuration and observations'),
      element('p', readiness, 'hint'), element('p', accessText(app.metadata?.access), 'hint'), element('small', ruleText(app.expected_access)), element('small', 'Metadata observed: ' + time(app.metadata?.observed_at)));
    access.append(inspection);
    // Join only exact application IDs. Distinct registered scopes/history are never merged.
    const saved = state.engagements.filter(e => e.application?.id === app.id).sort((a,b) => Date.parse(b.latest_attempt?.created_at || b.created_at) - Date.parse(a.latest_attempt?.created_at || a.created_at));
    const latest = saved[0], assessment = element('td'); assessment.dataset.label = 'Latest saved assessment';
    if (latest) assessment.append(element('span', attemptText(latest), 'badge'), element('small', 'Scope: ' + latest.state), element('small', latest.id), element('small', 'Attempt created: ' + time(latest.latest_attempt?.created_at)));
    else assessment.append(element('span', 'No saved assessment'), element('small', 'No security result is inferred.'));
    const action = element('td'); action.dataset.label = 'Next action';
    const b = element('button', 'Review scope →', 'button choose-app'); b.type = 'button'; b.dataset.application = app.id;
    b.addEventListener('click', () => choose(app)); action.append(b);
    if (latest) { const resume = element('button', 'Resume saved assessment', 'button subtle open-engagement'); resume.type = 'button'; resume.addEventListener('click', () => void act(() => openEngagement(latest.id))); action.append(resume); }
    row.append(identity, access, assessment, action); body.append(row);
  }
  table.append(head, body);
  if (apps.length) list.append(table);
  else list.append(element('div', state.apps.length ? 'No applications match these filters. Clear the search or choose All registrations; saved history below is unchanged.' : 'No applications registered. Add an owned application to supply prerequisites; saving details starts no tests.', 'card empty'));
  $('engagement-list').replaceChildren();
  for (const e of [...state.engagements].sort((a, b) => Date.parse(b.latest_attempt?.created_at || b.created_at) - Date.parse(a.latest_attempt?.created_at || a.created_at))) {
    const button = element('button', undefined, 'history-row open-engagement'); button.type = 'button'; button.dataset.engagement = e.id;
    const text = element('span', e.name); text.append(element('small', (e.application?.id || 'Legacy lab') + ' · ' + e.id), element('small', 'Attempt created: ' + time(e.latest_attempt?.created_at) + ' · snapshot: ' + time(e.observed_at)));
    if (e.source_engagement_id) text.append(element('small', 'Retest of assessment ' + e.source_engagement_id));
    const facts = element('span', undefined, 'history-facts'); facts.append(element('span', 'Scope: ' + e.state, 'badge'), element('span', attemptText(e), 'badge'));
    button.append(text, facts);
    button.addEventListener('click', () => act(() => openEngagement(e.id))); $('engagement-list').append(button);
  }
  if (!state.engagements.length) $('engagement-list').textContent = 'No saved assessments for this operator. Draft, approval and Start are separate actions.';
  controls();
}
function choose(app) {
  if (state.busy || state.stopping) return;
  stopObservation(); resetRetest(); state.generation++; state.view = null; state.app = app; state.run = ''; state.status = null; state.statusStale = false;
  $('scope-reviewed').checked = false; state.reviewedScope = ''; $('reviewed-digest').value = ''; delete $('reviewed-digest').dataset.scope;
  state.stopReceipt = null; resetActivity(); resetEvidence(); selectedApplication(); renderPlan(); renderStatus(); page('plan'); message('');
}
async function openEngagement(id) {
  if (!idPattern.test(id)) throw new Error('Invalid engagement ID.');
  stopObservation(); resetRetest(); const generation = ++state.generation;
  const view = await api('GET', '/v1/engagements/' + id);
  if (generation !== state.generation) return;
  if (!view.engagement.application) throw new Error('This interface supports configured application engagements. Use the CLI for this legacy lab record.');
  state.view = view; state.app = view.engagement.application; state.run = view.engagement.run_ids?.at(-1) || ''; state.status = null; state.statusStale = false;
  state.stopReceipt = null; resetActivity(); resetEvidence(); selectedApplication(); renderPlan(); renderStatus();
  $('scope-reviewed').checked = false; state.reviewedScope = '';
  page(state.run ? 'run' : 'plan');
  if (state.run) await refresh();
  controls();
}
function refresh() {
  const id = state.view?.engagement.id;
  if (!id) return Promise.resolve();
  if (state.refreshJob) return state.refreshJob.promise;
  const job = {id, generation: state.generation, epoch: state.epoch, controller: new AbortController()};
  state.refreshJob = job;
  job.promise = refreshSnapshot(job).finally(() => { if (state.refreshJob === job) state.refreshJob = null; });
  return job.promise;
}
function currentRead(job) {
  return state.refreshJob === job && !job.controller.signal.aborted && job.generation === state.generation && job.epoch === state.epoch && state.view?.engagement.id === job.id;
}
async function refreshSnapshot(job) {
  const signal = job.controller.signal;
  try {
    const view = await api('GET', '/v1/engagements/' + job.id, undefined, signal);
    if (!currentRead(job)) return;
    if (view.engagement?.id !== job.id) throw new Error('Scope response did not match the selected assessment.');
    job.scopeLoaded = true;
    state.view = view;
    if (!state.run && view.engagement.run_ids?.length) state.run = view.engagement.run_ids.at(-1);
    if (state.stopReceipt?.id === job.id && (view.engagement.state === 'stopped' || view.metadata?.permission === 'revoked')) {
      if (state.stopReceipt.phase === 'unacknowledged' && !state.stopReceipt.observedAt)
        message('Stop response was not acknowledged; a later stored assessment read shows permission revoked. Drain, worker exit and cleanup remain separate observations.');
      state.stopReceipt.observedAt = view.metadata?.observed_at || new Date().toISOString();
    }
    selectedApplication(); renderPlan(); renderStatus();
    const selectedRun = state.run;
    if (selectedRun) {
      if (state.activity?.run !== selectedRun) resetActivity();
      const outcomes = await Promise.allSettled([
        (async () => {
          try {
            const status = await api('GET', '/v1/runs/' + selectedRun + '/status', undefined, signal);
            if (!currentRead(job) || selectedRun !== state.run) return;
            if (status.run_id !== selectedRun) throw new Error('Status response did not match the selected run.');
            state.status = status; state.statusStale = false; state.statusStaleReason = ''; renderStatus();
          } catch (error) {
            if (currentRead(job) && selectedRun === state.run) { state.statusStale = true; state.statusStaleReason = 'latest read failed'; renderStatus(); }
            throw error;
          }
        })(),
        (async () => {
          try {
            for (let pages = 0; pages < 6; pages++) {
              const feed = state.activity, after = feed.cursor;
              const data = await readRunActivity(selectedRun, after, 50, signal);
              if (!currentRead(job) || selectedRun !== state.run || state.activity !== feed) return;
              validateActivityPage(data, selectedRun, after);
              feed.items.push(...data.items);
              if (feed.items.length > 256) { feed.items = feed.items.slice(-256); feed.displayTruncated = true; }
              Object.assign(feed, {cursor: data.next_cursor, loaded: true, at: new Date().toISOString(), stale: false, error: '', tracked: data.tracked_from_creation,
                truncated: feed.truncated || data.history_truncated, gap: feed.gap || data.gap, hasMore: data.has_more});
              renderActivity();
              if (!data.has_more) return;
            }
          } catch (error) {
            if (currentRead(job) && selectedRun === state.run) { state.activity.stale = true; state.activity.error = error.message; renderActivity(); }
            throw error;
          }
        })()
      ]);
      if (!currentRead(job)) return;
      const failed = outcomes.find(o => o.status === 'rejected');
      if (failed) throw failed.reason;
    }
  } catch (error) {
    if (currentRead(job)) {
      // A failed scope read prevents a new status/feed read. Never clear the
      // prior snapshots or manufacture a completion to conceal the failure.
      if (!job.scopeLoaded) {
        state.statusStale = true; state.statusStaleReason = 'latest read failed';
        if (state.activity) { state.activity.stale = true; state.activity.error = 'Assessment read failed; activity could not be refreshed.'; }
      }
      renderStatus(); renderActivity();
    }
    throw error;
  }
}
async function poll() {
  if (!state.polling) return;
  const generation = state.generation, version = state.pollVersion;
  try {
    await refresh();
    if (generation !== state.generation || version !== state.pollVersion || !state.polling) return;
    const settled = (!state.run && state.view?.engagement.state === 'stopped') || (state.status && state.status.cleanup === 'api_resources_absent' && !state.statusStale &&
      ['drained_locally', 'no_local_work'].includes(state.status.activity) && state.status.local_operations === 0 && state.status.uncertain_operations === 0 && !state.activity?.hasMore);
    if (settled || ++state.polls >= 120) {
      stopObservation();
      if (state.polls >= 120) message('Observation window ended. This did not Stop the run; inspect state or explicitly Stop.');
      return;
    }
    state.timer = setTimeout(poll, 1500);
  } catch (error) {
    if (generation !== state.generation || version !== state.pollVersion || !state.polling) return;
    stopObservation(); message('Observation failed; last displayed values may be stale. Stop is a separate action. ' + error.message, true);
  }
}
function observe() { stopObservation(); state.polls = 0; state.polling = true; $('pause').textContent = 'Pause observation'; renderActivity(); void poll(); }
async function act(fn) {
  if (state.busy || state.stopping) return;
  const action = ++state.actionID;
  state.busy = true; controls(); message('');
  try { await fn(); } catch (error) { if (action === state.actionID && error.name !== 'AbortError') message(error.message, true); }
  finally { if (action === state.actionID) { state.busy = false; controls(); } }
}
function download(name, data, type) {
  const url = URL.createObjectURL(new Blob([data], {type}));
  const link = element('a'); link.href = url; link.download = name; document.body.append(link); link.click(); link.remove();
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
function renderEvidence(bundle) {
  $('evidence-summary').textContent = `Snapshot: ${time(new Date().toISOString())} · ${bundle.finding_candidates?.length || 0} exported candidates · ${bundle.toolkit_findings?.length || 0} recorded reviews/findings · ${bundle.artifacts?.length || 0} captures. Full export explicitly loaded; it may predate later activity.`;
  $('draft-report').textContent = bundle.report_markdown || 'No report included.';
  controls();
}

function cancelFindingReads() {
  for (const key of ['findingsJob','rawJob']) { const job=state[key]; state[key]=null; job?.controller.abort(); }
  controls();
}
function findingContext(job) {
  return job.epoch===state.epoch && job.generation===state.generation && job.id===state.view?.engagement.id;
}
function loadFindings() {
  if (!state.token || !state.view) return Promise.resolve();
  if (state.findingsJob) return state.findingsJob.promise;
  if(state.rawJob){state.rawJob.controller.abort();state.rawJob=null;}
  const job={epoch:state.epoch,generation:state.generation,id:state.view.engagement.id,controller:new AbortController()};
  state.findingsJob=job;
  $('finding-summary').textContent=state.findings?'Refreshing metadata; previous snapshot '+time(state.findings.observed_at)+' remains displayed.':'Loading finding metadata. No security result is inferred while this read is pending.';
  controls();
  job.promise=(async()=>{
    try {
      const snapshot=await api('GET','/v1/engagements/'+job.id+'/findings',undefined,job.controller.signal);
      if(!findingContext(job)||state.findingsJob!==job)return;
      if(snapshot.schema_version!==1||snapshot.engagement_id!==job.id||!Array.isArray(snapshot.candidates)||!Array.isArray(snapshot.assessments)||!Array.isArray(snapshot.captures)||
        !snapshot.assessments.every(a=>idPattern.test(a.run_id)&&Array.isArray(a.checks)&&Array.isArray(a.protected_fields))||
        !snapshot.candidates.every(c=>Array.isArray(c.run_ids)&&Array.isArray(c.evidence_ids))||
        !snapshot.captures.every(c=>/^E-[a-f0-9]{64}$/.test(c.id)&&c.id==='E-'+c.sha256&&idPattern.test(c.run_id)&&idPattern.test(c.capture_id)))throw new Error('Unsupported finding metadata snapshot.');
      state.findings=snapshot;state.findingsStale=false;state.rawEvidence=null;
      $('capture-detail').hidden=true;$('capture-detail').textContent='';
      renderFindings();
    } catch(error) {
      if(findingContext(job)&&state.findingsJob===job){state.findingsStale=true;$('finding-summary').textContent='Finding metadata read failed. '+(state.findings?'STALE · retaining snapshot '+time(state.findings.observed_at)+'. ':'No snapshot received. ')+error.message;}
      throw error;
    } finally {if(state.findingsJob===job){state.findingsJob=null;controls();}}
  })();
  return job.promise;
}
const evidenceRoles={owner_identity:'Owner identity',peer_identity:'Peer identity',owner_baseline:'Owner baseline',peer_baseline:'Peer baseline',peer_probe:'Peer probe',owner_recheck:'Owner recheck',peer_identity_recheck:'Peer identity recheck',earlier_unresolved_probe:'Earlier unresolved exposure'};
function rawEvidenceButton(reference,label='Inspect raw capture') {
  const button=element('button',label,'button inspect-capture');button.type='button';button.dataset.evidence=reference.id;
  button.addEventListener('click',()=>void loadRawEvidence(reference).catch(()=>{}));return button;
}
function renderFindings() {
  const snapshot=state.findings;if(!snapshot)return;
  $('finding-summary').textContent=`Metadata snapshot: ${time(snapshot.observed_at)} · ${snapshot.candidates.length} evidence-supported candidates · ${snapshot.assessments.length} attempts. Captures require explicit inspection; this metadata read sends no target requests.`;
  $('candidates').replaceChildren();$('finding-detail').hidden=true;
  for(const candidate of snapshot.candidates){
    const card=element('article',undefined,'card candidate');card.dataset.candidate=candidate.id;
    const review=candidate.review;
    const provenance=review?`Recorded ${review.purpose} · ${review.decision} · ${review.actor} at ${time(review.reviewed_at)}${review.purpose==='automated_lab_test'?' · not human signoff':''}`:'Human review pending · severity unrated';
    card.append(element('span',candidate.status,'badge attention'),element('h2',candidate.display_title),element('p',candidate.reason),element('p',provenance,'hint'));
    if(review?.severity)card.append(element('p',`Recorded rating: ${review.severity} · CVSS ${review.cvss_score}; review purpose: ${review.purpose}`,'hint'));
    for(const id of candidate.run_ids){const button=element('button','Follow evidence and remediation · '+id,'button primary');button.dataset.evidenceRun=id;button.addEventListener('click',()=>showFindingRun(id));card.append(button);}
    const technical=element('details');technical.append(element('summary','Candidate identity and fingerprint'),element('pre',pretty({id:candidate.id,candidate_digest:candidate.candidate_digest,saved_title:candidate.title,procedure:candidate.procedure,evidence_ids:candidate.evidence_ids}),'code'));card.append(technical);$('candidates').append(card);
  }
  $('evidence-go-run').hidden=!!snapshot.assessments.length;
  if(!snapshot.candidates.length)$('candidates').append(element('p','No confirmed candidate in this snapshot. A missing, stopped, failed or inconclusive attempt is not a clean result; no_issue_observed covers only the selected rule.','hint'));
  $('attempt-evidence').replaceChildren();
  for(const attempt of snapshot.assessments){
    const button=element('button',`${attempt.run_id} · ${attempt.state} · ${attempt.result?.verdict || 'no completed validation'}`,'history-row');button.dataset.evidenceRun=attempt.run_id;
    button.addEventListener('click',()=>showFindingRun(attempt.run_id));$('attempt-evidence').append(button);
  }
  $('captures').replaceChildren();
  for(const capture of snapshot.captures){
    const row=element('div',undefined,'capture-row'+(capture.outcome==='blocked'?' blocked':''));row.dataset.capture=capture.capture_id;
    row.append(element('span',`${capture.outcome} · ${capture.status?'HTTP '+capture.status:'no HTTP status'}${capture.request_path?' · GET '+capture.request_path:''}`),rawEvidenceButton(capture));$('captures').append(row);
  }
  if(!snapshot.captures.length)$('captures').textContent='No captures in this snapshot. Missing evidence is not a clean result.';
  const selected=snapshot.assessments.find(a=>a.run_id===state.run)||snapshot.assessments.at(-1);
  if(selected)showFindingRun(selected.run_id);
  renderRetestChooser();
}
function verdictClass(verdict) {
  return 'badge' + (verdict==='confirmed'?' attention':verdict==='no_issue_observed'?' good':verdict==='inconclusive'?' info':'');
}
function observedProbeText(outcome) {
  return {expected_object_and_proof_returned:'The peer received the owner’s object and the approved private proof.', known_private_value_returned:'The peer received a known owner-only field value.', configured_fields_omitted:'The configured private fields were omitted from the shared response.', configured_denial:'The peer request received a configured denial response.', response_unproven:'The response does not establish the configured authorization outcome.', missing:'No selected peer-probe capture is available.', not_evaluated:'The peer probe has not been evaluated.', blocked:'The broker blocked this request.', failed_or_interrupted:'The request failed or was interrupted; no access outcome is inferred.'}[outcome] || 'No supported peer-probe observation is available.';
}
function showFindingRun(id) {
  const snapshot=state.findings,attempt=snapshot?.assessments.find(a=>a.run_id===id);if(!attempt)return;
  if(state.rawJob){state.rawJob.controller.abort();state.rawJob=null;}
  state.rawEvidence=null;$('capture-detail').hidden=true;$('capture-detail').textContent='';$('raw-evidence-status').textContent='No raw response loaded. Choose a specific supporting capture.';
  $('finding-detail').hidden=false;$('finding-title').textContent=attempt.display_title;
  const verdict=attempt.result?.verdict;
  $('finding-verdict').textContent=verdict || 'No completed validation'; $('finding-verdict').className=verdictClass(verdict);
  $('finding-expected').textContent=ruleText(attempt.expected_access);
  const probe=attempt.checks.find(check=>check.role==='peer_probe');
  $('finding-observed').textContent=observedProbeText(probe?.outcome)+(verdict?'':' This attempt has no completed trusted validation.');
  const review=snapshot.candidates.find(candidate=>candidate.run_ids.includes(id))?.review;
  $('finding-review').textContent=review?`Recorded review: ${review.decision} · ${review.purpose}${review.purpose==='automated_lab_test'?' · not human signoff':''}${review.severity?' · recorded severity: '+review.severity:''}.` : verdict==='confirmed'?'Evidence-supported candidate · human impact review pending · severity unrated.':'This result covers the configured rule only; it is not an application-wide security verdict or independent patch verification.';
  details($('finding-context'),[['Attempt',id],['State',attempt.state],['Read-time verdict',attempt.result?.verdict || 'No completed validation'],['Stored verdict',attempt.stored_verdict || 'Not recorded'],['Expected rule',ruleText(attempt.expected_access)],['Owner account',attempt.owner_account||'Not mapped'],['Peer account',attempt.peer_account||'Not mapped'],['Object path',attempt.object_path||'Not mapped'],['Protected field names',attempt.protected_fields.join(', ')||'Not mapped']]);
  $('finding-observation').textContent=attempt.result?.reason || 'This attempt has no completed trusted validation. Captures may be incomplete; no finding or clean result is inferred.';
  $('finding-remediation').textContent=attempt.remediation || 'No supported remediation mapping available.';
  const captures=new Map(snapshot.captures.map(c=>[c.id,c]));
  $('evidence-comparison').replaceChildren();
  for(const role of ['owner_baseline','peer_probe']){
    const check=attempt.checks.find(c=>c.role===role),reference=captures.get(check?.evidence_id),card=element('article',undefined,'card comparison-card');
    card.append(element('h3',evidenceRoles[role]));
    if(check)card.append(element('p',check.account+' · GET '+check.path),element('strong',check.outcome));
    if(reference){
      card.append(element('p','HTTP '+(reference.status||'not recorded')+' · '+time(reference.finished_at),'hint'),element('code',reference.id,'proof-reference'),rawEvidenceButton(reference));
      const provenance=element('details');provenance.append(element('summary','Capture provenance and hash'),element('pre',pretty({artifact:reference.id,run:reference.run_id,capture:reference.capture_id,sha256:reference.sha256,finished_at:reference.finished_at}),'code'));card.append(provenance);
    }
    else card.append(element('p','No selected supporting capture; this comparison remains incomplete.','hint'));
    $('evidence-comparison').append(card);
  }
  const table=element('table'),head=element('thead'),headers=element('tr'),body=element('tbody');
  for(const label of ['Check','Reviewed read','Evidence outcome','Inspect']){const th=element('th',label);th.scope='col';headers.append(th);}head.append(headers);
  for(const check of attempt.checks){
    const reference=captures.get(check.evidence_id),row=element('tr'),action=element('td');
    row.dataset.role=check.role;
    if(reference)action.append(rawEvidenceButton(reference));else action.textContent='No selected capture';
    row.append(element('td',evidenceRoles[check.role]||check.role),element('td',check.account+' · GET '+check.path),element('td',check.outcome+(reference?.status?' · HTTP '+reference.status:'')),action);body.append(row);
  }
  table.append(head,body);$('evidence-checks').replaceChildren(table);
  $('evidence-check-help').textContent=attempt.checks.length?'These check outcomes come from the validator’s selected captures. A probe response alone cannot establish a finding; all required identity, baseline and follow-up checks matter.':'Detailed check-role mapping is unavailable for this historical/catalog procedure. No role or prerequisite is invented.';
  controls();
}
async function loadRawEvidence(reference) {
  if(!state.findings?.captures.some(c=>c.id===reference.id))return;
  if(state.rawJob)state.rawJob.controller.abort();
  const job={epoch:state.epoch,generation:state.generation,id:state.view.engagement.id,snapshot:state.findings,controller:new AbortController()};state.rawJob=job;
  state.rawEvidence=null;$('capture-detail').hidden=true;$('capture-detail').textContent='';$('raw-evidence-status').textContent='Reading selected raw evidence; no target request is being replayed.';controls();
  try {
    const artifact=await api('GET','/v1/engagements/'+job.id+'/evidence/'+reference.id,undefined,job.controller.signal);
    const bytes=Uint8Array.from(atob(artifact.data_base64),c=>c.charCodeAt(0));
    const digest=[...new Uint8Array(await crypto.subtle.digest('SHA-256',bytes))].map(b=>b.toString(16).padStart(2,'0')).join('');
    const decoded=JSON.parse(new TextDecoder().decode(bytes));
    if(artifact.id!==reference.id||artifact.sha256!==reference.sha256||digest!==reference.sha256||artifact.id!=='E-'+digest||artifact.run_id!==reference.run_id||artifact.capture_id!==reference.capture_id||decoded.engagement_id!==job.id||decoded.run_id!==reference.run_id||decoded.procedure!==job.snapshot.assessments.find(a=>a.run_id===reference.run_id)?.procedure||decoded.capture?.id!==reference.capture_id)throw new Error('Evidence identity or byte hash differs from the selected metadata.');
    if(!findingContext(job)||state.rawJob!==job||state.findings!==job.snapshot)return;
    state.rawEvidence={artifact,bytes};$('capture-detail').hidden=false;$('capture-detail').textContent=pretty(decoded);
    $('raw-evidence-status').textContent='Explicit raw inspection · '+reference.capture_id+' · SHA-256 '+digest+'. Byte integrity checked; this is not independent authenticity or general redaction.';
  } catch(error){if(findingContext(job)&&state.rawJob===job)$('raw-evidence-status').textContent='Raw evidence unavailable; no new body was promoted. '+error.message;throw error;}
  finally{if(state.rawJob===job){state.rawJob=null;controls();}}
}

function cancelRetestRead() {
  const job = state.retestJob; state.retestJob = null; job?.controller.abort();
}
function resetRetest() {
  cancelRetestRead(); state.retest = null; state.retestStale = false;
  for (const id of ['retest-plan', 'retest-comparison']) { $(id).replaceChildren(); $(id).hidden = true; }
  $('retest-form').hidden = true;
  $('retest-application').replaceChildren(element('option', 'Select an owned application'));
  $('retest-selection').textContent = ''; $('retest-name').value = ''; $('retest-authorization').value = ''; $('retest-executor').value = '';
}
function renderRetestChooser() {
  $('retest-form').hidden = !state.view;
  const selected = $('retest-application').value;
  const placeholder = element('option', 'Select an owned application'); placeholder.value = '';
  $('retest-application').replaceChildren(placeholder);
  for (const app of state.apps) {
    const option = element('option', app.title + ' · ' + app.id); option.value = app.id;
    option.disabled = !app.metadata?.procedure?.supported;
    $('retest-application').append(option);
  }
  $('retest-application').value = state.apps.some(app => app.id === selected) ? selected : '';
  retestSelection();
}
function retestSelection() {
  const app = state.apps.find(app => app.id === $('retest-application').value);
  $('retest-selection').textContent = app ? `${app.origin} · ${app.version} · ${ruleText(app.expected_access)} ${accessText(app.metadata?.access)}. Exact scope changes are shown after saving the draft; no testing is authorized yet.` : 'No target selected. Choose explicitly; no fixed result is assumed.';
  controls();
}
function renderRetest() {
  const e = state.view?.engagement;
  for (const id of ['retest-plan', 'retest-comparison']) {
    const node = $(id); node.replaceChildren(); node.hidden = !e?.source_engagement_id;
    if (node.hidden) continue;
    node.append(element('h2', 'Linked retest · separate scope and evidence'));
    const refresh = element('button', 'Refresh retest comparison', 'button'); refresh.type = 'button';
    refresh.addEventListener('click', () => void loadRetestComparison().catch(() => {})); node.append(refresh);
    if (!state.retest) {
      node.append(element('p', (state.retestStale ? 'Comparison unavailable. ' : 'Loading comparison. ') + 'Source: ' + e.source_engagement_id + '. Approval waits for the scope comparison; Stop remains available.', 'hint')); continue;
    }
    const data = state.retest;
    node.append(element('p', `${state.retestStale ? 'STALE · latest read failed or permission changed. ' : ''}Comparison snapshot: ${time(data.observed_at)} · ${data.compatibility}`, 'hint'), element('p', data.reason));
    const sides = element('div', undefined, 'evidence-comparison');
    for (const [label, side] of [['Source', data.source], ['Retest', data.retest]]) {
      const card = element('article', undefined, 'comparison-card'); card.append(element('h3', label));
      const facts = element('dl', undefined, 'details'), a = side.latest_attempt;
      details(facts, [['Assessment', side.engagement.id], ['Target', side.engagement.target], ['Approval record', side.engagement.state],
        ['Scope expiry', time(side.engagement.expires_at)], ['Procedure', side.procedure.id], ['Profile/version', side.procedure.version], ['Definition digest', side.procedure.digest],
        ['Latest reserved run', a?.run_id || 'Not started'], ['Attempt state', a?.state || side.attempt_status],
        ['Read-time verdict', a?.result?.verdict || 'No completed validation'], ['Stored verdict', a?.stored_verdict || 'Not recorded']]);
      const primary = element('dl', undefined, 'details');
      details(primary, [['Target', side.engagement.target], ['Attempt', a?.state || side.attempt_status], ['Scope permission record', side.engagement.state]]);
      card.append(element('span', a?.result?.verdict || 'No completed validation', verdictClass(a?.result?.verdict) + ' comparison-verdict'),
        element('p', a?.result?.reason || 'Unfinished, stopped or failed work supplies no clean result.'), primary);
      const provenance = element('details'); provenance.append(element('summary', 'Scope, procedure and attempt provenance'), facts); card.append(provenance);
      const refs = element('details'); refs.append(element('summary', 'Selected evidence references'), element('pre', pretty(a?.evidence_ids || []), 'code')); card.append(refs);
      const open = element('button', 'Open ' + label.toLowerCase() + ' evidence', 'button'); open.type = 'button';
      open.addEventListener('click', () => void act(async () => { await openEngagement(side.engagement.id); page('evidence'); })); card.append(open); sides.append(card);
    }
    node.append(sides);
    const changes = element('details'); changes.open = id === 'retest-plan'; changes.append(element('summary', 'Changed scope and bindings · review before approval'));
    for (const diff of data.differences) {
      const row = element('div', undefined, 'scope-boundary'); row.append(element('strong', diff.field), element('p', 'Source: ' + diff.source), element('p', 'Retest: ' + diff.retest)); changes.append(row);
    }
    node.append(changes, element('p', 'This link transfers no permission or credentials. Review and approve this saved scope, then Start separately. Outcomes cover only the configured checks; no deployed patch or application-wide safety is inferred.', 'hint'));
  }
  controls();
}
function loadRetestComparison() {
  const e = state.view?.engagement;
  if (!state.token || !e?.source_engagement_id) return Promise.resolve();
  if (state.retestJob) return state.retestJob.promise;
  const job = {id:e.id, epoch:state.epoch, generation:state.generation, controller:new AbortController()};
  state.retestJob = job; renderRetest();
  job.promise = (async () => {
    try {
      const data = await api('GET', '/v1/engagements/' + job.id + '/retest', undefined, job.controller.signal);
      if (!findingContext(job) || state.retestJob !== job) return;
      if (data.schema_version !== 1 || data.retest?.engagement.id !== job.id || data.source?.engagement.id !== e.source_engagement_id || data.retest.engagement.scope_digest !== e.scope_digest || !['comparable','incompatible'].includes(data.compatibility) || !Array.isArray(data.differences) || !data.source.procedure || !data.retest.procedure) throw new Error('Unsupported retest comparison.');
      state.retest = data; state.retestStale = false; renderRetest();
    } catch (error) {
      if (findingContext(job) && state.retestJob === job) { state.retestStale = true; renderRetest(); }
      throw error;
    } finally { if (state.retestJob === job) { state.retestJob = null; controls(); } }
  })();
  return job.promise;
}

const setupUI=createOnboardingUI({api,act,message,page,openEngagement,element,details,time,pretty,actionTable,ruleText});
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
  stopObservation(); resetRetest(); state.epoch++; state.generation++; state.actionID++;
  for (const controller of state.readControllers) controller.abort();
  state.token = ''; state.view = null; state.app = null; state.run = ''; state.status = null; state.statusStale = false; state.stopReceipt = null; resetActivity();
  state.apps = []; state.engagements = []; state.listsAt = null; state.reviewedScope = ''; state.trace = []; state.selectedTrace = null; state.busy = false; state.stopping = false;
  state.startRequests.clear(); state.uncertainStarts.clear();
  setupUI.reset();
  $('application-list').replaceChildren(element('div', 'Disconnected. Reconnect to read saved state.', 'card empty'));
  $('lab-shortcut').hidden = true; $('application-count').textContent = 'Applications not loaded.';
  $('engagement-list').textContent = 'Not connected.'; $('api-log').replaceChildren(); $('api-detail').textContent = 'Disconnected.';
  $('dashboard-observed-at').textContent = 'Connect to read registered applications and saved attempts.';
  $('connect-panel').hidden = false; $('connection-state').textContent = 'Disconnected'; $('connection-state').className = 'badge';
  $('route-note').textContent = ''; $('operator-token').value = '';
  resetEvidence(); selectedApplication(); renderPlan(); renderStatus(); page('applications');
  message('Disconnected from the viewer. This did not Stop any assessment. Reconnect to inspect or revoke work.');
}
$('disconnect').addEventListener('click', disconnect);
$('application-search').addEventListener('input', renderDashboard);
$('application-filter').addEventListener('change', renderDashboard);
$('use-reviewed-lab').addEventListener('click', () => setupUI.start('vampi-vulnerable'));
for (const button of document.querySelectorAll('[data-select-application]')) button.addEventListener('click', () => page('applications'));
$('reload').addEventListener('click', () => void act(lists));
$('add-application').addEventListener('click', () => setupUI.start());
$('draft-form').addEventListener('submit', event => { event.preventDefault(); void act(async () => {
  if (!state.app || state.uncertainDraft) throw new Error('Select an application and inspect saved assessments first.');
  let created;
  try { created = await api('POST', '/v1/engagements', {application: state.app.id, name: $('assessment-name').value.trim(), authorization_ref: $('authorization-ref').value.trim(), executor: $('executor').value}); }
  catch (error) { state.uncertainDraft = true; throw new Error('Draft creation not confirmed. Refresh saved assessments before creating another. ' + error.message); }
  await openEngagement(created.id);
  message('Draft saved. No target requests. Review the exact scope before approval.');
}); });
$('reviewed-digest').addEventListener('input', controls);
$('scope-reviewed').addEventListener('input', () => { state.reviewedScope = $('scope-reviewed').checked ? state.view?.engagement.scope_digest || '' : ''; controls(); });
$('approval-form').addEventListener('submit', event => { event.preventDefault(); void act(async () => {
  const e = state.view.engagement;
  if (e.state !== 'draft' || !$('scope-reviewed').checked || state.reviewedScope !== e.scope_digest || $('reviewed-digest').value.trim() !== e.scope_digest) throw new Error('Review the displayed saved scope and explicitly approve its exact version.');
  if (e.source_engagement_id && (!state.retest || state.retestStale || state.retest.retest.engagement.scope_digest !== e.scope_digest)) throw new Error('Load and review the retest scope comparison before approving.');
  state.view = await api('POST', '/v1/engagements/' + e.id + '/approve', {scope_digest: state.reviewedScope});
  renderPlan(); message('Approval recorded. Start is a separate action; no agent was launched by approval.');
}); });
$('go-run').addEventListener('click', () => page('run'));
$('start').addEventListener('click', () => void act(async () => {
  const e = state.view.engagement;
  const generation = state.generation;
  let action = state.startRequests.get(e.id);
  if (e.state !== 'active' || e.approved_digest !== e.scope_digest || state.run || e.run_ids?.length || (state.uncertainStarts.has(e.id) && action?.scope !== e.scope_digest) || state.stopReceipt?.id === e.id) throw new Error('Start requires an approved plan; recover uncertain work with its original action identity.');
  if (!action) {
    action = {scope: e.scope_digest, request: {idempotency_key: crypto.randomUUID().replaceAll('-', ''), assessment: e.items[0].procedure, execution: 'scripted'}};
    state.startRequests.set(e.id, action);
  }
  if (action.scope !== e.scope_digest) throw new Error('Scope changed after Start; recover saved work before another action.');
  const recovering = state.uncertainStarts.has(e.id);
  state.uncertainStarts.add(e.id);
  let run;
  try { run = await api('POST', '/v1/engagements/' + e.id + '/runs', action.request); }
  catch (error) { throw new Error('Start not confirmed; no automatic retry was made. Refresh to read child IDs, or explicitly Recover Start using the same action key. ' + error.message); }
  if (generation !== state.generation || state.view?.engagement.id !== e.id) return;
  if (!idPattern.test(run.id)) throw new Error('Start response has no valid run ID; inspect saved assessment state before retrying.');
  state.uncertainStarts.delete(e.id);
  if (state.run !== run.id) { cancelRefresh(); state.run = run.id; state.status = null; resetActivity(); }
  resetEvidence(); renderPlan(); renderStatus();
  if (state.currentPage === 'run') observe(); else void refresh().catch(() => {});
  if (!state.stopReceipt) message(recovering ? 'Start action recovered. Inspect the current permission/execution status; recovery never renews permission or launches replacement work.' : 'Run recorded. Reconciliation creates/observes its Kubernetes Job. The browser does not send target requests.');
}));
async function stopAssessment() {
  const id = state.view?.engagement.id;
  if (!id || state.stopping) return;
  const generation = state.generation, epoch = state.epoch;
  stopObservation(); cancelFindingReads(); cancelRetestRead();
  if (state.retest) { state.retestStale = true; renderRetest(); }
  if(state.findings){state.findingsStale=true;$('finding-summary').textContent='STALE · Stop requested after metadata snapshot '+time(state.findings.observed_at)+'. Refresh to inspect captured work; no completion is inferred.';}
  state.statusStale = true; state.statusStaleReason = 'Stop requested after this snapshot';
  state.stopReceipt = {id, phase: 'pending'}; state.stopping = true; controls(); renderStatus();
  try {
    const receipt = await api('POST', '/v1/engagements/' + id + '/stop', {});
    if (generation !== state.generation || epoch !== state.epoch || state.view?.engagement.id !== id) return;
    if (!Array.isArray(receipt.stopped) || !receipt.stopped.every(run => idPattern.test(run))) throw new Error('Stop response was not a recognized durable acknowledgement.');
    state.stopReceipt = {id, phase: 'acknowledged', at: new Date().toISOString()};
    message('Stop acknowledged: engagement permission revoked. Request drain, worker execution and cleanup still require separate observations. Target-side effects cannot be undone.');
  } catch (error) {
    if (generation !== state.generation || epoch !== state.epoch || state.view?.engagement.id !== id) return;
    state.stopReceipt = {id, phase: 'unacknowledged'};
    message('Stop was NOT acknowledged. Do not assume permission was revoked or work ended. ' + error.message, true);
  } finally {
    if (generation === state.generation && epoch === state.epoch && state.view?.engagement.id === id) { state.stopping = false; controls(); }
  }
  // A receipt is never held hostage by a following read. Recovery is read-only;
  // neither polling nor a failed acknowledgement retries the Stop mutation.
  if (generation === state.generation && epoch === state.epoch && state.view?.engagement.id === id) {
    if (state.currentPage === 'run') observe();
    else void refresh().catch(error => { if (generation === state.generation && epoch === state.epoch && error.name !== 'AbortError') message('Follow-up observation failed. The Stop receipt remains separate; drain and cleanup are unknown.', true); });
  }
}
$('stop').addEventListener('click', () => void stopAssessment());
$('refresh-run').addEventListener('click', () => void act(refresh));
$('show-rejection').addEventListener('click', () => {
  const row = [...$('activity-events').children].find(node => node.dataset.kind === 'request.rejected');
  if (row) { row.tabIndex=-1; row.scrollIntoView({block: 'nearest'}); row.focus({preventScroll:true}); }
});
$('pause').addEventListener('click', () => {
  if (state.polling) { stopObservation(); message('Observation paused. This did not Stop the run.'); } else observe();
});
$('run-picker').addEventListener('change', () => void act(async () => {
  stopObservation(); state.generation++; state.run = $('run-picker').value; state.status = null; state.statusStale = false; resetActivity(); resetEvidence(); renderStatus(); await refresh();
}));
$('evidence-go-run').addEventListener('click', () => page('run'));
$('go-evidence').addEventListener('click', () => page('evidence'));
$('load-findings').addEventListener('click', () => { void loadFindings().catch(() => {}); void loadRetestComparison().catch(() => {}); });
$('retest-application').addEventListener('change', retestSelection);
$('retest-form').addEventListener('submit', event => { event.preventDefault(); void act(async () => {
  const source = state.view?.engagement, app = state.apps.find(app => app.id === $('retest-application').value);
  if (!source || !app?.metadata?.procedure?.supported || state.uncertainDraft) throw new Error('Select a registered supported application and recover any uncertain draft first.');
  let created;
  try {
    created = await api('POST', '/v1/engagements', {application:app.id, source_engagement_id:source.id, name:$('retest-name').value.trim(), authorization_ref:$('retest-authorization').value.trim(), executor:$('retest-executor').value});
  } catch (error) { state.uncertainDraft = true; throw new Error('Retest draft not confirmed. No retry was made; refresh saved assessments to recover its ID. ' + error.message); }
  await openEngagement(created.id);
  message('Retest draft saved with its own scope. Review changed bindings and explicitly approve, then Start separately. No permission or credential was copied.');
}); });
$('download-capture').addEventListener('click', () => {
  if(state.rawEvidence)download('evidence-'+state.rawEvidence.artifact.sha256+'.json',state.rawEvidence.bytes,'application/json');
});
$('load-evidence').addEventListener('click', () => void act(async () => {
  const id = state.view.engagement.id, generation = state.generation;
  const bundle = await api('GET', '/v1/engagements/' + id + '/export');
  if (generation !== state.generation || state.view?.engagement.id !== id) return;
  state.bundle = bundle; renderEvidence(bundle);
}));
$('download-bundle').addEventListener('click', () => { if (state.bundle) download('tsunami-' + state.view.engagement.id + '-bundle.json', pretty(state.bundle), 'application/json'); });
$('download-report').addEventListener('click', () => { if (state.bundle) download('tsunami-' + state.view.engagement.id + '-draft.md', state.bundle.report_markdown, 'text/markdown'); });
window.addEventListener('pagehide', disconnect);
controls();
