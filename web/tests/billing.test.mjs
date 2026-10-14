import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync, statSync } from 'node:fs';
import { join } from 'node:path';
import { routes, parse } from '../src/lib/billing/routes.ts';
import { parseMoney, usd, us } from '../src/lib/billing/format.ts';

const root = new URL('../src/lib/billing/', import.meta.url).pathname;
const read = (p) => readFileSync(new URL(p, import.meta.url), 'utf8');
const goCodes = JSON.parse(read('../../internal/billing/codes.json'));
const screens = JSON.parse(read('../src/lib/billing/screens.json'));
const spec = read('../../docs/BILLING_UI_WORKFLOW.md');

function files(dir) { return readdirSync(dir).flatMap(f => { const p = join(dir, f); return statSync(p).isDirectory() ? files(p) : [p]; }); }
const components = new Map(files(root).filter(f => f.endsWith('.svelte')).map(f => [f.split('/').pop().replace('.svelte', ''), f]));

test('the UI and the Go server read one code-list file', () => {
  assert.match(read('../src/lib/billing/codes.ts'), /internal\/billing\/codes\.json/);
  assert.match(read('../../internal/billing/store.go'), /go:embed codes\.json/);
});

test('code lists match spec §4 exactly', () => {
  const row = (name) => spec.split('\n').find(l => l.startsWith(`| ${name} |`)).split('|')[2].trim();
  const list = (name) => row(name).split(/,\s*/);
  assert.deepEqual(goCodes.disbursementReasons, list('Disbursement reason'));
  assert.deepEqual(goCodes.suspenseActions, list('Suspense actions'));
  assert.deepEqual(goCodes.accountPlans, row('Account plans').split(/,\s*(?=Paper)/));
  assert.deepEqual(goCodes.accountClasses, list('Account classes'));
  assert.deepEqual(goCodes.membershipTypes, list('Membership types'));
  assert.deepEqual(goCodes.partySearchBy, list('Party search by'));
  assert.deepEqual(goCodes.errorTypes, list('Error types'));
  assert.deepEqual(goCodes.holdResumeOptions, list('Hold resume option'));
  assert.deepEqual(goCodes.paymentMethodFilters, row('Payment methods (filter)').split('.')[0].split(/,\s*/));
  assert.deepEqual(goCodes.reverseReasons, ['RP-Reverse Payment']);
  assert.equal(goCodes.reports.length, 9);
});

test('every route resolves to a component', () => {
  for (const r of routes) {
    assert(components.has(r.component), `${r.pattern} → ${r.component}.svelte missing`);
    const sample = '#billing/' + r.pattern.replace(/:(\w+)/g, (_, k) => k === 'kind' ? 'address' : k === 'mode' ? 'correct' : 'X1');
    assert.equal(parse(sample).route.component, r.component, sample);
  }
  // Each component named by a route is rendered by one of the dispatchers.
  const dispatch = ['BillingWorkspace.svelte', 'account/AccountLayout.svelte', 'party/PartyLayout.svelte'].map(f => readFileSync(join(root, f), 'utf8')).join('\n');
  for (const c of new Set(routes.map(r => r.component))) assert(dispatch.includes(`'${c}'`) || c === 'Home' && dispatch.includes("'Home'"), `${c} is not dispatched`);
  assert.equal(parse('#browse').route.component, 'Home');
  assert.equal(parse('#billing/account/CLAD3057414/suspense').params.account, 'CLAD3057414');
});

test('every screenshot #001–#117 is mapped by exactly one screen', () => {
  const seen = new Map();
  for (const s of screens) for (const n of s.shots) { assert(!seen.has(n), `#${n} mapped by ${seen.get(n)} and ${s.id}`); seen.set(n, s.id); }
  for (let n = 1; n <= 117; n++) assert(seen.has(n), `#${String(n).padStart(3, '0')} is not mapped`);
  assert.equal(seen.size, 117);
  for (const s of screens) {
    assert(components.has(s.component), s.component);
    if (s.route) assert(routes.some(r => r.pattern === s.route && r.component === s.component), `${s.id}: route ${s.route} → ${s.component}`);
  }
});

test('money and dates are handled without floats', () => {
  assert.equal(parseMoney('$1,234.5'), 123450);
  assert.equal(parseMoney('0.1'), 10);
  assert.equal(parseMoney('10.27'), 1027);
  assert.equal(parseMoney('1.234'), null);
  assert.equal(usd(98100), '$981.00');
  assert.equal(usd(-5), '-$0.05');
  assert.equal(us('2027-01-01'), '1/1/2027');
});
