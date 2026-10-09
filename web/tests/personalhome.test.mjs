import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { activeFlow, defaultScenario, flow, catalog, report, ruleset, templates, correspondences, templateLinks, inventoryFor, mappingFor, matchesField, exportManifest } from '../src/lib/personalhome/model.ts';
const schema = JSON.parse(readFileSync(new URL('../../testdata/describe.json', import.meta.url)));

test('default journey follows narrative order and skips suppressed/conditional pages', () => {
  assert.deepEqual(activeFlow(defaultScenario).map(p => p.id), ['newquote','account','dwellinginfo','dwellingcoverage','underwriting','insurancehistory','claimshistory','summary','additionalinterests','commit']);
});
test('all branch combinations honor CLUE conjunction and payment in-force gate', () => {
  for (let bits = 0; bits < 32; bits++) {
    const s = {...defaultScenario, clueSuccess: !!(bits&1), clueClaims: !!(bits&2), scoreAvailable: !!(bits&4), billing: !!(bits&8), inForce: !!(bits&16)};
    const ids = activeFlow(s).map(p => p.id);
    assert.equal(ids.includes('clue'), s.clueSuccess && s.clueClaims);
    assert.equal(ids.includes('score'), s.scoreAvailable);
    assert.equal(ids.includes('payment'), s.billing && !s.inForce);
    assert.equal(ids.includes('billing'), s.billing);
    assert(!ids.includes('policydetail') && !ids.includes('pricing') && !ids.includes('locationdetail'));
  }
});
test('all inventories are accessible and source fingerprints match', () => {
  assert.equal(catalog.pages.length, 9);
  assert.equal(catalog.pages.reduce((sum,p) => sum+p.fields.length, 0),170);
  for (const p of catalog.pages) {
    assert(flow.some(node => inventoryFor(node)?.id === p.id));
    assert.equal(new Set(p.fields.map(f => f.id)).size,p.fields.length);
  }
  for (const data of [catalog,report]) {
    const bytes = readFileSync(new URL(`../../reference/${data.source}`, import.meta.url));
    assert.equal(createHash('sha256').update(bytes).digest('hex'),data.sha256);
  }
});
test('mapping report retains unapproved statuses and exact rule count', () => {
  assert.equal(report.columns.length,81);
  assert.equal(report.columns.filter(c => c.status.startsWith('Confirmed')).length,20);
  assert.equal(report.columns.filter(c => c.status.startsWith('Review')).length,39);
  assert.equal(report.columns.filter(c => c.status === 'Not Found').length,22);
  // 20 rules from confirmed rows (BU excluded, Number of Stories split by form) and 35 for Review / Not found columns.
  assert.equal(ruleset.rules.length,55);
  assert.equal(ruleset.rules.filter(r => r.report_status.startsWith('Confirmed')).length,20);
  for (const rule of ruleset.rules) assert(report.columns.some(c => c.column === rule.excel_column && c.status === rule.report_status));
});
test('correspondences and template targets reference actual fields', () => {
  for (const [key,id] of Object.entries(correspondences)) {
    const [p,f] = key.split(':');
    assert(catalog.pages.find(x => x.id === p)?.fields.some(x => x.id === +f));
    const rule = ruleset.rules.find(r => r.id === id); assert(rule);
    assert(schema[rule.target.variant]?.fields.some(f => f.key === rule.target.field));
  }
  for (const link of Object.values(templateLinks)) {
    assert(schema[link.variant]?.fields.some(f => f.key === link.field));
    assert(templates.records.some(r => r.variant === link.variant && r.fields?.[link.field]));
  }
});
test('unmapped and ambiguous mailing county never get guessed targets', () => {
  const account = catalog.pages.find(p => p.id === 'account');
  assert.equal(mappingFor('account',account.fields.find(f => f.id === 21)).status,'Unmapped');
  assert.equal(mappingFor('account',account.fields.find(f => f.id === 2)).target,undefined);
});
test('current job overrides and exclusions propagate to preview and search', () => {
  const field = catalog.pages.find(p => p.id === 'dwellinginfo').fields.find(f => f.id === 6);
  const effective = [{id:'HO-10',effective_target:{variant:'custom.variant',field:'custom_year'},excluded:false,overridden:true,effective_transform:{case:'upper'}}];
  assert.equal(mappingFor('dwellinginfo',field,effective).target.field,'custom_year');
  assert(matchesField('dwellinginfo',field,'custom.variant',effective));
  effective[0].excluded = true;
  assert.equal(mappingFor('dwellinginfo',field,effective).status,'Excluded in current job');
});
test('manifest is deterministic and explicitly not a bind request', () => {
  assert.deepEqual(exportManifest(defaultScenario),exportManifest(defaultScenario));
  assert.match(exportManifest(defaultScenario).scope,/not a quote or bind request/);
});
