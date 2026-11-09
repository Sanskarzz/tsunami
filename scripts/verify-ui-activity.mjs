#!/usr/bin/env node
// Invoked only by the opt-in Go test's disposable loopback fixture. Real UI,
// broker, Mecatl and target traffic; no cluster, external target/model, npm
// dependency or product-data injection. Failure injections delay/drop real
// viewer responses after delivery, never replace backend findings or events.
import {spawn} from 'node:child_process';
import {mkdtemp,mkdir,readFile,writeFile,rm} from 'node:fs/promises';
import {once} from 'node:events';
import os from 'node:os';
import path from 'node:path';
import {createHash} from 'node:crypto';

const origin=process.env.TSUNAMI_TEST_ORIGIN,token=process.env.TSUNAMI_TEST_TOKEN;
const onboarding=process.argv.includes('--onboarding');
const findings=process.argv.includes('--findings');
const retest=process.argv.includes('--retest');
if(!/^http:\/\/127\.0\.0\.1:\d+$/.test(origin||'')||!token)throw new Error('Use make verify-ui-activity-browser with its disposable loopback fixture.');
const out=path.resolve('artifacts/'+(onboarding?'ui-onboarding':findings?'ui-findings':retest?'ui-retest':'ui-activity')+'/local-'+new Date().toISOString().replace(/[-:.]/g,''));
await mkdir(out,{recursive:true,mode:0o700});
const summary={version:onboarding?'tsunami.local-ui-onboarding-test/v1':findings?'tsunami.local-ui-findings-test/v1':retest?'tsunami.local-ui-retest-test/v1':'tsunami.local-ui-activity-test/v1',checked_at:new Date().toISOString(),passed:false,execution:'scripted_mecatl_local_scheduler',kubernetes_qualified:false,cases:[],checks:[]};
const pending=new Map(),network=[],exceptions=[];
let profile,chrome,socket,callID=0;
const externalProfile=onboarding?process.env.TSUNAMI_TEST_CHROME_PROFILE:undefined;
const check=(value,label)=>{if(!value)throw new Error(label);};
const sleep=ms=>new Promise(resolve=>setTimeout(resolve,ms));
async function wait(fn,label,timeout=12000){const end=Date.now()+timeout;do{if(await fn())return;await sleep(40);}while(Date.now()<end);throw new Error('Timed out: '+label);}
function cdp(method,params={}){return new Promise((resolve,reject)=>{const id=++callID;const timer=setTimeout(()=>{pending.delete(id);reject(new Error('Browser protocol timeout'));},12000);pending.set(id,{resolve,reject,timer});socket.send(JSON.stringify({id,method,params}));});}
async function evaluate(expression){const result=await cdp('Runtime.evaluate',{expression,awaitPromise:true,returnByValue:true});if(result.exceptionDetails)throw new Error('Browser evaluation failed; details suppressed');return result.result.value;}
const click=id=>evaluate(`document.getElementById(${JSON.stringify(id)}).click()`);
const text=id=>evaluate(`document.getElementById(${JSON.stringify(id)}).textContent`);
async function fill(id,value){await evaluate(`document.getElementById(${JSON.stringify(id)}).value=${JSON.stringify(value)};document.getElementById(${JSON.stringify(id)}).dispatchEvent(new Event('input',{bubbles:true}))`);}
async function operator(route,method='GET',body){const res=await fetch(origin+route,{method,body:body===undefined?undefined:JSON.stringify(body),headers:{Authorization:'Bearer '+token,'Content-Type':'application/json'},redirect:'error'});check(res.ok,'operator observation failed');return res.json();}
const startCount=()=>network.filter(r=>r.method==='POST'&&r.path.endsWith('/runs')).length;
async function prepare(application){
  await evaluate(`document.querySelector('[data-page="applications"]').click()`);await click('reload');
  await wait(()=>evaluate(`!document.getElementById('reload').disabled && !!document.querySelector('[data-application="${application}"]')`),'application list');
  await evaluate(`document.querySelector('[data-application="${application}"]').click()`);
  await fill('authorization-ref','owned-local-ui-test');await fill('assessment-name','Local activity check: '+application);await fill('executor','');
  await click('create-draft');await wait(()=>evaluate(`!document.getElementById('plan-card').hidden && !document.getElementById('reload').disabled`),'scope loaded');
  const id=await evaluate(`document.querySelector('#scope-details dd').textContent`);
  const digest=await evaluate(`[...document.querySelectorAll('#scope-details dt')].find(n=>n.textContent==='Scope digest').nextElementSibling.textContent`);
  await fill('reviewed-digest',digest);await evaluate(`document.getElementById('scope-reviewed').checked=true;document.getElementById('scope-reviewed').dispatchEvent(new Event('input'))`);
  await click('approve');await wait(async()=>await text('engagement-state')==='active','explicit approval');
  await click('go-run');await wait(()=>evaluate(`!document.getElementById('start').disabled`),'Start ready');
  return id;
}
async function runID(id){const view=await operator('/v1/engagements/'+id);return view.engagement.run_ids.at(-1);}
async function heldRequest(id){await wait(async()=>{const run=await runID(id);if(!run)return false;const status=await operator('/v1/runs/'+run+'/status');return status.activity==='active_locally'&&status.target_requests_reserved===5;},'real held broker request',6000);}
async function drain(){await wait(async()=>await text('status-activity')==='drained_locally','observed local drain',6000);check(await text('status-execution')==='dispatch_intent','local test invented Kubernetes worker/exit observation');}

async function onboardingJourney(){
  const applications=await operator('/v1/applications'),app=applications.find(a=>a.id==='browser-vulnerable');
  const observations=()=>operator('/test/observations');
  check((await observations()).requests===0,'connection made target requests');
  check(await evaluate(`!document.getElementById('api-inspector').open`),'technical inspector expanded by default');
  async function save(name,auth='',source=''){
    await click('new-setup');await fill('setup-name',name);await fill('setup-environment','dev');await fill('setup-frontend','https://owned-frontend.example.test');await fill('setup-reference','owned-local-ui-onboarding');
    if(auth){await fill('setup-api',app.origin);await fill('setup-auth',auth);}
    if(source){await fill('setup-source',source);await evaluate(`document.getElementById('setup-source').dispatchEvent(new Event('change'))`);check((await text('setup-source-details')).includes(app.origin),'lab reuse target missing');}
    await evaluate(`document.getElementById('setup-synthetic').checked=true`);await click('save-setup');
    await wait(()=>evaluate(`!document.getElementById('setup-current').hidden && !document.getElementById('refresh-setup').disabled`),'saved setup');
    return (await evaluate(`document.querySelector('#setup-facts dd').textContent`)).slice(0,32);
  }
  for(const auth of ['password','sso']){
    const id=await save('Unsupported '+auth,auth);
    check(await text('setup-state')==='unsupported','unsupported auth appeared available');
    check(await evaluate(`document.getElementById('validate-setup').disabled && document.getElementById('prepare-setup').disabled`),'unsupported auth enabled execution');
    check((await observations()).requests===0,'unsupported auth made target requests');
    summary.cases.push({setup_id:id,case:'unsupported_'+auth,target_requests:0});await click('stop-setup');await wait(async()=>await text('setup-state')==='stopped','unsupported setup Stop');
  }
  const missing=await save('Missing backend facts');
  check(await evaluate(`document.querySelector('#setup-question form')?.dataset.question==='api_origin'`),'missing API question not shown');
  check((await observations()).requests===0,'missing facts caused target requests');
  summary.cases.push({setup_id:missing,case:'missing_api',target_requests:0});await click('stop-setup');await wait(async()=>await text('setup-state')==='stopped','incomplete setup Stop');
  const id=await save('Owned integration','',app.id);
  await wait(async()=>await text('setup-state')==='ready_for_validation','reused facts');
  check((await observations()).requests===0,'saving reused facts granted target traffic');
  await click('update-setup-access');
  await evaluate(`(()=>{const fields=document.querySelectorAll('#setup-question input[type="password"]');fields[0].value=${JSON.stringify('a'.repeat(48))};fields[1].value=${JSON.stringify('b'.repeat(48))};document.querySelector('#setup-question button[type="submit"]').click()})()`);
  await wait(()=>evaluate(`!document.querySelector('#setup-question input[type="password"]') && !document.getElementById('edit-setup').disabled`),'protected access saved');
  const old=await operator('/v1/onboarding/'+id);
  await evaluate(`document.getElementById('setup-preflight-approved').checked=true;document.getElementById('setup-preflight-approved').dispatchEvent(new Event('input'))`);
  check(await evaluate(`!document.getElementById('validate-setup').disabled`),'current grant cannot be reviewed');
  await click('edit-setup');await fill('setup-name','Owned <img src=x onerror=alert(1)>');await click('save-setup');
  await wait(()=>evaluate(`!document.getElementById('edit-setup').disabled && document.getElementById('setup-form').hidden`),'revised setup');
  check(await evaluate(`!document.getElementById('setup-preflight-approved').checked && document.getElementById('validate-setup').disabled`),'changed setup retained preflight approval');
  const stale=await fetch(origin+'/v1/onboarding/'+id+'/validate',{method:'POST',headers:{Authorization:'Bearer '+token,'Content-Type':'application/json'},body:JSON.stringify({revision:old.revision,digest:old.preflight_digest,confirm:true})});
  check(stale.status===409,'stale grant accepted');check((await observations()).requests===0,'stale grant caused target traffic');
  check(await evaluate(`document.querySelectorAll('#setup-preflight-actions tbody tr').length===4`),'four-read review table missing');
  check(await evaluate(`!document.querySelector('img[src="x"]') && document.getElementById('setup-title').textContent.includes('<img')`),'setup text was interpreted as HTML');
  await evaluate(`document.getElementById('setup-validation').scrollIntoView()`);
  const preflightShot=await cdp('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});await writeFile(path.join(out,'preflight-review.png'),Buffer.from(preflightShot.data,'base64'),{mode:0o600});
  await evaluate(`document.getElementById('setup-preflight-approved').checked=true;document.getElementById('setup-preflight-approved').dispatchEvent(new Event('input'))`);await click('validate-setup');
  await wait(async()=>await text('setup-state')==='ready_for_approval','real four-read validation',20000);
  const ready=await operator('/v1/onboarding/'+id),preflight=await observations();
  check(ready.validation_requests===4&&preflight.requests===4&&preflight.probes===0&&!ready.engagement_id,'preflight made a security probe or extra request');
  await click('prepare-setup');await wait(()=>evaluate(`!document.querySelector('[data-panel="plan"]').hidden && !document.getElementById('refresh-run').disabled`),'saved unapproved scope');
  const eID=await evaluate(`document.querySelector('#scope-details dd').textContent`),view=await operator('/v1/engagements/'+eID);
  check(view.engagement.state==='draft'&&startCount()===0&&(await observations()).requests===4,'prepare silently approved/started work');
  const rows=await evaluate(`JSON.stringify([...document.querySelectorAll('#scope-actions tbody tr')].map(row=>[...row.cells].slice(0,2).map(cell=>cell.textContent)))`);
  const p=view.engagement.application,reads=[...p.identity,...p.baselines,p.probe,p.baselines[0],p.identity[1]];
  check(rows===JSON.stringify(reads.map(r=>[r.account,'GET '+r.path])),'readable matrix differs from saved scope');
  check(await evaluate(`!document.getElementById('scope-technical').open && document.getElementById('reviewed-digest').readOnly`),'raw scope/digest editing required');
  await evaluate(`document.getElementById('scope-summary').scrollIntoView()`);
  const scopeShot=await cdp('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});await writeFile(path.join(out,'scope-review.png'),Buffer.from(scopeShot.data,'base64'),{mode:0o600});
  await cdp('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:false});check(await evaluate(`document.documentElement.scrollWidth<=innerWidth`),'scope narrow layout overflow');
  await cdp('Emulation.setDeviceMetricsOverride',{width:1360,height:1050,deviceScaleFactor:1,mobile:false});
  await evaluate(`document.getElementById('scope-reviewed').checked=true;document.getElementById('scope-reviewed').dispatchEvent(new Event('input'))`);await click('approve');
  await wait(async()=>await text('engagement-state')==='active','explicit scope approval');check((await observations()).requests===4&&startCount()===0,'approval launched assessment');
  await click('go-run');await click('start');await wait(async()=>await text('verdict')==='confirmed','onboarded actual assessment',20000);
  const actual=await observations();check(actual.requests===11&&actual.probes===1&&startCount()===1,'assessment departed from bounded approved workflow');
  await evaluate(`document.querySelector('[data-page="applications"]').click()`);await click('reload');await wait(()=>evaluate(`!document.getElementById('reload').disabled`),'dashboard history');
  check(await evaluate(`(()=>{const row=document.querySelector('[data-engagement="${eID}"]');return row?.textContent.includes('Scope: active')&&row.textContent.includes('Latest attempt: completed · confirmed')})()`),'dashboard confused completed result and approval');
  check(await evaluate(`!document.querySelector('img[src="x"]')`),'dashboard rendered target text as HTML');
  const shot=await cdp('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});await writeFile(path.join(out,'onboarding-dashboard.png'),Buffer.from(shot.data,'base64'),{mode:0o600});
  summary.cases.push({setup_id:id,engagement_id:eID,run_id:await runID(eID),case:'owned_add_to_assessment',validation_requests:4,assessment_requests:7,verdict:'confirmed',stale_grant_rejected:true});
  summary.checks=['invalid operator token rejected','missing API facts asked','password/SSO unavailable without traffic','lab reuse target explicit','protected access omitted from UI inspection','changed setup clears review; stale grant rejected without traffic','four legitimate preflight reads; zero probes before separate scope approval/Start','readable account/action matrix matches stored scope','approve exact scope without JSON/digest typing','one real scripted Mecatl assessment','dashboard scope and stored result separate','injection rendered as text','narrow scope layout'];
}

async function findingsJourney(){
  const directory=path.join(out,'downloads');await mkdir(directory,{mode:0o700});await cdp('Browser.setDownloadBehavior',{behavior:'allow',downloadPath:directory});
  for(const [application,verdict] of [['browser-vulnerable','confirmed'],['browser-fixed','no_issue_observed'],['browser-property','confirmed'],['browser-property-filtered','no_issue_observed'],['browser-property-ambiguous','inconclusive']]){
    const id=await prepare(application);await click('start');await wait(async()=>await text('verdict')===verdict,'trusted assessment '+application);
    const before=network.filter(r=>r.path.endsWith('/export')||r.path.includes('/evidence/')).length;
    await click('go-evidence');await wait(async()=>(await text('finding-summary')).startsWith('Metadata snapshot:'),'metadata-only finding view');
    check(network.filter(r=>r.path.endsWith('/export')||r.path.includes('/evidence/')).length===before,'ordinary finding view downloaded raw bodies');
    const snapshot=await operator('/v1/engagements/'+id+'/findings'),attempt=snapshot.assessments[0];
    check(attempt.result.verdict===verdict,'metadata invented verdict');
    check(await evaluate(`document.querySelectorAll('#evidence-checks tbody tr').length===7`),'selected prerequisite evidence missing');
    check(!(await evaluate(`document.body.textContent`)).includes('synthetic-private-proof-'),'ordinary metadata streamed proof body');
    check(await evaluate(`document.getElementById('capture-detail').hidden`),'raw body visible before explicit inspection');
    const expected=verdict==='confirmed'?1:0;check(snapshot.candidates.length===expected&&await evaluate(`document.querySelectorAll('#candidates [data-candidate]').length`)===expected,'candidate count diverged from validator');
    if(verdict==='confirmed'){
      const title=await text('finding-title');check(application==='browser-property'?title.includes('property'):title.includes('Cross-user'),'failure-specific title missing');
      check((await text('candidates')).includes('severity unrated'),'candidate acquired an invented rating');
      check((await text('finding-remediation')).includes(application==='browser-property'?'response property':'object read'),'wrong remediation profile');
      const proof=[];
      for(const role of ['owner_baseline','peer_probe']){
        const checkRef=attempt.checks.find(c=>c.role===role),reference=snapshot.captures.find(c=>c.id===checkRef.evidence_id);
        await evaluate(`document.querySelector('#evidence-checks tr[data-role="${role}"] button').click()`);
        await wait(async()=>(await text('raw-evidence-status')).includes('Byte integrity checked'),'explicit raw artifact hash check');
        const body=JSON.parse(await text('capture-detail'));check(body.capture.id===checkRef.capture_id&&body.run_id===attempt.run_id,'selected role opened another capture');
        proof.push(JSON.parse(body.capture.body).proof);
        await click('download-capture');const filename='evidence-'+reference.sha256+'.json';
        await wait(async()=>{try{const bytes=await readFile(path.join(directory,filename));return createHash('sha256').update(bytes).digest('hex')===reference.sha256;}catch{return false;}},'exact inspected artifact download');
        check(!(await text('api-detail')).includes('synthetic-private-proof-'),'API inspector repeated raw sensitive bodies');
      }
      check(proof[0]===proof[1]&&proof[0]==='synthetic-private-proof-owner','owner/probe proof correspondence failed');
      await click('load-evidence');await wait(async()=>(await text('evidence-summary')).startsWith('Snapshot:'),'explicit full export');
      await click('download-bundle');const filename='tsunami-'+id+'-bundle.json';
      await wait(async()=>{try{await readFile(path.join(directory,filename));return true;}catch{return false;}},'full bundle download');
      const exported=JSON.parse(await readFile(path.join(directory,filename),'utf8'));
      check(exported.finding_candidates[0].id===snapshot.candidates[0].id&&exported.finding_candidates[0].candidate_digest===snapshot.candidates[0].candidate_digest,'export candidate identity changed');
      for(const ref of snapshot.candidates[0].evidence_ids){const artifact=exported.artifacts.find(a=>a.id===ref);check(artifact&&createHash('sha256').update(Buffer.from(artifact.data_base64,'base64')).digest('hex')===artifact.sha256,'export evidence reference/hash failed');}
    }else{
      check((await text('finding-context')).includes(verdict),'non-finding verdict was lost');
      check(!(await text('finding-title')).includes('exposed'),'non-finding was titled as a leak');
    }
    summary.cases.push({application,engagement_id:id,run_id:attempt.run_id,verdict,candidates:expected,evidence_roles:attempt.checks.map(c=>({role:c.role,capture_id:c.capture_id,evidence_id:c.evidence_id,outcome:c.outcome}))});
  }
  summary.checks=['metadata-only finding navigation without automatic exports/raw reads','five real object/property controls','failure-specific titles and deterministic profile remediation','selected baseline/probe/recheck references from trusted validation','explicit owner/probe proof correspondence','single artifact byte-hash/download integrity','candidate/evidence identity matches full exports','no automatic review or severity','inconclusive and no-issue retained','ordinary metadata excludes body/proof values'];
}

async function retestJourney(){
  const source=await prepare('browser-vulnerable');await click('start');await wait(async()=>await text('verdict')==='confirmed','real vulnerable source');
  await click('go-evidence');await wait(async()=>(await text('finding-summary')).startsWith('Metadata snapshot:'),'source evidence metadata');
  // A stopped source remains valid historical evidence, never permission.
  await click('stop');await wait(async()=>(await text('stop-receipt')).includes('Stop acknowledged'),'source permission revoked');
  const original=await operator('/v1/engagements/'+source+'/export'),sourceRun=await runID(source);
  const starts=startCount();
  async function draft(application,name){
    const beforeStarts=startCount();
    await fill('retest-application',application);await evaluate(`document.getElementById('retest-application').dispatchEvent(new Event('change'))`);
    await fill('retest-name',name);await fill('retest-authorization','owned-local-fresh-retest');await fill('retest-executor','');
    check(await evaluate(`!document.getElementById('create-retest').disabled`),'retest selection unavailable');
    await click('create-retest');await wait(async()=>(await text('retest-plan')).includes('Comparison snapshot:')&&await text('engagement-state')==='draft','saved scope comparison');
    check(await evaluate(`!document.getElementById('scope-reviewed').checked&&document.getElementById('approve').disabled`),'retest inherited scope approval');
    check(startCount()===beforeStarts,'draft automatically started work');
    return evaluate(`document.querySelector('#scope-details dd').textContent`);
  }
  const id=await draft('browser-fixed','Owned linked fixed-app check');
  const draftView=await operator('/v1/engagements/'+id),initial=await operator('/v1/engagements/'+id+'/retest');
  check(draftView.engagement.source_engagement_id===source&&draftView.engagement.run_ids.length===0&&!draftView.engagement.approved_at,'draft link/permission incorrect');
  check(initial.source.engagement.target!==initial.retest.engagement.target&&initial.source.engagement.scope_digest!==initial.retest.engagement.scope_digest,'changed origin/scope not preserved');
  check(initial.compatibility==='comparable'&&initial.source.latest_attempt.result.verdict==='confirmed'&&!initial.retest.latest_attempt,'draft invented outcome');
  check(initial.differences.some(d=>d.field==='Origin')&&(await text('retest-plan')).includes('Origin'),'changed target not displayed');
  check((await operator('/test/observations')).requests===7,'draft/comparison sent target traffic');
  const denied=await fetch(origin+'/v1/engagements/'+id+'/runs',{method:'POST',headers:{Authorization:'Bearer '+token,'Content-Type':'application/json'},body:JSON.stringify({assessment:draftView.engagement.items[0].procedure,execution:'scripted'})});
  check(denied.status===409,'source approval authorized changed target');
  await evaluate(`document.getElementById('retest-plan').scrollIntoView()`);
  const scopeShot=await cdp('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});await writeFile(path.join(out,'retest-scope.png'),Buffer.from(scopeShot.data,'base64'),{mode:0o600});
  await evaluate(`document.getElementById('scope-reviewed').checked=true;document.getElementById('scope-reviewed').dispatchEvent(new Event('input'))`);
  await click('approve');await wait(async()=>await text('engagement-state')==='active','fresh retest approval');
  await click('go-run');await click('start');await wait(async()=>await text('verdict')==='no_issue_observed','real fixed result');
  await click('go-evidence');await wait(async()=>(await text('retest-comparison')).includes('no_issue_observed'),'completed linked comparison');
  const completed=await operator('/v1/engagements/'+id+'/retest'),fixedRun=await runID(id);
  check(completed.source.latest_attempt.run_id===sourceRun&&completed.retest.latest_attempt.run_id===fixedRun&&fixedRun!==sourceRun,'comparison runs incorrect');
  check(completed.source.latest_attempt.result.verdict==='confirmed'&&completed.retest.latest_attempt.result.verdict==='no_issue_observed','comparison verdicts incorrect');
  check(completed.source.latest_attempt.evidence_ids.length===7&&completed.retest.latest_attempt.evidence_ids.length===7,'comparison evidence missing');
  check(!(await text('retest-comparison')).includes('application is secure'),'unsupported security claim');
  const after=await operator('/v1/engagements/'+source+'/export');
  check(JSON.stringify(original.finding_candidates)===JSON.stringify(after.finding_candidates)&&JSON.stringify(original.artifacts)===JSON.stringify(after.artifacts),'retest overwrote original finding/evidence');
  check(network.every(r=>!r.path.endsWith('/export')&&!r.path.includes('/evidence/')),'normal linked navigation downloaded raw evidence');
  await evaluate(`[...document.querySelectorAll('#retest-comparison button')].find(b=>b.textContent==='Open source evidence').click()`);
  await wait(async()=>(await text('finding-context')).includes(sourceRun)&&(await text('finding-title')).includes('Cross-user'),'original finding navigation');
  const incompatible=await draft('browser-property-filtered','Different procedure control');
  check((await text('retest-plan')).includes('incompatible'),'different profile not labeled incompatible');
  await click('stop');await wait(async()=>(await text('stop-receipt')).includes('Stop acknowledged'),'incompatible draft revoked');
  await evaluate(`document.querySelector('[data-page="applications"]').click()`);await click('reload');
  await wait(()=>evaluate(`!!document.querySelector('[data-engagement="${id}"]')&&!document.getElementById('reload').disabled`),'durable linked history');
  check(await evaluate(`document.querySelector('[data-engagement="${id}"]').textContent.includes('Retest of assessment ${source}')`),'history omitted source link');
  await evaluate(`document.querySelector('[data-engagement="${id}"]').click()`);await wait(async()=>await text('verdict')==='no_issue_observed','recovered fixed run');
  await click('go-evidence');await wait(async()=>(await text('retest-comparison')).includes('no_issue_observed'),'recovered linked comparison');
  check((await operator('/test/observations')).requests===14&&startCount()===starts+1,'inspection/recovery replayed execution');
  summary.cases=[{source_engagement_id:source,source_run_id:sourceRun,retest_engagement_id:id,retest_run_id:fixedRun,source_verdict:'confirmed',retest_verdict:'no_issue_observed',compatibility:completed.compatibility,changed_fields:completed.differences.map(d=>d.field),source_evidence_ids:completed.source.latest_attempt.evidence_ids,retest_evidence_ids:completed.retest.latest_attempt.evidence_ids},{engagement_id:incompatible,case:'incompatible_profile_draft',state:'stopped',target_requests:0}];
  summary.checks=['two real native Mecatl attempts across distinct owned origins','stopped source remains history without granting authority','linked draft performs no target reads','changed scope shown and separate approval required','confirmed/no_issue_observed results and exact evidence references','original candidate/artifact bytes unchanged','metadata-only source/evidence/history navigation','incompatible profile labeled without inferred fix','history recovery preserves link without replay','narrow layout and independent Stop'];
}

try{
  if(externalProfile){
    check(externalProfile.startsWith(path.join(os.tmpdir(),'tsunami-ui-onboarding-')),'unexpected test browser profile');
    await readFile(path.join(externalProfile,'tsunami-test-profile'));profile=externalProfile;
  }else{
    profile=await mkdtemp(path.join(os.tmpdir(),'tsunami-d5-browser-'));
    chrome=spawn('google-chrome',['--headless=new','--disable-gpu','--no-proxy-server','--no-first-run','--no-default-browser-check','--disable-background-networking','--disable-component-update','--disable-default-apps','--disable-sync','--disable-extensions','--metrics-recording-only','--remote-debugging-address=127.0.0.1','--remote-debugging-port=0','--user-data-dir='+profile,'about:blank'],{stdio:'ignore',env:{...process.env,XDG_CONFIG_HOME:path.join(profile,'config')}});
  }
  let debugPort;await wait(async()=>{try{debugPort=(await readFile(path.join(profile,'DevToolsActivePort'),'utf8')).split('\n')[0];return !!debugPort;}catch{return false;}},'Chrome');
  const targets=await(await fetch('http://127.0.0.1:'+debugPort+'/json/list')).json();
  socket=new WebSocket(targets.find(t=>t.type==='page').webSocketDebuggerUrl);await new Promise((resolve,reject)=>{socket.onopen=resolve;socket.onerror=reject;});
  socket.onmessage=message=>{const d=JSON.parse(message.data);if(d.id){const p=pending.get(d.id);if(!p)return;pending.delete(d.id);clearTimeout(p.timer);if(d.error)p.reject(new Error('Browser protocol failure'));else p.resolve(d.result);}else if(d.method==='Runtime.exceptionThrown')exceptions.push('runtime exception');else if(d.method==='Network.requestWillBeSent'){const r=d.params.request,u=new URL(r.url);if(/^https?:$/.test(u.protocol))network.push({method:r.method,path:u.pathname,same_origin:u.origin===origin});}};
  await cdp('Runtime.enable');await cdp('Page.enable');await cdp('Network.enable');
  await cdp('Emulation.setDeviceMetricsOverride',{width:1360,height:1050,deviceScaleFactor:1,mobile:false});
  await cdp('Page.navigate',{url:origin+'/ui/'});await wait(()=>evaluate(`document.getElementById('stop-receipt')?.textContent.includes('Closing or pausing')`),'UI module ready');
  if(onboarding){await fill('operator-token','invalid-local-operator-token');await evaluate(`document.getElementById('connect-form').requestSubmit()`);await wait(async()=> (await text('message')).includes('401'),'invalid operator token');}
  await fill('operator-token',token);await evaluate(`document.getElementById('connect-form').requestSubmit()`);
  await wait(()=>evaluate(`document.getElementById('connection-state').textContent==='Operator connected'`),'operator connection');

  if(onboarding)await onboardingJourney();else if(findings)await findingsJourney();else if(retest)await retestJourney();else {
  for(const [application,verdict] of [['browser-vulnerable','confirmed'],['browser-fixed','no_issue_observed']]){
    const id=await prepare(application),before=startCount();
    if(application==='browser-fixed')await evaluate(`window.realFetch=window.fetch;window.fetch=async(...args)=>{const r=await window.realFetch(...args);if(args[0]==='/v1/engagements/${id}/runs'&&r.ok)throw new TypeError('lost committed Start response');return r;}`);
    await click('start');
    if(application==='browser-fixed'){
      await wait(async()=> (await text('message')).startsWith('Start not confirmed'),'lost Start');
      await evaluate(`window.fetch=window.realFetch;delete window.realFetch`);await click('refresh-run');
      await wait(()=>evaluate(`!document.getElementById('refresh-run').disabled`),'read-only Start recovery');await click('pause');
    }
    await wait(async()=>await text('verdict')===verdict,'trusted result');
    await wait(()=>evaluate(`!!document.querySelector('#activity-events [data-kind="validation.completed"]')`),'persisted validator event');
    check(await evaluate(`document.querySelectorAll('#activity-events [data-kind="request.rejected"]').length===1`),'missing real scope rejection');
    check((await text('activity-events')).includes('scope_not_approved'),'scope rejection not explained');
    check(startCount()===before+1,'Start was retried');
    const raw=await operator('/v1/runs/'+await runID(id)+'/events?after=0&limit=100');
    check(await evaluate(`JSON.stringify([...document.querySelectorAll('#activity-events [data-sequence]')].map(n=>Number(n.dataset.sequence)))`)===JSON.stringify(raw.items.map(e=>e.sequence)),'UI event sequence differs from stored journal');
    summary.cases.push({application,engagement_id:id,run_id:await runID(id),verdict,event_sequences:raw.items.map(e=>e.sequence)});
  }

  for(const failure of ['held_export','held_start','lost_stop_response']){
    const id=await prepare('browser-held'),before=startCount();
    if(failure==='held_start')await evaluate(`window.realFetch=window.fetch;window.fetch=async(...args)=>{const r=await window.realFetch(...args);if(args[0]==='/v1/engagements/${id}/runs')await new Promise(resolve=>window.releaseHeldStart=resolve);return r;}`);
    await click('start');await heldRequest(id);
    if(failure==='held_export'){
      await wait(()=>evaluate(`document.querySelectorAll('#activity-events [data-kind="request.sent"]').length===5`),'actual active activity');
      await evaluate(`document.querySelector('[data-page="evidence"]').click();window.realFetch=window.fetch;window.fetch=async(...args)=>{const r=await window.realFetch(...args);if(args[0]==='/v1/engagements/${id}/export')await new Promise(resolve=>window.releaseHeldExport=resolve);return r;}`);
      await click('load-evidence');await wait(()=>evaluate(`!!window.releaseHeldExport`),'pending real export');
      check(await evaluate(`!document.getElementById('stop').disabled && !document.getElementById('assessment-safety').hidden && document.getElementById('load-evidence').disabled`),'pending export hid/disabled Stop');
    }
    if(failure==='lost_stop_response')await evaluate(`window.realFetch=window.fetch;window.fetch=async(...args)=>{const r=await window.realFetch(...args);if(args[0]==='/v1/engagements/${id}/stop'&&r.ok)throw new TypeError('lost committed Stop response');return r;}`);
    const stopPosts=network.filter(r=>r.method==='POST'&&r.path.endsWith('/stop')).length;
    await click('stop');
    await wait(async()=> (await text('stop-receipt')).includes(failure==='lost_stop_response'?'NOT acknowledged':'Stop acknowledged'),'separate Stop receipt',4000);
    if(failure==='held_start')await evaluate(`window.releaseHeldStart();window.fetch=window.realFetch;delete window.realFetch`);
    if(failure==='held_export'){
      await evaluate(`window.releaseHeldExport();window.fetch=window.realFetch;delete window.realFetch;document.querySelector('[data-page="run"]').click()`);
      await wait(()=>evaluate(`!document.getElementById('refresh-run').disabled`),'export release');await click('refresh-run');
    }
    if(failure==='lost_stop_response'){
      await wait(async()=> (await text('stop-receipt')).includes('Revocation separately observed'),'lost Stop recovered through stored state');
      await evaluate(`window.fetch=window.realFetch;delete window.realFetch`);
    }
    await drain();
    await wait(()=>evaluate(`!!document.querySelector('#activity-events [data-kind="permission.revoked"]') && !!document.querySelector('#activity-events [data-kind="activity.drained_locally"]')`),'actual revocation/drain events');
    check(startCount()===before+1,'pending/lost acknowledgement retried Start');
    check(network.filter(r=>r.method==='POST'&&r.path.endsWith('/stop')).length===stopPosts+1,'Stop automatically retried');
    check(await evaluate(`!document.querySelector('#activity-events [data-kind="execution.exit_observed"]') && !document.querySelector('#activity-events [data-kind="validation.completed"]')`),'Stop invented validation/exit');
    summary.cases.push({application:'browser-held',engagement_id:id,run_id:await runID(id),verdict:'stopped',failure_injection:failure});
  }
  }
  await evaluate(`document.getElementById(${JSON.stringify(retest?'retest-comparison':'activity-heading')}).scrollIntoView()`);
  const wide=await cdp('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});await writeFile(path.join(out,retest?'retest-desktop.png':findings?'finding-desktop.png':'activity-desktop.png'),Buffer.from(wide.data,'base64'),{mode:0o600});
  await cdp('Emulation.setDeviceMetricsOverride',{width:390,height:844,deviceScaleFactor:1,mobile:false});
  check(await evaluate(`document.documentElement.scrollWidth<=innerWidth`),'narrow layout overflow');
  check(await evaluate(`!document.getElementById('stop').disabled`),'Stop unavailable at narrow width');
  check(await evaluate(`!document.body.textContent.includes('synthetic-private-proof-')&&!document.body.textContent.includes(${JSON.stringify(token)})&&!document.body.textContent.includes(${JSON.stringify('a'.repeat(48))})`),'ordinary UI streamed raw proof or credentials');
  const shot=await cdp('Page.captureScreenshot',{format:'png',captureBeyondViewport:false});await writeFile(path.join(out,retest?'retest-mobile.png':findings?'finding-mobile.png':'activity-stop.png'),Buffer.from(shot.data,'base64'),{mode:0o600});
  const postsBefore=network.filter(r=>r.method==='POST').length;
  await click('disconnect');check((await text('message')).includes('did not Stop'),'disconnect implication incorrect');
  check(network.filter(r=>r.method==='POST').length===postsBefore,'disconnect sent mutation');
  check(exceptions.length===0,'browser runtime exception');check(network.every(r=>r.same_origin),'browser made target/external request');
  if(!onboarding&&!findings&&!retest)summary.checks=['real persisted rejection/validation sequence matches UI','lost Start recovered without retry','Stop during real held target request and pending export/Start','lost Stop response distinguished from observed durable revocation','no invented worker exit or completed validation on Stop','narrow layout','ordinary UI excludes raw proofs/credentials','same-origin browser requests','disconnect sends no Stop'];
  summary.passed=true;
  const source=await readFile('internal/webui/static/app.js');summary.app_js_sha256=createHash('sha256').update(source).digest('hex');
}catch(error){summary.error=error.message;if(socket?.readyState===WebSocket.OPEN){try{summary.failure_snapshot=await evaluate(`({message:document.getElementById('message')?.textContent,receipt:document.getElementById('stop-receipt')?.textContent,permission:document.getElementById('status-permission')?.textContent})`);}catch{}}console.error('FAIL local activity browser: '+error.message);process.exitCode=1;}
finally{
  try{
    if(socket)socket.close();for(const p of pending.values()){clearTimeout(p.timer);p.reject(new Error('browser closing'));}
    if(chrome&&chrome.exitCode===null){const exited=once(chrome,'exit');chrome.kill('SIGTERM');await exited;}
    if(profile&&!externalProfile)await rm(profile,{recursive:true,force:true,maxRetries:10,retryDelay:100});
  }catch{summary.passed=false;summary.cleanup_error='Chrome/profile cleanup was not confirmed';process.exitCode=1;console.error('FAIL local browser cleanup');}
  await writeFile(path.join(out,'summary.json'),JSON.stringify(summary,null,2)+'\n',{mode:0o600});
  if(summary.passed){console.log(onboarding?'PASS local Chrome onboarding: four approved prerequisite reads, explicit scope approval, one real Mecatl assessment; negative/stale/injection/history checks passed.':findings?'PASS local Chrome findings: five real object/property controls; metadata navigation, selected raw evidence and exact artifact/export hashes verified.':retest?'PASS local Chrome retest: changed-origin draft, fresh approval, real confirmed/no-issue comparison, original evidence preservation, incompatible draft and history recovery.':'PASS local Chrome: five real Mecatl attempts; vulnerable/fixed results, rejected scope escape and three active Stop cases.');console.log('Private local record: '+path.relative(process.cwd(),out));}
}
