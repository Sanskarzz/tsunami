// Concurrency/failure injections against the actual frontend code. These isolated
// fixtures are not product events, findings, or a deployment qualification.
import {test} from 'node:test';
import assert from 'node:assert/strict';
import {ui} from './ui-test-support.mjs';

const runID = 'd'.repeat(32), engagementID = 'a'.repeat(32);
const flush = () => new Promise(resolve => setImmediate(resolve));
const response = (data, status = 200) => new Response(JSON.stringify(data), {status});
function deferred() { let resolve, reject; const promise = new Promise((a,b) => {resolve=a;reject=b;}); return {promise,resolve,reject}; }
function event(sequence, kind = 'request.admitted', data = {}) {
  const source=kind==='validation.completed'?'validator':kind==='permission.revoked'?'control_plane':kind.startsWith('cleanup.')?'controller':'broker';
  return {schema_version:1, sequence, recorded_at:'2030-01-01T00:00:01Z', source, engagement_id:engagementID, run_id:runID, kind, data};
}
function page(items = [], options = {}) {
  const latest = items.at(-1)?.sequence || 0;
  return {items,next_cursor:latest,oldest_sequence:latest ? 1 : 0,latest_sequence:latest,has_more:false,history_truncated:false,gap:false,tracked_from_creation:true,...options};
}
function client() {
  let timerID=0;
  const timers=new Map(), calls=[];
  const p=ui({setTimeout(fn,ms){const id=++timerID;timers.set(id,{fn,ms});return id;},clearTimeout(id){timers.delete(id);}});
  p.run("state.token='local-test-operator-token';state.currentPage='run';resetActivity();renderPlan();renderStatus()");
  let events=page(), storedView=structuredClone(p.view), status=structuredClone(p.status);
  const route=async (path,options)=>{
    calls.push({path,method:options.method,signal:options.signal});
    if(path.endsWith('/stop')) {storedView.engagement.state='stopped';storedView.metadata.permission='revoked';status.permission='revoked';status.activity='drained_locally';return response({stopped:[runID]});}
    if(path.endsWith('/status'))return response(status);
    if(path.includes('/events?'))return response(events);
    if(path===`/v1/engagements/${engagementID}`)return response(storedView);
    throw new Error('unexpected test route');
  };
  p.setFetch(route);
  return {...p,calls,timers,route,setEvents(v){events=v;},storedView,status};
}

test('cursor pages render actual rejection and validation without raw payloads or duplicate rows',async()=>{
  const p=client();let reads=0;
  const rows=[event(1,'request.admitted',{read_index:5,account_slot:2,reservation:5}),event(2,'request.rejected',{stage:'scope',code:'scope_not_approved',url:'secret-raw-url-canary',body:'secret-body-canary'}),
    event(3,'validation.completed',{verdict:'confirmed',validation:{required:6,verified:6,probe_captured:true}})];
  p.setFetch(async(path,o)=>{if(!path.includes('/events?'))return p.route(path,o);reads++;return response(reads===1?page(rows.slice(0,2),{has_more:true,latest_sequence:3}):reads===2?page(rows.slice(2)):page([],{next_cursor:3,oldest_sequence:1,latest_sequence:3}));});
  await p.run('refresh()');await p.run('refresh()');
  assert.equal(reads,3);
  assert.equal(p.nodes.get('activity-events').children.length,3);
  assert.match(p.text('activity-events'),/Admission is not target receipt/);
  assert.match(p.text('activity-events'),/REJECTED by broker.*scope_not_approved/);
  assert.match(p.text('activity-events'),/Trusted validation: confirmed.*6\/6/);
  assert.equal(p.text('activity-rejections'),'Scope rejections in displayed records: 1');
  assert.doesNotMatch(p.text('activity-events'),/secret-|undefined/);
  assert.equal(p.run('state.activity.cursor'),3);
});

test('retention gaps, partial historical tracking and local display bounds stay visible',async()=>{
  const p=client();
  p.setEvents(page([event(9),event(10)],{oldest_sequence:9,history_truncated:true,gap:true,tracked_from_creation:false}));
  await p.run('refresh()');
  assert.match(p.text('activity-notice'),/Tracking did not start.*retention removed.*Cursor gap/s);
  p.setEvents(page([],{next_cursor:10,oldest_sequence:9,latest_sequence:10,history_truncated:true,tracked_from_creation:false}));
  await p.run('refresh()');
  assert.match(p.text('activity-notice'),/Cursor gap/);
  p.run('state.activity.items=Array.from({length:256},(_,i)=>({...state.activity.items[0],sequence:i+1}));state.activity.cursor=256');
  p.setEvents(page([event(257)],{oldest_sequence:2,history_truncated:true}));
  await p.run('refresh()');
  assert.equal(p.nodes.get('activity-events').children.length,256);
  assert.match(p.text('activity-notice'),/Only the latest 256/);
});

test('missing history and incompatible pages never become a clean or fabricated timeline',async()=>{
  const p=client();p.setEvents(page([],{tracked_from_creation:false}));await p.run('refresh()');
  assert.match(p.text('activity-events'),/Missing events are not proof/);
  for(const bad of [page([event(2)]),page([event(1)],{next_cursor:2}),page([{...event(1),run_id:'e'.repeat(32)}]),page([{...event(1),schema_version:2}]),page([event(1,'request.capture_persisted',{evidence_ids:'incompatible'})]),page([{...event(1,'validation.completed'),source:'broker'}])]){
    p.setEvents(bad);await assert.rejects(p.run('refresh()'),/Activity|activity/);
    assert.equal(p.run('state.activity.cursor'),0);
    assert.equal(p.nodes.get('activity-events').children.length,0);
    assert.match(p.text('activity-observed-at'),/STALE/);
    assert.match(p.text('activity-notice'),/inconsistent|cursor|invalid activity page|incompatible/i);
  }
});

test('manual refresh and repeated Resume share one observation with one scheduled loop',async()=>{
  const p=client(), held=deferred();let reads=0;
  p.status.cleanup='pending';p.status.state='running';
  p.setFetch(async(path,o)=>{if(path===`/v1/engagements/${engagementID}`){reads++;return held.promise;}return p.route(path,o);});
  const a=p.run('refresh()'),b=p.run('refresh()');assert.equal(a,b);assert.equal(reads,1);
  held.resolve(response(p.storedView));await Promise.all([a,b]);
  p.setFetch(p.route);p.run('observe()');await flush();await flush();
  assert.equal([...p.timers.values()].filter(t=>t.ms===1500).length,1);
  const tick=[...p.timers.values()].find(t=>t.ms===1500);p.timers.clear();await tick.fn();
  assert.equal([...p.timers.values()].filter(t=>t.ms===1500).length,1);
  p.run('stopObservation()');assert.equal(p.timers.size,0);
});

test('Stop cancels a held observation and old successful responses cannot overwrite revocation',async()=>{
  const p=client(), held=deferred();let first=true, heldSignal;
  p.setFetch(async(path,o)=>{if(path.endsWith('/status')&&first){first=false;heldSignal=o.signal;return held.promise;}return p.route(path,o);});
  const old=p.run('refresh()');await flush();
  assert.equal(p.run("$('stop').disabled"),false);
  await p.run('stopAssessment()');await flush();await flush();
  assert.match(p.text('stop-receipt'),/Stop acknowledged/);
  assert.equal(p.run('state.stopping'),false);
  assert.equal(p.text('status-permission'),'revoked');
  held.resolve(response({...p.status,permission:'active'}));await old;
  assert.equal(p.text('status-permission'),'revoked');
  assert.equal(p.calls.filter(c=>c.method==='POST').length,1);
  assert.equal(heldSignal.aborted,true);
  p.run('stopObservation()');
});

test('Stop receipt is displayed immediately even when its follow-up read never resolves',async()=>{
  const p=client(), held=deferred();p.setFetch(async(path,o)=>path===`/v1/engagements/${engagementID}`?held.promise:p.route(path,o));
  await p.run('stopAssessment()');
  assert.equal(p.run('state.stopping'),false);
  assert.match(p.text('stop-receipt'),/Stop acknowledged.*separate observations/);
  assert.equal(p.run("$('stop').disabled"),false);
  p.run('stopObservation()');held.resolve(response(p.storedView));await flush();
});

test('unacknowledged Stop stays distinct from separately observed durable revocation',async()=>{
  const p=client();let posts=0;
  p.setFetch(async(path,o)=>{if(path.endsWith('/stop')){posts++;await p.route(path,o);throw new TypeError('lost committed Stop response');}return p.route(path,o);});
  await p.run('stopAssessment()');await flush();await flush();
  assert.match(p.text('stop-receipt'),/NOT acknowledged.*Revocation separately observed/s);
  assert.equal(p.run('state.stopReceipt.phase'),'unacknowledged');
  assert.equal(p.text('status-permission'),'revoked');
  assert.equal(posts,1);assert.equal(p.run("$('start').disabled"),true);
  p.run('stopObservation()');
});

test('failed Stop and failed follow-up remain uncertain without retry or invented drain',async()=>{
  const p=client();let posts=0;
  p.setFetch(async(path,o)=>{if(path.endsWith('/stop')){posts++;throw new TypeError('unavailable');}throw new TypeError('read unavailable');});
  await p.run('stopAssessment()');await flush();
  assert.match(p.text('stop-receipt'),/NOT acknowledged/);
  assert.doesNotMatch(p.text('stop-receipt'),/Revocation separately observed/);
  assert.equal(p.run('state.stopReceipt.observedAt'),undefined);
  assert.equal(posts,1);assert.equal(p.run("$('stop').disabled"),false);
  assert.match(p.text('observed-at'),/STALE/);
});

test('status and feed fail independently, retain their last values and recover with the same cursor',async()=>{
  const p=client();p.setEvents(page([event(1)]));await p.run('refresh()');
  p.setFetch(async(path,o)=>path.includes('/events?')?response({error:'activity unavailable'},503):p.route(path,o));
  await assert.rejects(p.run('refresh()'),/503/);
  assert.match(p.text('activity-observed-at'),/STALE/);assert.doesNotMatch(p.text('observed-at'),/STALE/);
  assert.equal(p.run('state.activity.cursor'),1);assert.equal(p.nodes.get('activity-events').children.length,1);
  p.setEvents(page([event(2,'permission.revoked',{code:'run_stop'})]));
  p.setFetch(async(path,o)=>path.endsWith('/status')?response({error:'status unavailable'},503):p.route(path,o));
  await assert.rejects(p.run('refresh()'),/503/);
  assert.match(p.text('observed-at'),/STALE/);assert.doesNotMatch(p.text('activity-observed-at'),/STALE/);
  assert.equal(p.run('state.activity.cursor'),2);assert.match(p.text('activity-events'),/Permission durably revoked/);
});

test('Stop remains available during an export and a pending Start, and late Start cannot erase its receipt',async()=>{
  for(const operation of ['export','start']){
    const p=client(),held=deferred();
    if(operation==='start')p.run("state.run='';state.status=null;state.view.engagement.run_ids=[];controls()");
    p.setFetch(async(path,o)=>{
      if((operation==='export'&&path.endsWith('/export'))||(operation==='start'&&path.endsWith('/runs')&&o.method==='POST'))return held.promise;
      return p.route(path,o);
    });
    await p.click(operation==='export'?'load-evidence':'start');await flush();
    assert.equal(p.run('state.busy'),true);assert.equal(p.run("$('stop').disabled"),false);
    await p.run('stopAssessment()');await flush();
    assert.match(p.text('stop-receipt'),/Stop acknowledged/);
    held.resolve(response(operation==='start'?{id:runID}:{finding_candidates:[],artifacts:[],report_markdown:'Stored evidence snapshot'}));
    await flush();await flush();
    assert.match(p.text('stop-receipt'),/Stop acknowledged/);
    assert.equal(p.run('state.busy'),false);
    assert.equal(p.run("$('start').disabled"),true);
    p.run('stopObservation()');
  }
});

test('lost committed Start is recovered by reads without a second Start',async()=>{
  const p=client();let starts=0;
  p.run("state.run='';state.view.engagement.run_ids=[];controls()");
  p.setFetch(async(path,o)=>{if(path.endsWith('/runs')&&o.method==='POST'){starts++;throw new TypeError('lost response');}return p.route(path,o);});
  await p.click('start');await flush();
  assert.match(p.text('message'),/Start not confirmed/);
  assert.equal(p.run("$('start').disabled"),false);assert.equal(p.text('start'),'Recover Start');
  await p.run('refresh()');
  assert.equal(p.run('state.run'),runID);
  await p.click('start');await flush();
  assert.equal(starts,1);
});

test('explicit Recover Start reuses the exact original action and never retries automatically',async()=>{
  const p=client(), requests=[];
  p.run("state.run='';state.view.engagement.run_ids=[];controls()");
  p.setFetch(async(path,o)=>{
    if(path.endsWith('/runs')&&o.method==='POST'){
      requests.push(JSON.parse(o.body));
      if(requests.length===1)throw new TypeError('lost committed response');
      return response({id:runID});
    }
    return p.route(path,o);
  });
  await p.click('start');await flush();await flush();
  assert.equal(requests.length,1);assert.match(requests[0].idempotency_key,/^[a-f0-9]{32}$/);
  assert.equal(p.text('start'),'Recover Start');assert.equal(p.run("$('start').disabled"),false);
  await p.click('start');await flush();await flush();
  assert.equal(requests.length,2);assert.deepEqual(requests[1],requests[0]);
  assert.equal(p.run('state.run'),runID);assert.equal(p.run("$('start').disabled"),true);
  p.run('stopObservation()');
});

test('navigation, Pause and disconnect cancel viewer reads without sending Stop or leaking late data',async()=>{
  for(const action of ["page('evidence')",'stopObservation()','disconnect()']){
    const p=client(),held=deferred();let scopeRead;
    p.setFetch(async(path,o)=>{if(path===`/v1/engagements/${engagementID}`){scopeRead=o;return held.promise;}return p.route(path,o);});
    p.run('observe()');await flush();p.run(action);
    assert.equal(scopeRead.signal.aborted,true);
    held.resolve(response(p.storedView));await flush();await flush();
    assert.equal(p.run('state.polling'),false);assert.equal(p.calls.filter(c=>c.method==='POST').length,0);
    if(action==='disconnect()'){assert.equal(p.run('state.view'),null);assert.equal(p.run('state.token'),'');assert.equal(p.nodes.get('activity-events').children.length,0);}
  }
});

test('restart uncertainty and API object absence never become exit or cancellation proof',async()=>{
  const p=client();Object.assign(p.status,{activity:'uncertain_prior_or_unpersisted_completion',uncertain_operations:1,execution:'unknown_after_resource_removal',cleanup:'api_resources_absent',exit_observed_at:null});
  p.setEvents(page([event(1,'cleanup.resources_absent')]));p.run('observe()');await flush();await flush();
  assert.equal(p.text('status-execution'),'unknown_after_resource_removal');
  assert.equal(p.rows('lifecycle-observations')['Uncertain operations'],'1');
  assert.equal(p.rows('run-policy')['Exit observed'],'Not recorded');
  assert.match(p.text('activity-events'),/not process-exit proof/);
  assert.equal(p.run('state.polling'),true);
  p.run('stopObservation()');
});
