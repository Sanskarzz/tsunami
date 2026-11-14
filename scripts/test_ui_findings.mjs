// D7 presentation/byte-integrity tests. Fixtures are isolated from product data;
// the browser campaign exercises the real broker, validator and artifact bytes.
import {test} from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {ui} from './ui-test-support.mjs';
const run='d'.repeat(32),engagement='a'.repeat(32),procedure='owned-api/object-access/v2';
const flush=()=>new Promise(resolve=>setImmediate(resolve));
const response=data=>new Response(JSON.stringify(data));
function deferred(){let resolve;const promise=new Promise(r=>resolve=r);return {promise,resolve};}
function fixture(){
  const p=ui(),artifacts=[],captures=[],checks=[];
  const roles=['owner_identity','peer_identity','owner_baseline','peer_baseline','peer_probe','owner_recheck','peer_identity_recheck'];
  for(let i=0;i<7;i++){
    const id=(i+1).toString(16).padStart(32,'0'),account=[1,3,4,6].includes(i)?'peer':'owner',requestPath=[0,1,6].includes(i)?'/me':i===3?'/objects/b':'/objects/a';
    const bytes=Buffer.from(JSON.stringify({engagement_id:engagement,run_id:run,procedure,capture:{id,url:'https://owned.example.test'+requestPath,account,body:'synthetic-private-canary',status:200}}));
    const hash=createHash('sha256').update(bytes).digest('hex');
    artifacts.push({id:'E-'+hash,run_id:run,capture_id:id,sha256:hash,path:'evidence/'+hash+'.json',data_base64:bytes.toString('base64')});
    captures.push({id:'E-'+hash,run_id:run,capture_id:id,sha256:hash,artifact_path:'evidence/'+hash+'.json',account,request_path:requestPath,status:200,outcome:'response_captured',finished_at:'2030-01-01T00:00:01Z'});
    checks.push({role:roles[i],account,path:requestPath,capture_id:id,evidence_id:'E-'+hash,outcome:i===4?'expected_object_and_proof_returned':'verified'});
  }
  const snapshot={schema_version:1,engagement_id:engagement,observed_at:'2030-01-01T00:00:02Z',scope_digest:'c'.repeat(64),captures,
    assessments:[{run_id:run,state:'completed',procedure,display_title:'Cross-user access to a protected object',stored_verdict:'confirmed',expected_access:'deny',owner_account:'owner',peer_account:'peer',object_path:'/objects/a',protected_fields:['/private'],checks,remediation:'Authorize every object read on the server.',result:{verdict:'confirmed',reason:'The peer received the exact approved protected proof.',evidence_ids:checks.map(c=>c.capture_id)}}],
    candidates:[{id:'F-'+'1'.repeat(32),candidate_digest:'f'.repeat(64),display_title:'Cross-user access to a protected object',title:'Configured application title',status:'needs_human_review',reason:'The peer received the exact approved protected proof.',procedure,run_ids:[run],evidence_ids:captures.map(c=>c.id)}]};
  const calls=[];p.run("state.token='local-test-operator-token';state.currentPage='evidence';state.status=null");
  const route=async(path,options)=>{calls.push({path,method:options.method});if(path.endsWith('/findings'))return response(snapshot);if(path.includes('/evidence/'))return response(artifacts.find(a=>path.endsWith(a.id)));throw new Error('test route unavailable');};
  p.setFetch(route);return {...p,snapshot,artifacts,captures,checks,calls,route};
}

test('finding navigation loads metadata and selected check comparison without export or raw-body reads',async()=>{
  const p=fixture();await p.run('loadFindings()');
  assert.equal(p.calls.length,1);assert.match(p.calls[0].path,/\/findings$/);
  assert.match(p.text('candidates'),/Cross-user access to a protected object/);
  assert.match(p.text('candidates'),/Human review pending · severity unrated/);
  assert.match(p.text('evidence-comparison'),/Owner baseline.*owner · GET \/objects\/a.*Peer probe.*peer · GET \/objects\/a/s);
  assert.equal(p.nodes.get('evidence-checks').children[0].children[1].children.length,7);
  assert.match(p.text('finding-remediation'),/Authorize every object read/);
  for(const id of ['candidates','finding-context','evidence-comparison','evidence-checks','captures'])assert.doesNotMatch(p.text(id),/synthetic-private-canary/);
  assert.equal(p.run('state.rawEvidence'),null);
});

test('explicit raw inspection validates exact artifact bytes and keeps raw content out of API inspection',async()=>{
  const p=fixture();await p.run('loadFindings()');
  await p.run('loadRawEvidence(state.findings.captures[4])');
  assert.equal(p.calls.length,2);assert.equal(p.calls[1].path,`/v1/engagements/${engagement}/evidence/${p.artifacts[4].id}`);
  assert.match(p.text('capture-detail'),/synthetic-private-canary/);
  assert.match(p.text('raw-evidence-status'),/Byte integrity checked.*not independent authenticity/);
  assert.doesNotMatch(p.run('JSON.stringify(state.trace)'),/synthetic-private-canary|data_base64/);
  const downloaded=Buffer.from(p.run('state.rawEvidence.bytes'));
  assert.equal(createHash('sha256').update(downloaded).digest('hex'),p.artifacts[4].sha256);
  assert.equal(p.run("$('download-capture').disabled"),false);
});

test('tampered or mismatched raw evidence is never promoted to the body panel',async()=>{
  const p=fixture();await p.run('loadFindings()');
  p.setFetch(async(path,o)=>path.includes('/evidence/')?response({...p.artifacts[4],data_base64:Buffer.from('tampered sensitive body').toString('base64')}):p.route(path,o));
  await assert.rejects(p.run('loadRawEvidence(state.findings.captures[4])'));
  assert.equal(p.run('state.rawEvidence'),null);assert.equal(p.run("$('capture-detail').hidden"),true);
  assert.doesNotMatch(p.text('capture-detail'),/tampered/);assert.match(p.text('raw-evidence-status'),/no new body was promoted/);
});

test('missing rechecks, stopped attempts and no_issue_observed remain distinct from a finding',async()=>{
  const p=fixture();p.snapshot.candidates=[];p.snapshot.assessments[0].result={verdict:'inconclusive',reason:'Owner recheck missing.'};p.snapshot.assessments[0].display_title='Configured authorization check';
  p.snapshot.assessments[0].checks[5]={...p.checks[5],capture_id:undefined,evidence_id:undefined,outcome:'missing'};
  await p.run('loadFindings()');assert.match(p.text('evidence-checks'),/Owner recheck.*missing.*No selected capture/s);assert.match(p.text('finding-observation'),/Owner recheck missing/);
  p.snapshot.assessments[0].state='stopped';p.snapshot.assessments[0].result=undefined;await p.run('loadFindings()');
  assert.match(p.rows('finding-context')['Read-time verdict'],/No completed validation/);assert.match(p.text('finding-observation'),/no finding or clean result/);
  p.snapshot.assessments[0].state='completed';p.snapshot.assessments[0].result={verdict:'no_issue_observed',reason:'This configured probe was denied.'};await p.run('loadFindings()');
  assert.equal(p.rows('finding-context')['Read-time verdict'],'no_issue_observed');assert.equal(p.nodes.get('candidates').children.length,1);
  assert.match(p.text('candidates'),/covers only the selected rule/);
});

test('failed metadata retains an explicitly stale snapshot and renders untrusted text as text',async()=>{
  const p=fixture();p.snapshot.candidates[0].display_title='<img src=x onerror=alert(1)>';p.snapshot.assessments[0].remediation='<script>untrusted text</script>';
  await p.run('loadFindings()');assert.match(p.text('candidates'),/<img src=x/);assert.equal(p.text('finding-remediation'),'<script>untrusted text</script>');
  p.setFetch(async()=>{throw new TypeError('read unavailable');});await assert.rejects(p.run('loadFindings()'));
  assert.match(p.text('finding-summary'),/STALE.*2030/);assert.match(p.text('candidates'),/<img src=x/);
  assert.equal(p.run('state.findingsStale'),true);
});

test('Stop remains independent of a held finding read and invalidates the obsolete snapshot response',async()=>{
  const p=fixture(),held=deferred();
  p.setFetch(async(path,o)=>{if(path.endsWith('/findings'))return held.promise;if(path.endsWith('/stop'))return response({stopped:[run]});throw new Error('follow-up read unavailable');});
  const pending=p.run('loadFindings()');assert.equal(p.run("$('stop').disabled"),false);
  await p.run('stopAssessment()');assert.match(p.text('stop-receipt'),/Stop acknowledged/);
  held.resolve(response(p.snapshot));await pending;await flush();
  assert.equal(p.run('state.findings'),null);assert.equal(p.text('candidates'),'');
});

test('disconnect clears evidence detail and late raw responses cannot repopulate it',async()=>{
  const p=fixture(),held=deferred();await p.run('loadFindings()');
  p.setFetch(async(path,o)=>path.includes('/evidence/')?held.promise:p.route(path,o));
  const raw=p.run('loadRawEvidence(state.findings.captures[4])');p.run('disconnect()');held.resolve(response(p.artifacts[4]));
  await assert.rejects(raw);assert.equal(p.run('state.rawEvidence'),null);assert.equal(p.text('capture-detail'),'');assert.equal(p.text('finding-context'),'');
});

test('expected and observed comparison uses trusted selected check metadata and preserves inconclusive outcomes',async()=>{
  const p=fixture();await p.run('loadFindings()');
  assert.equal(p.text('finding-verdict'),'confirmed');assert.match(p.text('finding-expected'),/must not receive/);assert.match(p.text('finding-observed'),/owner’s object.*private proof/);assert.match(p.text('finding-review'),/severity unrated/);
  assert.match(p.text('evidence-comparison'),new RegExp(p.captures[4].id));
  p.snapshot.assessments[0].result={verdict:'inconclusive',reason:'Incomplete owner recheck.'};await p.run('loadFindings()');
  assert.equal(p.text('finding-verdict'),'inconclusive');assert.doesNotMatch(p.text('finding-review'),/severity unrated/);
  assert.equal(p.calls.filter(c=>c.path.includes('/evidence/')||c.path.endsWith('/export')).length,0);
});
