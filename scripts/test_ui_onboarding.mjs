// D6 rendering and review guards against the actual UI modules. All inputs here
// are isolated test fixtures; connected-browser tests use the real backend.
import {test} from 'node:test';
import assert from 'node:assert/strict';
import {ui} from './ui-test-support.mjs';
const flush=()=>new Promise(resolve=>setImmediate(resolve));
const response=data=>new Response(JSON.stringify(data));

test('Dashboard separates registered access, scope state, stored result and missing attempt metadata',()=>{
  const p=ui();p.run(`state.apps=[{...state.app,title:'Owned <img src=x>',metadata:{registration:'startup',procedure:{supported:true},access:{availability:'configured'},observed_at:'2030-01-01T00:00:00Z'}}];
    state.engagements=[{...state.view.engagement,name:'First',latest_attempt_status:'recorded',latest_attempt:{state:'completed',verdict:'confirmed',created_at:'2030-01-01T00:00:00Z'}},{...state.view.engagement,name:'Second',latest_attempt_status:'unavailable'}];renderDashboard()`);
  assert.match(p.text('application-list'),/Reviewed lab integration.*Owned <img src=x>/);
  assert.match(p.text('application-list'),/Scope review required.*current session health unknown/);
  assert.match(p.text('engagement-list'),/Scope: active.*Latest attempt: completed · confirmed/);
  assert.match(p.text('engagement-list'),/Latest attempt metadata unavailable/);
  assert.doesNotMatch(p.text('engagement-list'),/Scope: completed/);
});

test('readable scope contains exact actions and limits; checking review binds approval without digest typing',async()=>{
  const p=ui();p.run("state.token='local-test-operator-token';state.view.engagement.state='draft';state.view.metadata.permission='unapproved';renderPlan()");
  assert.equal(p.rows('scope-summary').Target,'https://owned.example.test');
  assert.match(p.rows('scope-summary')['Intended rule'],/must not receive/);
  assert.match(p.rows('scope-summary').Limits,/3 target reservations/);
  const body=p.nodes.get('scope-actions').children[0].children[1];
  assert.equal(body.children.length,7);
  assert.match(body.children[4].textContent,/peerGET \/objects\/aCheck cross-user object access/);
  assert.equal(p.run("$('reviewed-digest').value"),'c'.repeat(64));
  p.run("$('scope-reviewed').checked=true");await p.click('scope-reviewed','input');
  assert.equal(p.run("$('approve').disabled"),false);
  p.run("state.view.engagement.scope_digest='f'.repeat(64);renderPlan()");
  assert.equal(p.run("$('scope-reviewed').checked"),false);
  assert.equal(p.run("$('approve').disabled"),true);
  assert.equal(p.run('state.reviewedScope'),'');
});

test('unsupported procedures keep approval and Start disabled while scope remains inspectable',async()=>{
  const p=ui();p.run("state.token='local-test-operator-token';state.view.engagement.state='draft';state.view.metadata.permission='unapproved';state.view.metadata.procedures[0].supported=false;renderPlan();$('scope-reviewed').checked=true");
  await p.click('scope-reviewed','input');
  assert.equal(p.run("$('approve').disabled"),true);assert.equal(p.run("$('start').disabled"),true);
  assert.match(p.text('scope-support'),/unsupported.*Approval and Start are disabled/);
  assert.match(p.rows('scope-summary').Limits,/not available/);
});

test('approval submits the digest bound to explicit review and does not launch work',async()=>{
  const p=ui(), calls=[];
  p.run("state.token='local-test-operator-token';state.view.engagement.state='draft';state.view.metadata.permission='unapproved';renderPlan()");
  const approved=structuredClone(p.view);approved.engagement.state='active';approved.metadata.permission='active';
  p.setFetch(async(path,options)=>{calls.push({path,body:JSON.parse(options.body)});return response(approved);});
  p.run("$('scope-reviewed').checked=true");await p.click('scope-reviewed','input');
  await p.click('approval-form','submit');await flush();await flush();
  assert.equal(calls.length,1);assert.match(calls[0].path,/\/approve$/);
  assert.deepEqual(calls[0].body,{scope_digest:'c'.repeat(64)});
  assert.equal(p.run('state.view.engagement.state'),'active');
});

function setup(){return {id:'e'.repeat(32),revision:1,state:'ready_for_validation',expires_at:new Date(Date.now()+3600000).toISOString(),created_at:'2030-01-01T00:00:00Z',access_ready:true,validation_attempts:0,validation_requests:0,preflight_digest:'a'.repeat(64),messages:[],
  details:{name:'Owned setup',environment:'dev',frontend_url:'https://front.example.test',api_origin:'https://owned.example.test',approved_addresses:['192.0.2.1'],auth_method:'bearer',identity_path:'/me',owner_path:'/records/a',peer_path:'/records/b',expected_access:'deny'}};}
async function loadedSetup(){
  const p=ui({onboarding:true});let stored=setup();const calls=[];
  p.setFetch(async(path,o)=>{calls.push({path,method:o.method});return response(stored);});
  p.run("state.token='local-test-operator-token';controls()");
  p.run(`setupUI.lists([${JSON.stringify(stored)}],[]);$('setup-history').children[0].listeners.click[0]()`);await flush();await flush();
  return {...p,calls,setStored(value){stored=value;}};
}

test('preflight shows exactly four legitimate reads and resets permission on a changed revision',async()=>{
  const p=await loadedSetup();
  const body=p.nodes.get('setup-preflight-actions').children[0].children[1];
  assert.equal(body.children.length,4);assert.match(body.children[3].textContent,/peerGET \/records\/b/);
  assert.doesNotMatch(body.textContent,/cross-user/);
  p.run("$('setup-preflight-approved').checked=true");await p.click('setup-preflight-approved','input');
  assert.equal(p.run("$('validate-setup').disabled"),false);
  p.setStored({...setup(),revision:2,preflight_digest:'b'.repeat(64)});await p.click('refresh-setup');await flush();await flush();
  assert.equal(p.run("$('setup-preflight-approved').checked"),false);
  assert.equal(p.run("$('validate-setup').disabled"),true);
  p.setStored(setup());await p.click('refresh-setup');await flush();await flush();
  assert.match(p.rows('setup-facts').Setup,/revision 2/);
  assert.equal(p.calls.filter(c=>c.method==='POST').length,0);
});

test('unsupported authentication stays unavailable and lab reuse explains its fixed API target',async()=>{
  const p=await loadedSetup();
  p.setStored({...setup(),revision:2,state:'unsupported',access_ready:false,details:{...setup().details,auth_method:'sso'},question:{field:'unsupported_auth',text:'No SSO adapter is available.'}});
  await p.click('refresh-setup');await flush();await flush();
  assert.match(p.text('setup-next'),/authentication adapter is unavailable/);
  assert.equal(p.run("$('validate-setup').disabled"),true);assert.equal(p.run("$('prepare-setup').disabled"),true);
  p.run(`setupUI.lists([], [{...state.app,metadata:{registration:'startup'}}]);setupUI.start();$('setup-source').value='owned-api'`);await p.click('setup-source','change');
  assert.match(p.text('setup-source-details'),/https:\/\/owned.example.test.*does not change this API target/s);
  assert.equal(p.run("$('setup-extra').hidden"),true);
  assert.equal(p.calls.filter(c=>c.method==='POST').length,0);
});
