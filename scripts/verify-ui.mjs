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
const includeOnboarding=process.argv.includes('--onboarding');
summary.includes_onboarding=includeOnboarding;
summary.onboarding=[];
let forward, chrome, socket, profile, origin, token = '', callID = 0;
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
    await click('stop');
    await wait(async () => await value('status-permission') === 'revoked', 'Stop revocation');
    await evaluate(`window.fetch=window.beforeHeldRead;delete window.beforeHeldRead;window.releaseViewerRead();delete window.releaseViewerRead`);
    await wait(() => evaluate(`!document.getElementById('load-evidence').disabled`), 'held viewer completion');
    record.stop_during_pending_export = true;
    await wait(async () => await value('status-cleanup') === 'api_resources_absent', 'Stop cleanup', 25000);
    record.verdict = 'stopped';
  } else {
    await wait(async () => await value('verdict') === expected, 'trusted verdict', 35000);
    await wait(async () => await value('status-cleanup') === 'api_resources_absent', 'completed cleanup', 20000);
    record.verdict = await value('verdict');
  }
  const status = await operator('GET', '/v1/runs/' + record.run_id + '/status');
  record.status = status;
  check(status.cleanup === 'api_resources_absent', 'Cleanup observation missing.');
  if (held) check(status.permission === 'revoked' && status.local_operations === 0, 'Stop did not drain local broker operations.');
  await click('go-evidence'); await click('load-evidence');
  await wait(() => evaluate(`document.getElementById('evidence-summary').textContent.startsWith('Snapshot:')`), 'evidence');
  const candidates = await evaluate(`document.querySelectorAll('[data-candidate]').length`);
  check(candidates === (expected === 'confirmed' ? 1 : 0), 'Candidate count differs.');
  check(await evaluate(`document.querySelectorAll('.capture-row.blocked').length===1`), 'Scope denial capture missing.');
  await evaluate(`document.querySelector('.capture-row.blocked').click()`);
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
  await click('new-setup');
  await fill('setup-name','Frontend-only metadata example');await fill('setup-frontend','https://frontend.example.test');await fill('setup-reference','owned-lab/no-external-requests');
  await click('save-setup');
  await wait(()=>evaluate(`document.querySelector('#setup-question form')?.dataset.question==='api_origin'`),'missing backend question');
  let id=(await evaluate(`document.querySelector('#setup-facts dd').textContent`)).slice(0,32);
  summary.onboarding.push({setup_id:id,case:'missing_backend',target_requests:0});
  let session=await operator('GET','/v1/onboarding/'+id);
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
  await click('go-evidence');await click('load-evidence');await wait(()=>evaluate(`document.getElementById('evidence-summary').textContent.startsWith('Snapshot:')`),'onboarded evidence');
  check(await evaluate(`document.querySelectorAll('[data-candidate]').length===1 && document.querySelectorAll('.capture-row.blocked').length===1`),'Onboarded finding/scope capture missing.');
  await snapshot('onboarded-finding');await click('download-bundle');
  const filename='tsunami-'+engagement+'-bundle.json';await wait(async()=> (await readdir(path.join(out,'downloads'))).includes(filename),'onboarded download');
  const raw=await readFile(path.join(out,'downloads',filename));check(!raw.includes(Buffer.from(token)),'Operator token in onboarding export.');
  const bundle=JSON.parse(raw);for(const a of bundle.artifacts)check(createHash('sha256').update(Buffer.from(a.data_base64,'base64')).digest('hex')===a.sha256,'Onboarded artifact hash differs.');
  command(['./bin/tsunami','export-bundle','--input',path.join(out,'downloads',filename),'--out',path.join(out,'onboarded-vampi')]);
  record.bundle='onboarded-vampi/bundle.json';record.verdict='confirmed';record.candidates=1;record.passed=true;
  summary.cases.push({...record,application:record.application,blocked_captures:1});
  await operator('POST','/v1/onboarding/'+id+'/stop',{});
  console.log('PASS browser onboarding: missing backend asked; unsupported login refused; four preflight reads; explicit approval; real gVisor/Mecatl assessment and candidate.');
}

try {
  // Read-only guard. Refuses other clusters, expired prerequisites and active work.
  const prereq = command(['python3', 'scripts/showcase.py', 'check']);
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
  await journey('vampi-vulnerable', 'confirmed');
  await journey('vampi-fixed', 'no_issue_observed');
  await journey('stop-lab-run', 'stopped', true);
  if(includeOnboarding)await onboardingJourney();
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
  chrome?.kill(); forward?.kill(); token = '';
  if (profile) { await sleep(700); await rm(profile, {recursive: true, force: true}); }
  await writeFile(path.join(out, 'summary.json'), JSON.stringify(summary, null, 2) + '\n', {mode: 0o600});
  await writeFile(path.join(out, 'operator-requests.json'), JSON.stringify(network, null, 2) + '\n', {mode: 0o600});
}
