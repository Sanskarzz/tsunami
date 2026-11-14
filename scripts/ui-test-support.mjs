// Isolated DOM/API fixtures for tests only; never shipped as product data.
import {readFileSync} from 'node:fs';
import vm from 'node:vm';
import {webcrypto} from 'node:crypto';

class Element {
  constructor(tag = 'div') { this.tag = tag; this.children = []; this.dataset = {}; this.value = ''; this.listeners = {}; }
  set textContent(value) { this.text = String(value); this.children = []; }
  get textContent() { return (this.text || '') + this.children.map(c => c.textContent ?? String(c)).join(''); }
  append(...children) { this.children.push(...children); }
  replaceChildren(...children) { this.text = ''; this.children = [...children]; }
  addEventListener(name, fn) { (this.listeners[name] ||= []).push(fn); }
  get options() { return this.children; }
}

export function ui(options = {}) {
  const nodes = new Map();
  const document = {
    getElementById(id) { if (!nodes.has(id)) nodes.set(id, new Element()); return nodes.get(id); },
    createElement(tag) { return new Element(tag); },
    createTextNode(text) { const node=new Element('text');node.textContent=text;return node; },
    querySelectorAll() { return []; },
    querySelector() { return null; }
  };
  const context = vm.createContext({document, window: {addEventListener() {}},
    createOnboardingUI: () => ({lock() {},reset() {}}), setTimeout: options.setTimeout || setTimeout, clearTimeout: options.clearTimeout || clearTimeout, AbortController,
    performance, TextEncoder, TextDecoder, Uint8Array, crypto:webcrypto, atob, fetch: options.fetch || (async () => { throw new TypeError('read unavailable'); })});
  const source = readFileSync(new URL('../internal/webui/static/app.js', import.meta.url), 'utf8');
  if(options.onboarding)vm.runInContext(readFileSync(new URL('../internal/webui/static/onboarding.js', import.meta.url),'utf8').replace('export function createOnboardingUI','function createOnboardingUI'),context);
  vm.runInContext(source.replace(/^import \{createOnboardingUI\} from '.\/onboarding.js';$/m, ''), context);
  const plan = {id: 'owned-api', title: 'Owned application', version: 'http-object-access/v2', origin: 'https://owned.example.test',
    approved_addresses: ['192.0.2.10'], expected_access: 'deny', identity: [{account: 'owner', path: '/me'}, {account: 'peer', path: '/me'}],
    baselines: [{account: 'owner', path: '/objects/a'}, {account: 'peer', path: '/objects/b'}], probe: {account: 'peer', path: '/objects/a'}};
  const digest = 'b'.repeat(64), id = 'a'.repeat(32);
  const view = {engagement: {id, application: plan, state: 'active', target: plan.origin, items: [{procedure: 'owned-api/object-access/v2', definition_digest: digest}],
    scope_digest: 'c'.repeat(64), approved_digest: 'c'.repeat(64), max_runs: 7, max_concurrent: 1, run_ids: ['d'.repeat(32)], events: [], expires_at: new Date(Date.now() + 3600000).toISOString()},
    metadata: {permission: 'active', execution_mode: 'scripted', executor: 'native', max_run_duration_seconds: 17,
      observed_at: '2030-01-01T00:00:00Z', access: {availability: 'configured', credential_health: 'unknown'},
      procedures: [{id: 'owned-api/object-access/v2', version: 'http-object-access/v2', digest, supported: true, max_requests: 3, min_interval_ms: 2000}],
      latest_attempt: {id: 'd'.repeat(32), state: 'completed', verdict: 'no_issue_observed'}}};
  const status = {run_id: 'd'.repeat(32), local_operations: 0, state: 'completed', permission: 'closed', execution: 'unknown_after_resource_removal', activity: 'no_local_work', cleanup: 'api_resources_absent',
    execution_mode: 'scripted', executor: '', target_request_limit: 2, target_requests_reserved: 2, uncertain_operations: 0,
    procedure_id: 'owned-api/object-access/v2', procedure_version: 'http-object-access/v2', procedure_digest: digest,
    expires_at: '2030-01-01T00:00:11Z', observed_at: '2030-01-01T00:00:12Z', assessment: {verdict: 'no_issue_observed', reason: 'Approved probe denied',
      validation: {baselines_verified: 6, baselines_required: 6, probe_captured: true}}};
  vm.runInContext(`state.view=${JSON.stringify(view)};state.app=state.view.engagement.application;state.run=state.view.engagement.run_ids[0];state.status=${JSON.stringify(status)};`, context);
  const run = expression => vm.runInContext(expression, context);
  const text = id => document.getElementById(id).textContent;
  const rows = id => Object.fromEntries(document.getElementById(id).children.reduce((pairs, node, i, all) => {
    if (i % 2 === 0) pairs.push([node.textContent, all[i + 1].textContent]); return pairs;
  }, []));
  const click = (id, type = 'click') => Promise.all((document.getElementById(id).listeners[type] || []).map(fn => fn({preventDefault() {}})));
  const setFetch = fetch => { context.fetch = fetch; };
  return {run, text, rows, click, setFetch, nodes, view, status};
}
