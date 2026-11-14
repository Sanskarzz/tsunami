// Presentation tests only: these fixtures never enter the running product or
// contact a target. Exercise the actual UI code without browser dependencies.
import {test} from 'node:test';
import assert from 'node:assert/strict';
import {ui} from './ui-test-support.mjs';


test('scope and attempt use separate authoritative limits and states', () => {
  const page = ui();
  page.run('selectedApplication();renderPlan();renderStatus()');
  const scope = page.rows('scope-details');
  assert.match(scope.Limits, /3 target reservations\/run/);
  assert.match(scope.Limits, /up to 17 seconds\/run, capped at scope expiry/);
  assert.match(scope.Limits, /7 child runs maximum · 1 concurrent maximum/);
  assert.match(scope['Per-run rate'], /2000 ms/);
  assert.equal(scope['Definition digest'], 'b'.repeat(64));
  assert.match(scope['Access availability'], /configured locally.*session health unknown/);
  const context = page.rows('run-context');
  assert.equal(context['Approval record'], 'active');
  assert.equal(context['Latest attempt'], 'completed · no_issue_observed');
  const attempt = page.rows('run-policy');
  assert.equal(attempt['Target reservations'], '2/2');
  assert.match(attempt['Actual run deadline'], /2030/);
  assert.match(page.text('observed-at'), /executor check: Not recorded/);
  assert.doesNotMatch(page.text('observed-at'), /STALE/);
});

test('missing metadata stays unknown rather than inventing allowances or health', () => {
  const page = ui();
  page.run('delete state.view.metadata;delete state.status.target_request_limit;delete state.status.procedure_digest;renderPlan();renderStatus()');
  assert.match(page.rows('scope-details').Limits, /not available/);
  assert.match(page.rows('scope-details')['Access availability'], /not returned.*health unknown/);
  assert.equal(page.rows('run-policy')['Target reservations'], '2/unknown');
  assert.equal(page.rows('run-policy')['Definition digest'], 'Not recorded');
  assert.equal(page.rows('run-context')['Latest attempt'], 'Not returned by server');
});

test('failed refresh retains the last snapshot and marks it stale', async () => {
  const page = ui();
  page.run("state.token='local-test-operator-token';renderPlan();renderStatus()");
  await assert.rejects(page.run('refresh()'), /read unavailable/);
  assert.match(page.text('observed-at'), /STALE · latest read failed/);
  assert.equal(page.rows('run-policy')['Target reservations'], '2/2');
  assert.equal(page.text('verdict'), 'no_issue_observed');
});

test('unavailable access or scope gates disable Start without disabling Stop', () => {
  const page = ui();
  page.run("state.token='local-test-operator-token';state.view.engagement.run_ids=[];state.run='';controls()");
  assert.equal(page.run("$('start').disabled"), false);
  for (const change of ["state.view.metadata.permission='disabled'", "state.view.metadata.permission='active';state.view.metadata.access.availability='unavailable'", 'delete state.view.metadata']) {
    page.run(change + ';controls()');
    assert.equal(page.run("$('start').disabled"), true);
    assert.equal(page.run("$('stop').disabled"), false);
  }
});

test('activity client requests an explicit bounded page without starting polling', async () => {
  const page = ui();
  page.run(`state.token='local-test-operator-token';globalThis.calls=[];
    fetch=async (path, options)=>{calls.push({path,method:options.method});
      const bytes=new TextEncoder().encode(JSON.stringify({items:[],next_cursor:7,has_more:false,gap:false}));
      let read=false;return {status:200,ok:true,body:{getReader(){return {async read(){if(read)return {done:true};read=true;return {done:false,value:bytes}}}}}};
    };`);
  const result = await page.run(`readRunActivity('${'d'.repeat(32)}',7,20)`);
  assert.equal(result.next_cursor, 7);
  assert.deepEqual(JSON.parse(page.run('JSON.stringify(calls)')), [{path: '/v1/runs/' + 'd'.repeat(32) + '/events?after=7&limit=20', method: 'GET'}]);
  assert.equal(page.run('state.polling'), false);
  for (const args of ["'invalid',0,50", `'${'d'.repeat(32)}',-1,50`, `'${'d'.repeat(32)}',0,101`, `'${'d'.repeat(32)}',9007199254740992,50`]) {
    assert.throws(() => page.run('readRunActivity(' + args + ')'), /Invalid activity cursor/);
  }
  assert.equal(page.run('calls.length'), 1);
});

test('remaining allowance is derived only from recorded attempt policy and keeps stale context',()=>{
  const p=ui();p.run('renderStatus()');assert.equal(p.text('budget-remaining'),'0 of 2 unreserved');
  p.run('state.statusStale=true;renderStatus()');assert.match(p.text('budget-remaining'),/^STALE/);
  p.run('delete state.status.target_request_limit;renderStatus()');assert.match(p.text('budget-remaining'),/Not observed/);
  p.run('state.status=null;state.statusStale=false;renderStatus()');assert.equal(p.text('budget-deadline'),'Not observed');
});
