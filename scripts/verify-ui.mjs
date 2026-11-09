#!/usr/bin/env node
// Real Chrome + the owned Kubernetes lab. DevTools protocol reference:
// https://chromedevtools.github.io/devtools-protocol/
// No npm dependencies. Credentials remain in process/page memory, not argv,
// URLs, output, screenshots, traces, browser storage or retained test records.
import {spawn, spawnSync} from 'node:child_process';
import {mkdtemp, mkdir, readFile, writeFile, readdir, rm} from 'node:fs/promises';
import path from 'node:path';
import os from 'node:os';
import net from 'node:net';
import {createHash} from 'node:crypto';

const root = process.cwd();
const env = {...process.env, KUBECONFIG: path.resolve('.local/sandbox.kubeconfig')};
const out = path.resolve('artifacts/ui/browser-' + new Date().toISOString().replace(/[-:.]/g, ''));
await mkdir(out, {recursive: true, mode: 0o700});
const summary = {version: 'tsunami.ui-qualification/v1', checked_at: new Date().toISOString(), passed: false,
  context: 'kind-tsunami-lab', execution: 'scripted_mecatl', approval: 'automated_owned_lab', external_model_calls: 0, cases: [], checks: []};
const includeDemo=process.argv.includes('--demo'), rehearse=process.argv.includes('--rehearse');
check(!rehearse || includeDemo, '--rehearse requires --demo');
const coreOnly=process.argv.includes('--core-only');
check(!coreOnly || includeDemo, '--core-only requires --demo');
const includeProperties=includeDemo&&!coreOnly;
const includeOnboarding=process.argv.includes('--onboarding') || includeDemo;
summary.includes_onboarding=includeOnboarding;
summary.includes_d9=includeDemo; summary.assets={}; summary.rehearsal={requested:rehearse,kind:'automated_paced_browser_dry_run; no recorded human narration',stages:[]};
summary.includes_property_campaign=includeProperties;
summary.driver_sha256=createHash('sha256').update(await readFile('scripts/verify-ui.mjs')).digest('hex');
summary.binary_sha256=createHash('sha256').update(await readFile('bin/tsunami')).digest('hex');
summary.onboarding=[];
let forward, chrome, socket, profile, origin, token = '', callID = 0;
const diagnosticForwards=[];
const sensitiveValues=[];
const pending = new Map(), network = [], exceptions = [];
function check(condition, message) { if (!condition) throw new Error(message); }
function command(argv) {
  const result = spawnSync(argv[0], argv.slice(1), {env, encoding: 'utf8', maxBuffer: 4 << 20});
  if (result.status !== 0) throw new Error('Prerequisite/local command failed; inspect owned lab state.');
  return result.stdout;
}
const sleep = ms => new Promise(resolve => setTimeout(resolve, ms));
async function wait(fn, label, timeout = 20000) {
  const end = Date.now() + timeout;
  do { if (await fn()) return; await sleep(100); } while (Date.now() < end);
  throw new Error('Timed out: ' + label);
}
function cdp(method, params = {}) {
  const id = ++callID;
  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => { pending.delete(id); reject(new Error('Browser protocol response timed out.')); }, 20000);
    pending.set(id, {resolve, reject, timer}); socket.send(JSON.stringify({id, method, params}));
  });
}
async function evaluate(expression) {
  const result = await cdp('Runtime.evaluate', {expression, awaitPromise: true, returnByValue: true});
  if (result.exceptionDetails) throw new Error('Browser evaluation failed (details suppressed).');
  return result.result.value;
}
const click = id => evaluate(`document.getElementById(${JSON.stringify(id)}).click()`);
const value = id => evaluate(`document.getElementById(${JSON.stringify(id)}).textContent`);
const visible = id => evaluate(`!document.getElementById(${JSON.stringify(id)}).hidden`);
async function fill(id, text) { await evaluate(`document.getElementById(${JSON.stringify(id)}).value=${JSON.stringify(text)};document.getElementById(${JSON.stringify(id)}).dispatchEvent(new Event('input',{bubbles:true}))`); }
async function snapshot(name) {
  const result = await cdp('Page.captureScreenshot', {format: 'png', captureBeyondViewport: false});
  await writeFile(path.join(out, name + '.png'), Buffer.from(result.data, 'base64'), {mode: 0o600});
}
async function operator(method, route, body) {
  const response = await fetch(origin + route, {method, redirect: 'error', headers: {Authorization: 'Bearer ' + token, 'Content-Type': 'application/json'}, body: body === undefined ? undefined : JSON.stringify(body)});
  check(response.ok, 'Owned operator observation/cleanup failed.');
  return response.json();
}
async function stopOwn(id) {
  await operator('POST', '/v1/engagements/' + id + '/stop', {});
}
async function fixtureOrigin(family) {
  const existing=diagnosticForwards.find(p=>p.family===family);if(existing)return existing.origin;
  const config={stop:['tsunami-stop-lab','tsunami-stop-fixture'],property:['tsunami-property-lab','tsunami-property-fixture']}[family];
  check(!!config,'Unknown owned fixture diagnostic.');
  const listener=net.createServer();await new Promise(resolve=>listener.listen(0,'127.0.0.1',resolve));const port=listener.address().port;await new Promise(resolve=>listener.close(resolve));
  const proc=spawn('kubectl',['--context','kind-tsunami-lab','-n',config[0],'port-forward','--address=127.0.0.1','service/'+config[1],port+':8080'],{env,stdio:'ignore'});
  const endpoint='http://127.0.0.1:'+port;diagnosticForwards.push({family,origin:endpoint,proc});
  await wait(async()=>{try{return (await fetch(endpoint+'/healthz')).ok;}catch{return false;}},'owned diagnostic forward');return endpoint;
}
async function fixtureStats(application) {
  const family=application==='stop-lab-run'?'stop':'property';
  check(application==='stop-lab-run'||['property-vulnerable','property-filtered','property-ambiguous'].includes(application),'Unapproved diagnostic application.');
  const name=family==='stop'?'run':application.slice('property-'.length), endpoint=await fixtureOrigin(family);
  const response=await fetch(endpoint+'/cases/'+name+'/stats',{redirect:'error'});check(response.ok,'Fixture observations unavailable.');return response.json();
}
async function collectActivity(record) {
  const events=await operator('GET','/v1/runs/'+record.run_id+'/events?after=0&limit=100');
  check(events.tracked_from_creation&&!events.history_truncated&&!events.has_more,'Incomplete current activity record.');
  check(events.items.some(e=>e.kind==='request.rejected'),'Actual scope rejection missing from activity.');
  check(record.verdict==='stopped'?!events.items.some(e=>e.kind==='validation.completed'):events.items.some(e=>e.kind==='validation.completed'&&e.data.verdict===record.verdict),'Activity validation disagrees with outcome.');
  await click('refresh-run');await wait(()=>evaluate(`!document.getElementById('refresh-run').disabled`),'activity refresh');
  const displayed=await evaluate(`JSON.stringify([...document.querySelectorAll('#activity-events [data-sequence]')].map(n=>Number(n.dataset.sequence)))`);
  check(displayed===JSON.stringify(events.items.map(e=>e.sequence)),'Displayed activity differs from persisted events.');
  record.activity='events-'+record.run_id+'.json';await writeFile(path.join(out,record.activity),JSON.stringify(events,null,2)+'\n',{mode:0o600});
  await click('show-rejection');await snapshot(record.application+'-actual-activity');
}
async function openEvidenceMetadata() {
  const before=network.filter(r=>r.path.endsWith('/export')||r.path.includes('/evidence/')).length;
  await click('go-evidence');await wait(()=>evaluate(`document.getElementById('finding-summary').textContent.startsWith('Metadata snapshot:')`),'finding metadata');
  check(network.filter(r=>r.path.endsWith('/export')||r.path.includes('/evidence/')).length===before,'Normal finding navigation downloaded raw evidence.');
}
async function journey(application, expected, held = false) {
  await evaluate(`document.querySelector('[data-page="applications"]').click()`);
  await click('reload');
  await wait(() => evaluate(`!document.getElementById('reload').disabled && !!document.querySelector('[data-application=${JSON.stringify(application)}]')`), 'application list');
  await evaluate(`document.querySelector('[data-application=${JSON.stringify(application)}]').click()`);
  await fill('authorization-ref', 'owned-lab/ui-qualification');
  await fill('assessment-name', 'UI qualification: ' + application);
  const before = network.filter(r => r.method === 'POST' && r.path.endsWith('/runs')).length;
  await click('create-draft');
  await wait(() => visible('plan-card'), 'draft preview');
  const id = await evaluate(`document.querySelector('#scope-details dd').textContent`);
  check(/^[a-f0-9]{32}$/.test(id), 'Draft ID missing.');
  summary.cases.push({application, engagement_id: id, passed: false});
  const record = summary.cases.at(-1);
  if(includeDemo&&(held||application.startsWith('property-')))record.target_before=await fixtureStats(application);
  const digest = await evaluate(`[...document.querySelectorAll('#scope-details dt')].find(x=>x.textContent==='Scope digest').nextElementSibling.textContent`);
  check(network.filter(r => r.method === 'POST' && r.path.endsWith('/runs')).length === before, 'Draft unexpectedly launched work.');
  await fill('reviewed-digest', '0'.repeat(64));
  await evaluate(`document.getElementById('scope-reviewed').checked=true;document.getElementById('scope-reviewed').dispatchEvent(new Event('input'))`);
  check(await evaluate(`document.getElementById('approve').disabled`), 'Wrong digest enabled approval.');
  await fill('reviewed-digest', digest);
  await wait(() => evaluate(`!document.getElementById('approve').disabled`), 'approval readiness');
  await click('approve');
  await wait(async () => await value('engagement-state') === 'active', 'approval');
  await wait(() => evaluate(`!document.getElementById('start').disabled`), 'start readiness');
  await click('go-run');
  const lostStart = application === 'vampi-fixed';
  if (lostStart) {
    // Deliver the real POST, then hide its response from the UI. This is a
    // client failure injection, not a production partition qualification.
    await evaluate(`window.beforeLostStart=window.fetch;window.fetch=async (...args)=>{const response=await window.beforeLostStart(...args);if(args[0]==='/v1/engagements/${id}/runs'&&response.ok)throw new TypeError('Simulated lost Start response after commit');return response;}`);
  }
  await evaluate(`document.getElementById('start').click();document.getElementById('start').click()`);
  if (lostStart) {
    await wait(() => evaluate(`document.getElementById('message').textContent.startsWith('Start not confirmed')`), 'lost Start acknowledgement');
    check(await evaluate(`document.getElementById('start').disabled`), 'Uncertain Start could be retried.');
    await evaluate(`window.fetch=window.beforeLostStart;delete window.beforeLostStart`);
    await click('refresh-run');
    await wait(() => evaluate(`!document.getElementById('refresh-run').disabled`), 'lost Start recovery read');
    record.lost_start_response_recovered = true;
    await click('pause'); // Resume read-only observation after recovering IDs.
  }
  await wait(() => evaluate(`document.querySelector('#run-context dd:nth-of-type(3)').textContent.match(/^[a-f0-9]{32}$/)!==null`), 'run ID');
  record.run_id = await evaluate(`document.querySelector('#run-context dd:nth-of-type(3)').textContent`);
  check(network.filter(r => r.method === 'POST' && r.path.endsWith('/runs')).length === before + 1, 'Double start created duplicate requests.');
  if (held) {
    await wait(async () => {
      if (await value('status-activity') !== 'active_locally') return false;
      const status = await operator('GET', '/v1/runs/' + record.run_id + '/status');
      return status.activity === 'active_locally' && status.target_requests_reserved === 5;
    }, 'held request observation');
    if(includeDemo){record.target_held=await fixtureStats(application);check(record.target_held.active===1&&record.target_held.probe_started===record.target_before.probe_started+1,'Target did not observe an active held probe.');}
    await click('pause');
    check(await value('status-permission') === 'active', 'Pausing observation changed permission.');
    const stopsBefore = network.filter(r => r.method === 'POST' && r.path.endsWith('/stop')).length;
    await sleep(300);
    check(network.filter(r => r.method === 'POST' && r.path.endsWith('/stop')).length === stopsBefore, 'Pause sent Stop.');
    await snapshot('held-before-stop');
    // Hold a real export response in the browser. Stop must remain usable
    // independently of this busy viewer operation.
    await evaluate(`window.beforeHeldRead=window.fetch;window.fetch=async (...args)=>{const response=await window.beforeHeldRead(...args);if(args[0]==='/v1/engagements/${id}/export')await new Promise(resolve=>window.releaseViewerRead=resolve);return response;}`);
    await click('go-evidence'); await click('load-evidence');
    await wait(() => evaluate(`typeof window.releaseViewerRead==='function'`), 'held viewer read');
    check(await evaluate(`document.getElementById('load-evidence').disabled && !document.getElementById('stop').disabled`), 'Busy read disabled Stop.');
    await evaluate(`document.querySelector('[data-page="run"]').click()`);
    if(includeDemo)check(Date.now()-Date.parse(record.target_held.probe_started_at)<8000,'Held request too close to its deadline to attribute cancellation to Stop.');
    await click('stop');
    await wait(async () => await value('status-permission') === 'revoked', 'Stop revocation');
    await evaluate(`window.fetch=window.beforeHeldRead;delete window.beforeHeldRead;window.releaseViewerRead();delete window.releaseViewerRead`);
    await wait(() => evaluate(`!document.getElementById('load-evidence').disabled`), 'held viewer completion');
    record.stop_during_pending_export = true;
    await wait(async () => await value('status-cleanup') === 'api_resources_absent', 'Stop cleanup', 25000);
    record.verdict = 'stopped';
    if(includeDemo){await wait(async()=>{record.target_cancelled=await fixtureStats(application);return record.target_cancelled.active===0&&record.target_cancelled.probe_cancelled===record.target_held.probe_cancelled+1;},'target-observed cancellation');record.target_final=await fixtureStats(application);check(record.target_final.requests===record.target_held.requests&&record.target_final.probe_completed===record.target_held.probe_completed,'Stop allowed additional target work.');}
  } else {
    await wait(async () => await value('verdict') === expected, 'trusted verdict', 35000);
    await wait(async () => await value('status-cleanup') === 'api_resources_absent', 'completed cleanup', 20000);
    record.verdict = await value('verdict');
  }
  const status = await operator('GET', '/v1/runs/' + record.run_id + '/status');
  record.status = status;
  if(includeDemo){
    await collectActivity(record);
    if(application.startsWith('property-')){record.target_final=await fixtureStats(application);check(record.target_final.requests-record.target_before.requests===7&&record.target_final.probe_started-record.target_before.probe_started===1,'Property workflow departed from seven bounded reads.');}
  }
  check(status.cleanup === 'api_resources_absent', 'Cleanup observation missing.');
  if (held) check(status.permission === 'revoked' && status.local_operations === 0, 'Stop did not drain local broker operations.');
  await openEvidenceMetadata(); await click('load-evidence');
  await wait(() => evaluate(`document.getElementById('evidence-summary').textContent.startsWith('Snapshot:')`), 'evidence');
  await wait(() => evaluate(`document.getElementById('finding-summary').textContent.startsWith('Metadata snapshot:')`), 'finding metadata');
  const candidates = await evaluate(`document.querySelectorAll('[data-candidate]').length`);
  check(candidates === (expected === 'confirmed' ? 1 : 0), 'Candidate count differs.');
  check(await evaluate(`document.querySelectorAll('.capture-row.blocked').length===1`), 'Scope denial capture missing.');
  await evaluate(`document.querySelector('.capture-row.blocked button').click()`);
  await wait(()=>evaluate(`document.getElementById('raw-evidence-status').textContent.includes('Byte integrity checked')`),'explicit blocked artifact inspection');
  check((await value('capture-detail')).includes('not-approved.invalid'), 'Blocked intent details missing.');
  await snapshot(application);
  await click('download-bundle');
  const filename = 'tsunami-' + id + '-bundle.json';
  await wait(async () => (await readdir(path.join(out, 'downloads'))).includes(filename), 'bundle download');
  const raw = await readFile(path.join(out, 'downloads', filename));
  check(!raw.includes(Buffer.from(token)), 'Operator credential appeared in export.');
  const bundle = JSON.parse(raw);
  check(bundle.view.engagement.id === id && bundle.toolkit_findings.length === 0, 'Export identity/review differs.');
  for (const artifact of bundle.artifacts) check(createHash('sha256').update(Buffer.from(artifact.data_base64, 'base64')).digest('hex') === artifact.sha256, 'Downloaded artifact hash differs.');
  const materialized = path.join(out, application);
  command(['./bin/tsunami', 'export-bundle', '--input', path.join(out, 'downloads', filename), '--out', materialized]);
  record.bundle = path.relative(out, path.join(materialized, 'bundle.json'));
  record.candidates = candidates;
  record.blocked_captures = 1;
  record.passed = true;
  await stopOwn(id);
  console.log('PASS browser ' + application + ': ' + record.verdict + '; one explicit run; unreviewed evidence exported.');
}

async function onboardingJourney(){
  await evaluate(`document.querySelector('[data-page="setup"]').click()`);
  let id,session;
  if(!includeDemo){
  await click('new-setup');
  await fill('setup-name','Frontend-only metadata example');await fill('setup-frontend','https://frontend.example.test');await fill('setup-reference','owned-lab/no-external-requests');
  await click('save-setup');
  await wait(()=>evaluate(`document.querySelector('#setup-question form')?.dataset.question==='api_origin'`),'missing backend question');
  id=(await evaluate(`document.querySelector('#setup-facts dd').textContent`)).slice(0,32);
  summary.onboarding.push({setup_id:id,case:'missing_backend',target_requests:0});
  session=await operator('GET','/v1/onboarding/'+id);
  check(session.state==='needs_input'&&session.question.field==='api_origin'&&session.validation_requests===0,'Frontend-only setup considered ready.');
  await snapshot('onboarding-missing-api');
  await click('stop-setup');await wait(async()=>await value('setup-state')==='stopped','metadata Stop');
  await click('new-setup');await fill('setup-name','Unsupported login example');await fill('setup-frontend','https://frontend.example.test');await fill('setup-reference','owned-lab/no-login');
  await fill('setup-api','https://api.example.test');
  await evaluate(`document.getElementById('setup-auth').value='password'`);
  await click('save-setup');await wait(async()=>await value('setup-state')==='unsupported','unsupported login guard');
  id=(await evaluate(`document.querySelector('#setup-facts dd').textContent`)).slice(0,32);
  summary.onboarding.push({setup_id:id,case:'unsupported_login',target_requests:0});
  session=await operator('GET','/v1/onboarding/'+id);
  check(session.validation_requests===0&&!session.access_ready,'Unsupported login attempted access.');
  await click('stop-setup');await wait(async()=>await value('setup-state')==='stopped','unsupported setup Stop');
  }

  await click('new-setup');await fill('setup-name','Onboarded owned VAmPI');await fill('setup-frontend','https://owned-frontend.example.test');await fill('setup-reference','owned-lab/onboarding-browser');
  await evaluate(`document.getElementById('setup-source').value='vampi-vulnerable';document.getElementById('setup-synthetic').checked=true`);
  await click('save-setup');await wait(async()=>await value('setup-state')==='ready_for_validation','onboarding known prerequisites');
  id=(await evaluate(`document.querySelector('#setup-facts dd').textContent`)).slice(0,32);
  const record={setup_id:id,case:'owned_onboarding',passed:false};summary.onboarding.push(record);
  session=await operator('GET','/v1/onboarding/'+id);
  check(!session.question&&!session.plan&&!session.engagement_id&&session.validation_requests===0,'Known facts were re-asked or assessment silently started.');
  // Exercise the protected-input path with the same synthetic credentials.
  // Only the trusted harness reads this Secret; no value is printed/persisted.
  const registry=JSON.parse(Buffer.from(command(['kubectl','--context','kind-tsunami-lab','-n','tsunami','get','secret','tsunami-applications','-o','jsonpath={.data.config\\.json}']).trim(),'base64').toString());
  const credentials=registry.find(x=>x.tenant==='demo-a'&&x.plan.id==='vampi-vulnerable').credentials;
  sensitiveValues.push(credentials.owner,credentials.peer);
  await click('update-setup-access');
  await evaluate(`(()=>{const inputs=document.querySelectorAll('#setup-question input[type="password"]');inputs[0].value=${JSON.stringify(credentials.owner)};inputs[1].value=${JSON.stringify(credentials.peer)};})()`);
  await evaluate(`document.querySelector('#setup-question button[type="submit"]').click()`);
  await wait(()=>evaluate(`!document.getElementById('validate-setup').disabled || document.querySelector('#setup-question input[type="password"]')===null`),'protected access save');
  check(!(await evaluate(`document.body.textContent`)).includes(credentials.owner)&&!(await evaluate(`document.body.textContent`)).includes(credentials.peer),'Protected test token displayed in chat/API trace.');
  const starts=network.filter(r=>r.method==='POST'&&r.path.endsWith('/runs')).length;
  await evaluate(`document.getElementById('setup-preflight-approved').checked=true;document.getElementById('setup-preflight-approved').dispatchEvent(new Event('input'))`);
  await click('validate-setup');
  await wait(async()=>await value('setup-state')==='ready_for_approval','four legitimate validation reads',25000);
  session=await operator('GET','/v1/onboarding/'+id);
  check(session.validation_requests===4&&session.plan&&session.plan.min_interval_ms===1000&&!session.engagement_id,'Preflight did not produce a bounded unapproved plan.');
  check(network.filter(r=>r.method==='POST'&&r.path.endsWith('/runs')).length===starts,'Validation silently launched security test.');
  record.validation_requests=4;record.validation_id=session.validation_id;record.application=session.plan.id;
  await writeFile(path.join(out,'onboarding-session.json'),JSON.stringify(session,null,2)+'\n',{mode:0o600});
  await snapshot('onboarding-ready');
  await click('prepare-setup');await wait(()=>evaluate(`!document.querySelector('[data-panel="plan"]').hidden && document.querySelector('#scope-details dd:nth-of-type(2)')?.textContent===${JSON.stringify(record.application)} && !document.getElementById('refresh-run').disabled`),'onboarded scope');
  const engagement=await evaluate(`document.querySelector('#scope-details dd').textContent`);
  record.engagement_id=engagement;
  // Track immediately, so failed later observations still revoke our own work.
  summary.cases.push(record);
  check(await evaluate(`!document.getElementById('scope-reviewed').checked`),'Prepared scope inherited approval.');
  const unapproved=await operator('GET','/v1/engagements/'+engagement);
  check(unapproved.engagement.state==='draft'&&!unapproved.engagement.approved_at&&!unapproved.engagement.run_ids.length,'Prepare approved or started assessment.');
  const rows=await evaluate(`JSON.stringify([...document.querySelectorAll('#scope-actions tbody tr')].map(r=>[r.cells[0].textContent,r.cells[1].textContent]))`);
  check(rows===JSON.stringify([...session.plan.identity,...session.plan.baselines,session.plan.probe,session.plan.baselines[0],session.plan.identity[1]].map(r=>[r.account,'GET '+r.path])),'Displayed scope differs from approved reads.');
  await snapshot('onboarded-scope-review');
  const digest=await evaluate(`[...document.querySelectorAll('#scope-details dt')].find(x=>x.textContent==='Scope digest').nextElementSibling.textContent`);
  await fill('reviewed-digest',digest);await evaluate(`document.getElementById('scope-reviewed').checked=true;document.getElementById('scope-reviewed').dispatchEvent(new Event('input'))`);
  await wait(()=>evaluate(`!document.getElementById('approve').disabled`),'onboarded scope approval');await click('approve');
  await wait(async()=>await value('engagement-state')==='active','onboarded approval');await click('go-run');
  await wait(()=>evaluate(`!document.getElementById('start').disabled`),'onboarded Start');await click('start');
  await wait(()=>evaluate(`document.querySelector('#run-context dd:nth-of-type(3)')?.textContent.match(/^[a-f0-9]{32}$/)!==null`),'onboarded run ID');
  record.run_id=await evaluate(`document.querySelector('#run-context dd:nth-of-type(3)').textContent`);
  await wait(async()=>{
    const pods=JSON.parse(command(['kubectl','--context','kind-tsunami-lab','-n','tsunami','get','pods','-l','tsunami-run='+record.run_id,'-o','json'])).items;
    const pod=pods[0];if(!pod||pod.status.phase!=='Running')return false;
    check(pod.spec.runtimeClassName==='gvisor'&&pod.spec.nodeName==='tsunami-lab-control-plane','Unexpected onboarding worker runtime/node.');
    const sandboxes=JSON.parse(command(['docker','exec',pod.spec.nodeName,'crictl','pods','-o','json'])).items;
    const sandbox=sandboxes.find(x=>x.metadata.uid===pod.metadata.uid);if(!sandbox)return false;
    const runtime=JSON.parse(command(['docker','exec',pod.spec.nodeName,'crictl','inspectp',sandbox.id]));
    record.actual_runtime={handler:runtime.status.runtimeHandler,type:runtime.info.runtimeType};
    check(record.actual_runtime.handler==='runsc'&&record.actual_runtime.type==='io.containerd.runsc.v1','Actual onboarding worker did not use gVisor.');
    const containers=JSON.parse(command(['docker','exec',pod.spec.nodeName,'crictl','ps','-a','--pod',sandbox.id,'-o','json'])).containers;
    const container=containers.find(x=>x.metadata.name==='agent');if(!container)return false;
    record.worker_image_id=JSON.parse(command(['docker','exec',pod.spec.nodeName,'crictl','inspect',container.id])).status.imageRef;
    check(!!record.worker_image_id,'Actual onboarding worker image missing.');return true;
  },'onboarded actual sandbox inspection',12000);
  await wait(async()=>await value('verdict')==='confirmed','onboarded actual finding',35000);
  await wait(async()=>await value('status-cleanup')==='api_resources_absent','onboarded cleanup',20000);
  record.run_id=await evaluate(`document.querySelector('#run-context dd:nth-of-type(3)').textContent`);
  record.status=await operator('GET','/v1/runs/'+record.run_id+'/status');
  check(record.status.target_requests_reserved===7&&record.status.assessment.validation.baselines_verified===6,'Onboarded runtime validation incomplete.');
  record.verdict='confirmed';if(includeDemo)await collectActivity(record);
  await openEvidenceMetadata();await click('load-evidence');await wait(()=>evaluate(`document.getElementById('evidence-summary').textContent.startsWith('Snapshot:')`),'onboarded evidence');
  await wait(()=>evaluate(`document.getElementById('finding-summary').textContent.startsWith('Metadata snapshot:')`),'onboarded finding metadata');
  check(await evaluate(`document.querySelectorAll('[data-candidate]').length===1 && document.querySelectorAll('.capture-row.blocked').length===1`),'Onboarded finding/scope capture missing.');
  await snapshot('onboarded-finding');await click('download-bundle');
  const filename='tsunami-'+engagement+'-bundle.json';await wait(async()=> (await readdir(path.join(out,'downloads'))).includes(filename),'onboarded download');
  const raw=await readFile(path.join(out,'downloads',filename));check(!raw.includes(Buffer.from(token)),'Operator token in onboarding export.');
  const bundle=JSON.parse(raw);for(const a of bundle.artifacts)check(createHash('sha256').update(Buffer.from(a.data_base64,'base64')).digest('hex')===a.sha256,'Onboarded artifact hash differs.');
  command(['./bin/tsunami','export-bundle','--input',path.join(out,'downloads',filename),'--out',path.join(out,'onboarded-vampi')]);
  record.bundle='onboarded-vampi/bundle.json';record.verdict='confirmed';record.candidates=1;record.passed=true;
  record.blocked_captures=1;
  await operator('POST','/v1/onboarding/'+id+'/stop',{});
  console.log(includeDemo?'PASS browser onboarding: four approved preflight reads; explicit scope approval; real gVisor/Mecatl assessment and candidate.':'PASS browser onboarding: missing backend asked; unsupported login refused; four preflight reads; explicit approval; real gVisor/Mecatl assessment and candidate.');
  return record;
}

async function inspectFinding(record){
  await click('load-findings');await wait(()=>evaluate(`!document.getElementById('load-findings').disabled`),'fresh finding metadata');
  const metadata=await operator('GET','/v1/engagements/'+record.engagement_id+'/findings'),attempt=metadata.assessments.find(a=>a.run_id===record.run_id);
  check(attempt?.result?.verdict==='confirmed'&&attempt.checks.length===7&&metadata.candidates.length===1,'Finding support incomplete.');
  check(metadata.candidates[0].status==='needs_human_review'&&!metadata.candidates[0].review,'Finding acquired an invented review.');
  check((await value('finding-title')).includes('Cross-user')&&(await value('finding-remediation')).includes('object read'),'Failure-specific explanation/remediation missing.');
  const observed=[];
  for(const role of ['owner_baseline','peer_probe']){
    const support=attempt.checks.find(c=>c.role===role),reference=metadata.captures.find(c=>c.id===support.evidence_id);
    await evaluate(`document.querySelector('#evidence-checks tr[data-role="${role}"] button').click()`);
    await wait(()=>evaluate(`document.getElementById('raw-evidence-status').textContent.includes('Byte integrity checked')`),'selected artifact integrity');
    const decoded=JSON.parse(await value('capture-detail'));check(decoded.capture.id===support.capture_id&&decoded.run_id===record.run_id,'Selected capture provenance differs.');
    observed.push(JSON.parse(decoded.capture.body));
    await click('download-capture');const filename='evidence-'+reference.sha256+'.json';
    await wait(async()=>{try{return createHash('sha256').update(await readFile(path.join(out,'downloads',filename))).digest('hex')===reference.sha256;}catch{return false;}},'original artifact byte download');
  }
  const pointer=(body,name)=>name.split('/').slice(1).reduce((v,k)=>v?.[k.replaceAll('~1','/').replaceAll('~0','~')],body);
  for(const name of attempt.protected_fields)check(pointer(observed[0],name)!==undefined&&pointer(observed[0],name)===pointer(observed[1],name),'Owner/peer protected proof does not correspond.');
  record.finding_metadata='findings-'+record.run_id+'.json';await writeFile(path.join(out,record.finding_metadata),JSON.stringify(metadata,null,2)+'\n',{mode:0o600});
  await snapshot('finding-selected-proof');
}

async function openSavedEvidence(id){
  await evaluate(`document.querySelector('[data-page="applications"]').click()`);await click('reload');await wait(()=>evaluate(`!document.getElementById('reload').disabled&&!!document.querySelector('[data-engagement="${id}"]')`),'saved assessment history');
  await evaluate(`document.querySelector('[data-engagement="${id}"]').click()`);await wait(()=>evaluate(`!document.getElementById('refresh-run').disabled`),'saved assessment recovery');
  await openEvidenceMetadata();
}

async function linkedRetest(source){
  await openSavedEvidence(source.engagement_id);
  const original=await operator('GET','/v1/engagements/'+source.engagement_id+'/export'),before=network.filter(r=>r.method==='POST'&&r.path.endsWith('/runs')).length;
  await fill('retest-application','vampi-fixed');await evaluate(`document.getElementById('retest-application').dispatchEvent(new Event('change'))`);
  await fill('retest-name','D9 linked fixed-app retest');await fill('retest-authorization','owned-lab/D9-new-scope');await fill('retest-executor','curl-http/v1');await click('create-retest');
  await wait(()=>evaluate(`document.getElementById('retest-plan').textContent.includes('Comparison snapshot:')&&document.getElementById('engagement-state').textContent==='draft'`),'linked draft scope comparison');
  const id=await evaluate(`document.querySelector('#scope-details dd').textContent`),record={application:'vampi-fixed',engagement_id:id,source_engagement_id:source.engagement_id,passed:false};summary.cases.push(record);
  const comparison=await operator('GET','/v1/engagements/'+id+'/retest');
  check(comparison.compatibility==='comparable'&&comparison.retest.engagement.source_engagement_id===source.engagement_id&&comparison.source.engagement.target!==comparison.retest.engagement.target&&!comparison.retest.latest_attempt&&!comparison.retest.engagement.approved_at,'Retest inherited scope/result/approval.');
  check(await evaluate(`!document.getElementById('scope-reviewed').checked&&document.getElementById('approve').disabled`),'Retest inherited review.');
  check(comparison.differences.some(d=>d.field==='Origin')&&comparison.differences.some(d=>d.field==='Definition digest'),'Changed bindings not shown.');
  await snapshot('retest-new-scope');
  await evaluate(`document.getElementById('scope-reviewed').checked=true;document.getElementById('scope-reviewed').dispatchEvent(new Event('input'))`);await click('approve');await wait(async()=>await value('engagement-state')==='active','fresh retest approval');
  check(network.filter(r=>r.method==='POST'&&r.path.endsWith('/runs')).length===before,'Approval started retest.');await click('go-run');
  await evaluate(`window.beforeLostStart=window.fetch;window.fetch=async(...args)=>{const r=await window.beforeLostStart(...args);if(args[0]==='/v1/engagements/${id}/runs'&&r.ok)throw new TypeError('Simulated lost Start response after commit');return r}`);
  await click('start');await wait(()=>evaluate(`document.getElementById('message').textContent.startsWith('Start not confirmed')`),'lost committed retest Start');
  check(await evaluate(`document.getElementById('start').disabled`),'Uncertain Start allowed replay.');await evaluate(`window.fetch=window.beforeLostStart;delete window.beforeLostStart`);
  await click('refresh-run');await wait(()=>evaluate(`!document.getElementById('refresh-run').disabled`),'retained run recovery');await click('pause');
  await wait(async()=>await value('verdict')==='no_issue_observed','actual fixed retest outcome',35000);await wait(async()=>await value('status-cleanup')==='api_resources_absent','retest cleanup',20000);
  record.run_id=await evaluate(`document.querySelector('#run-context dd:nth-of-type(3)').textContent`);record.verdict='no_issue_observed';record.status=await operator('GET','/v1/runs/'+record.run_id+'/status');record.lost_start_response_recovered=true;
  check(network.filter(r=>r.method==='POST'&&r.path.endsWith('/runs')).length===before+1,'Lost response created duplicate work.');await collectActivity(record);
  await openEvidenceMetadata();await wait(()=>evaluate(`document.getElementById('retest-comparison').textContent.includes('no_issue_observed')`),'actual linked outcome comparison');
  const completed=await operator('GET','/v1/engagements/'+id+'/retest'),preserved=await operator('GET','/v1/engagements/'+source.engagement_id+'/export');
  check(completed.source.latest_attempt.result.verdict==='confirmed'&&completed.retest.latest_attempt.result.verdict==='no_issue_observed'&&completed.retest.latest_attempt.evidence_ids.length===7,'Linked actual results/evidence differ.');
  check(JSON.stringify(original.finding_candidates)===JSON.stringify(preserved.finding_candidates)&&JSON.stringify(original.artifacts)===JSON.stringify(preserved.artifacts),'Retest changed original evidence.');
  record.comparison='linked-retest.json';await writeFile(path.join(out,record.comparison),JSON.stringify(completed,null,2)+'\n',{mode:0o600});
  await evaluate(`document.getElementById('retest-comparison').scrollIntoView()`);await snapshot('linked-retest-results');
  await click('load-evidence');await wait(()=>evaluate(`document.getElementById('evidence-summary').textContent.startsWith('Snapshot:')`),'retest explicit export');await click('download-bundle');
  const filename='tsunami-'+id+'-bundle.json';await wait(async()=> (await readdir(path.join(out,'downloads'))).includes(filename),'retest download');command(['./bin/tsunami','export-bundle','--input',path.join(out,'downloads',filename),'--out',path.join(out,'linked-vampi-fixed')]);
  record.bundle='linked-vampi-fixed/bundle.json';record.candidates=0;record.blocked_captures=1;record.passed=true;await stopOwn(id);return record;
}

async function demoJourney(){
  summary.version=coreOnly?'tsunami.ui-demo-core-qualification/v1':'tsunami.ui-demo-qualification/v1';
  const start=performance.now();let source;
  async function stage(offset,deadline,name,fn){
    if(rehearse){while(performance.now()-start<offset*1000)await sleep(Math.min(1000,offset*1000-(performance.now()-start)));}
    const entry={name,scheduled_start_seconds:offset,deadline_seconds:deadline,started_seconds:(performance.now()-start)/1000};summary.rehearsal.stages.push(entry);
    console.log('D9 stage: '+name);await fn();entry.finished_seconds=(performance.now()-start)/1000;entry.within_deadline=entry.finished_seconds<=deadline;
    check(!rehearse||entry.within_deadline,'Rehearsal exceeded deadline: '+name);
  }
  await stage(0,30,'Problem and architecture',async()=>{await snapshot('demo-dashboard');});
  await stage(30,90,'Add application, scope review, approval and real agent',async()=>{source=await onboardingJourney();});
  await stage(90,150,'Finding, reproducible evidence and remediation',async()=>{await inspectFinding(source);});
  await stage(150,200,'Stop during an active request and pending viewer export',async()=>{await journey('stop-lab-run','stopped',true);});
  await stage(200,255,'Fresh linked fixed-app retest and lost Start recovery',async()=>{await linkedRetest(source);});
  await stage(255,300,'Limitations and original history',async()=>{await openSavedEvidence(source.engagement_id);check((await value('finding-title')).includes('Cross-user'),'Original finding unavailable after retest.');await snapshot('original-finding-preserved');});
  if(rehearse){while(performance.now()-start<300000)await sleep(Math.min(1000,300000-(performance.now()-start)));}
  summary.rehearsal.elapsed_seconds=(performance.now()-start)/1000;summary.rehearsal.passed=rehearse&&summary.rehearsal.stages.every(s=>s.within_deadline)&&summary.rehearsal.elapsed_seconds>=300&&summary.rehearsal.elapsed_seconds<302;
  if(rehearse)check(summary.rehearsal.passed,'Timed rehearsal did not complete within the five-minute window.');
  console.log(includeProperties?'D9 core journey complete; separate property qualification follows outside presentation time.':'Core-only rehearsal complete; property campaign explicitly excluded.');
  if(includeProperties)for(const [app,verdict] of [['property-vulnerable','confirmed'],['property-filtered','no_issue_observed'],['property-ambiguous','inconclusive']])await journey(app,verdict);
}

try {
  // Read-only guard. Refuses other clusters, expired prerequisites and active work.
  const prereq = command(['python3', 'scripts/showcase.py', 'check', ...(includeProperties?['--include-property']:[])]);
  await writeFile(path.join(out, 'prerequisites.txt'), prereq, {mode: 0o600});
  token = Buffer.from(command(['kubectl', '--context', 'kind-tsunami-lab', '-n', 'tsunami', 'get', 'secret', 'tsunami-operator', '-o', 'jsonpath={.data.admin-a}']).trim(), 'base64').toString();
  sensitiveValues.push(token);
  const listener = net.createServer(); await new Promise(resolve => listener.listen(0, '127.0.0.1', resolve));
  const port = listener.address().port; await new Promise(resolve => listener.close(resolve)); origin = 'http://127.0.0.1:' + port;
  forward = spawn('kubectl', ['--context', 'kind-tsunami-lab', '-n', 'tsunami', 'port-forward', '--address=127.0.0.1', 'service/tsunami-control', port + ':8080'], {env, stdio: 'ignore'});
  await wait(async () => { try { return (await fetch(origin + '/ui/')).ok; } catch { return false; } }, 'local UI port-forward');
  for (const name of ['index.html', 'app.js', 'onboarding.js', 'styles.css']) {
    const response = await fetch(origin + '/ui/' + (name === 'index.html' ? '' : name));
    const served = Buffer.from(await response.arrayBuffer());
    const local = await readFile(path.join(root, 'internal/webui/static', name));
    check(response.ok && served.equals(local), 'Deployed UI differs from local source; rebuild before qualification.');
    summary.assets[name]=createHash('sha256').update(served).digest('hex');
  }
  profile = await mkdtemp(path.join(os.tmpdir(), 'tsunami-ui-browser-'));
  chrome = spawn('google-chrome', ['--headless=new', '--disable-gpu', '--no-proxy-server', '--no-first-run', '--no-default-browser-check', '--disable-background-networking', '--disable-component-update', '--disable-default-apps', '--disable-sync', '--disable-extensions', '--metrics-recording-only', '--remote-debugging-address=127.0.0.1', '--remote-debugging-port=0', '--user-data-dir=' + profile, 'about:blank'], {stdio: 'ignore'});
  let debugPort;
  await wait(async () => { try { debugPort = (await readFile(path.join(profile, 'DevToolsActivePort'), 'utf8')).split('\n')[0]; return !!debugPort; } catch { return false; } }, 'isolated Chrome');
  const targets = await (await fetch('http://127.0.0.1:' + debugPort + '/json/list')).json();
  socket = new WebSocket(targets.find(t => t.type === 'page').webSocketDebuggerUrl);
  await new Promise((resolve, reject) => { socket.onopen = resolve; socket.onerror = reject; });
  socket.onmessage = event => {
    const data = JSON.parse(event.data);
    if (data.id) {
      const p = pending.get(data.id); if (!p) return;
      pending.delete(data.id); clearTimeout(p.timer);
      if (data.error) p.reject(new Error('Browser protocol command failed.')); else p.resolve(data.result);
    } else if (data.method === 'Runtime.exceptionThrown') exceptions.push('Browser runtime exception');
    else if (data.method === 'Network.requestWillBeSent') {
      const request = data.params.request, url = new URL(request.url);
      if (url.protocol === 'http:' || url.protocol === 'https:') network.push({method: request.method, path: url.pathname, same_origin: url.origin === origin});
    }
  };
  await cdp('Runtime.enable'); await cdp('Page.enable'); await cdp('Network.enable');
  await cdp('Emulation.setDeviceMetricsOverride', {width: 1480, height: 1100, deviceScaleFactor: 1, mobile: false});
  await mkdir(path.join(out, 'downloads'), {mode: 0o700});
  await cdp('Browser.setDownloadBehavior', {behavior: 'allow', downloadPath: path.join(out, 'downloads')});
  summary.navigation = await cdp('Page.navigate', {url: origin + '/ui/'});
  await wait(() => evaluate(`document.readyState==='complete' && !!document.getElementById('connect')`), 'interface');
  check(!network.some(r => r.path.startsWith('/v1/')), 'Page load contacted authenticated APIs without explicit connection.');
  await snapshot('disconnected');
  await fill('operator-token', 'invalid-owned-lab-operator-token'); await click('connect');
  await wait(() => evaluate(`document.getElementById('message').classList.contains('error')`), 'unauthenticated refusal');
  check(await visible('connect-panel'), 'Invalid credential connected.');
  await fill('operator-token', token); await click('connect');
  await wait(async () => !(await visible('connect-panel')), 'operator connection');
  check(await evaluate(`localStorage.length===0 && sessionStorage.length===0 && document.cookie==='' && document.getElementById('operator-token').value===''`), 'Credential/storage hygiene failed.');
  check(!(await evaluate(`document.body.textContent`)).includes(token), 'Operator token visible in interface.');
  await snapshot('applications');
  if(includeDemo){
    const history=await operator('GET','/v1/engagements');check(history.length+6<=100,'Six-assessment campaign exceeds remaining history capacity.');
    summary.initial_engagement_count=history.length;
    await demoJourney();
  }else{
    await journey('vampi-vulnerable', 'confirmed');
    await journey('vampi-fixed', 'no_issue_observed');
    await journey('stop-lab-run', 'stopped', true);
    if(includeOnboarding)await onboardingJourney();
  }
  // Hostile API display text must remain text. This changes only a browser-side
  // application-list response, never the target, evidence or recorded verdicts.
  await evaluate(`window.originalFetch=window.fetch;window.fetch=async (...args)=>{const response=await window.originalFetch(...args);if(args[0]==='/v1/applications'&&response.ok){const data=await response.json();for(const app of data)app.title='<img src=x onerror="window.uiInjectionExecuted=true">';return new Response(JSON.stringify(data),{status:200,headers:{'Content-Type':'application/json'}});}return response;}`);
  await click('reload'); await wait(() => evaluate(`!document.getElementById('reload').disabled`), 'display injection fixture');
  await evaluate(`document.querySelector('[data-application="vampi-fixed"]').click()`);
  check(await evaluate(`document.getElementById('selected-application').textContent.includes('<img') && !document.querySelector('#selected-application img') && !window.uiInjectionExecuted`), 'Untrusted API text became executable HTML.');
  await evaluate(`window.fetch=window.originalFetch;delete window.originalFetch`);
  await cdp('Emulation.setDeviceMetricsOverride', {width: 390, height: 844, deviceScaleFactor: 1, mobile: true});
  await evaluate(`document.querySelector('[data-page="applications"]').click()`);
  check(await evaluate(`document.documentElement.scrollWidth<=window.innerWidth+1`), 'Mobile layout overflow.');
  await snapshot('mobile');
  await click('disconnect');
  check(await visible('connect-panel'), 'Disconnect failed.');
  check(await evaluate(`document.querySelectorAll('[data-candidate]').length===0 && document.getElementById('operator-token').value==='' && document.getElementById('api-log').children.length===0`), 'Disconnect retained sensitive display state.');
  check(network.every(r => r.same_origin), 'Browser contacted another origin.');
  check(exceptions.length === 0, 'Browser runtime exception occurred.');
  summary.checks = ['invalid operator refused', 'no automatic assessment on connect', 'wrong digest disables approval', 'one POST per Start despite double click',
    'scope denial captured', 'pause does not Stop', 'held request revocation and local drain observed', 'downloaded artifact hashes match',
    'operator token absent from DOM/storage/export', 'lost committed Start response recovered without retry', 'Stop available during held viewer export',
    'hostile API text rendered as text', 'mobile layout within viewport', 'disconnect clears display', 'same-origin requests only', 'no browser runtime exceptions'];
  if(includeDemo)summary.checks.push('D6 readable preflight and exact scope matrix','D4 events correspond to displayed records','D7 metadata-only navigation and explicit artifact byte verification','D8 fresh changed-origin approval and linked actual comparison','original candidate/artifact identity preserved','target-observed active Stop cancellation',coreOnly?'three explicit core assessment attempts; property campaign excluded; no automatic replay':'six explicit assessment attempts; no automatic replay');
  if(includeProperties)summary.checks.push('property leak/filter/ambiguity use independent real scopes');
  summary.passed = true;
  console.log('PASS browser qualification. Records: ' + path.relative(root, out));
} catch (error) {
  summary.failure = String(error.message);for(const secret of sensitiveValues){if(secret)summary.failure=summary.failure.replaceAll(secret,'[credential omitted]');}
  console.error('Browser qualification failed: ' + summary.failure);
  if (socket?.readyState === WebSocket.OPEN) {
    try {
      summary.browser_diagnostic = await evaluate(`({url:location.href,title:document.title,ready:document.readyState,text:document.body?.textContent.slice(0,1200)})`);
      const unsafe=sensitiveValues.some(s=>s&&JSON.stringify(summary.browser_diagnostic).includes(s));
      if(unsafe)summary.browser_diagnostic={note:'Sensitive display suppressed.'};else await snapshot('failure');
    } catch { /* Preserve the original failure. */ }
  }
  process.exitCode = 1;
} finally {
  // Only revoke engagements created by this driver. Never silently stop others.
  for (const c of summary.cases) { try { await stopOwn(c.engagement_id); } catch { summary.cleanup_unconfirmed = true; } }
  for(const s of summary.onboarding){try{await operator('POST','/v1/onboarding/'+s.setup_id+'/stop',{});}catch{summary.cleanup_unconfirmed=true;}}
  if (summary.cleanup_unconfirmed) { summary.passed = false; process.exitCode = 1; }
  if (socket?.readyState === WebSocket.OPEN) socket.close();
  for (const p of pending.values()) { clearTimeout(p.timer); p.reject(new Error('Browser session ended.')); }
  pending.clear();
  chrome?.kill(); forward?.kill();for(const diagnostic of diagnosticForwards)diagnostic.proc.kill(); token = '';
  if (profile) { await sleep(700); await rm(profile, {recursive: true, force: true}); }
  await writeFile(path.join(out, 'summary.json'), JSON.stringify(summary, null, 2) + '\n', {mode: 0o600});
  await writeFile(path.join(out, 'operator-requests.json'), JSON.stringify(network, null, 2) + '\n', {mode: 0o600});
}
