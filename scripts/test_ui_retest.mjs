// Isolated presentation/concurrency fixtures. Real execution is covered by the
// opt-in browser campaign; these records never enter product data.
import {test} from 'node:test';
import assert from 'node:assert/strict';
import {ui} from './ui-test-support.mjs';
const source='e'.repeat(32),destination='a'.repeat(32),run='d'.repeat(32);
const response=data=>new Response(JSON.stringify(data));
const flush=()=>new Promise(resolve=>setImmediate(resolve));
function deferred(){let resolve;const promise=new Promise(r=>resolve=r);return {promise,resolve};}
function fixture(){
  const p=ui();p.view.engagement.source_engagement_id=source;p.view.engagement.state='draft';p.view.engagement.run_ids=[];p.view.metadata.permission='unapproved';
  const src=structuredClone(p.view.engagement);src.id=source;delete src.source_engagement_id;src.target='https://old-owned.example.test';src.state='stopped';src.scope_digest='f'.repeat(64);
  const comparison={schema_version:1,observed_at:'2030-01-01T00:00:02Z',compatibility:'comparable',reason:'Same configured check; separate bindings and approvals.',
    source:{engagement:src,procedure:p.view.metadata.procedures[0],attempt_status:'recorded',latest_attempt:{run_id:'9'.repeat(32),state:'completed',stored_verdict:'confirmed',result:{verdict:'confirmed',reason:'Protected proof returned.'},evidence_ids:['E-'+'1'.repeat(64)]}},
    retest:{engagement:p.view.engagement,procedure:p.view.metadata.procedures[0],attempt_status:'not_started'},
    differences:[{field:'Origin',source:JSON.stringify(src.target),retest:JSON.stringify(p.view.engagement.target)},{field:'Scope digest',source:src.scope_digest,retest:p.view.engagement.scope_digest}]};
  p.run(`state.token='local-test-operator-token';state.view=${JSON.stringify(p.view)};state.run='';state.status=null;state.currentPage='plan';`);
  const calls=[];
  p.setFetch(async(path,options)=>{calls.push({path,method:options.method,body:options.body});if(path.endsWith('/retest'))return response(comparison);throw new Error('test route unavailable');});
  return {...p,comparison,calls};
}

test('linked draft gates approval on actual scope comparison and requires a fresh review',async()=>{
  const p=fixture();p.run("renderPlan();$('scope-reviewed').checked=true;state.reviewedScope=state.view.engagement.scope_digest;controls()");
  assert.equal(p.run("$('approve').disabled"),true);assert.equal(p.run("$('stop').disabled"),false);
  await p.run('loadRetestComparison()');
  assert.match(p.text('retest-plan'),/Source.*old-owned\.example\.test.*confirmed.*Retest.*No completed validation/s);
  assert.match(p.text('retest-plan'),/Changed scope and bindings.*Origin.*Scope digest/s);
  assert.equal(p.run("$('approve').disabled"),false);
  p.run("state.view.engagement.scope_digest='7'.repeat(64);renderPlan()");
  assert.equal(p.run("$('scope-reviewed').checked"),false);assert.equal(p.run("$('approve').disabled"),true);
  assert.equal(p.calls.length,1);assert.equal(p.calls[0].method,'GET');
});

test('comparison preserves no-issue, incompatible, inconclusive and stopped outcomes without claiming a fix',async()=>{
  const p=fixture();await p.run('loadRetestComparison()');
  p.comparison.retest.latest_attempt={run_id:run,state:'completed',stored_verdict:'no_issue_observed',result:{verdict:'no_issue_observed',reason:'Configured probe denied.'},evidence_ids:['E-'+'2'.repeat(64)]};
  await p.run('loadRetestComparison()');
  assert.match(p.text('retest-comparison'),/confirmed.*no_issue_observed/s);assert.doesNotMatch(p.text('retest-comparison'),/issue fixed|application is secure/i);
  p.comparison.compatibility='incompatible';p.comparison.reason='Different configured access rule. Do not infer a fix.';
  p.comparison.retest.latest_attempt.result={verdict:'inconclusive',reason:'Missing recheck.'};
  await p.run('loadRetestComparison()');assert.match(p.text('retest-comparison'),/incompatible.*inconclusive/s);
  p.comparison.retest.latest_attempt={run_id:run,state:'stopped',evidence_ids:[]};
  await p.run('loadRetestComparison()');assert.match(p.text('retest-comparison'),/stopped.*No completed validation/s);
  assert.doesNotMatch(p.text('retest-comparison'),/no_issue_observed/);
});

test('retest draft submits only explicit destination selection and source ID without inherited approval or credentials',async()=>{
  const p=fixture();delete p.view.engagement.source_engagement_id;
  p.run(`state.view=${JSON.stringify(p.view)};state.apps=[{...state.app,metadata:{procedure:{supported:true}}}];renderRetestChooser();$('retest-application').value=state.app.id;$('retest-name').value='Fresh test';$('retest-authorization').value='new-permission';$('retest-executor').value='';`);
  let submitted;
  p.setFetch(async(path,options)=>{submitted=JSON.parse(options.body);return response({id:'b'.repeat(32)});});
  p.run('openEngagement=async id=>{state.opened=id}');
  await p.click('retest-form','submit');await flush();
  assert.equal(submitted.source_engagement_id,destination);assert.equal(submitted.application,'owned-api');assert.equal(submitted.authorization_ref,'new-permission');
  assert.deepEqual(Object.keys(submitted).sort(),['application','authorization_ref','executor','name','source_engagement_id']);
  assert.equal(p.run('state.opened'),'b'.repeat(32));assert.match(p.text('message'),/review changed bindings/i);
});

test('lost retest draft response does not automatically replay and requires history recovery',async()=>{
  const p=fixture();p.run("state.apps=[{...state.app,metadata:{procedure:{supported:true}}}];renderRetestChooser();$('retest-application').value=state.app.id;$('retest-name').value='Fresh';$('retest-authorization').value='owned'");
  let posts=0;p.setFetch(async()=>{posts++;throw new TypeError('lost response');});
  await p.click('retest-form','submit');await flush();await p.click('retest-form','submit');await flush();
  assert.equal(posts,1);assert.equal(p.run('state.uncertainDraft'),true);assert.equal(p.run("$('create-retest').disabled"),true);
});

test('failed or mismatched comparison retains stale facts, disables linked approval and renders text safely',async()=>{
  const p=fixture();p.comparison.reason='<img src=x onerror=alert(1)>';await p.run('loadRetestComparison()');assert.match(p.text('retest-plan'),/<img src=x/);
  p.comparison.retest.engagement.id='b'.repeat(32);await assert.rejects(p.run('loadRetestComparison()'));
  assert.equal(p.run('state.retestStale'),true);assert.match(p.text('retest-plan'),/STALE/);assert.equal(p.run("$('approve').disabled"),true);
  assert.equal(p.run("$('stop').disabled"),false);
});

test('held comparison reads cannot block Stop or repopulate a disconnected context',async()=>{
  const p=fixture(),held=deferred();
  p.setFetch(async(path)=>{if(path.endsWith('/retest'))return held.promise;if(path.endsWith('/stop'))return response({stopped:[]});throw new Error('follow-up unavailable');});
  const pending=p.run('loadRetestComparison()');await p.run('stopAssessment()');
  assert.match(p.text('stop-receipt'),/Stop acknowledged/);held.resolve(response(p.comparison));await pending;assert.equal(p.run('state.retest'),null);
  p.setFetch(async()=>response(p.comparison));await p.run('loadRetestComparison()');p.run('disconnect()');
  assert.equal(p.text('retest-plan'),'');assert.equal(p.text('retest-comparison'),'');assert.equal(p.run('state.retest'),null);
});
