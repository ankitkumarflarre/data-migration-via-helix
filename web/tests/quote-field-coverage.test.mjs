import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {execFileSync} from 'node:child_process';
import {createHash} from 'node:crypto';
import {pages, pageValues, matches, quotesAPI} from '../src/lib/personalhome/quote-model.ts';
import {fieldValue,mapURL} from '../src/lib/personalhome/field-values.ts';
const read=path=>JSON.parse(readFileSync(new URL(path,import.meta.url)));
const catalog=read('../src/lib/personalhome/catalog.json');
const audit=read('../src/lib/personalhome/field-audit.json');

test('all 170 original HTML inventory rows map to a rendered field, collection control or explicit unavailable action',()=>{
 const root=new URL('../../',import.meta.url);
 const source=readFileSync(new URL(`../../reference/${catalog.source}`,import.meta.url));
 assert.equal(createHash('sha256').update(source).digest('hex'),audit.sha256);
 const parsed=JSON.parse(execFileSync('python3',['-c','import sys,json;sys.path.insert(0,"scripts");from extract_personalhome import catalog;print(json.dumps(catalog()))'],{cwd:root,encoding:'utf8'}));
 assert.deepEqual(parsed,catalog,'catalog must match the actual HTML, not just its stored fingerprint');
 const refs=new Set();
 for(const p of pages){
  assert.equal(new Set(p.fields.map(f=>f.key)).size,p.fields.length,`duplicate fields on ${p.id}`);
  for(const f of [...p.fields,...p.collection?.fields??[],...p.actions??[]])for(const ref of f.sourceRefs??[]){assert(!refs.has(ref),`duplicate ${ref}`);refs.add(ref);}
  for(const [role,ref] of Object.entries(p.collection?.sourceControls??{})){assert(['collection-add','collection-remove','collection-details'].includes(role));assert(!refs.has(ref));refs.add(ref);}
 }
 assert.equal(refs.size,170);
 assert.equal(audit.entries.length,170);
 for(const sp of catalog.pages)for(const sf of sp.fields){
  const ref=`${sp.id}:${sf.id}`;
  assert(refs.has(ref),`missing ${ref}: ${sf.label}`);
  const row=audit.entries.find(r=>r.sourcePage===sp.id&&r.sourceID===sf.id);
  assert.equal(row.sourceKey,sf.tech);
  const page=pages.find(p=>p.id===row.targetPage);assert(page);
  if(row.target.startsWith('collection-'))assert.equal(page.collection.sourceControls[row.target],ref);
  else assert([...page.fields,...page.collection?.fields??[],...page.actions??[]].some(f=>f.key===row.target&&f.sourceRefs?.includes(ref)),`${ref} audit must resolve to the real schema`);
 }
});
test('every visible source field has at least one reachable form scenario',()=>{
 const base={'DwellingInput.Form':'HO3','DwellingInput.UseType':'Primary','DwellingInput.SwimmingPool':'Inground','ReplacementCostDwellingInput.Indicator':true,'LossAssessmentInput.Indicator':true,'History.HasPrior':true,'Claims.HasLosses':true};
 for(const p of pages)for(const f of [...p.fields,...p.collection?.fields??[]]){
  if(!f.sourceRefs||f.hidden)continue;
  assert(['HO3','HO6'].some(form=>matches(f.showWhen,{...base,'DwellingInput.Form':form})),`${p.id}: ${f.key} unreachable`);
 }
 const risk=pages.find(p=>p.id==='dwellinginfo');
 assert(matches(risk.fields.find(f=>f.key==='DwellingInput.NumberOfFloor').showWhen,base));
 assert(!matches(risk.fields.find(f=>f.key==='DwellingInput.NumberOfFloor').showWhen,{...base,'DwellingInput.Form':'HO6'}));
 for(const use of ['Secondary','Seasonal3to6','Seasonal6Mths'])assert(!matches(risk.fields.find(f=>f.key==='DwellingInput.MonthsUnoccupied').showWhen,{...base,'DwellingInput.UseType':use}));
});
test('read-only calculations update live and unknown provider values never become zero',()=>{
 const fields=pages.flatMap(p=>p.fields);
 const get=key=>fields.find(f=>f.key===key);
 assert.equal(fieldValue(get('PolicyInput.ExpirationDate'),{'PolicyInput.EffectiveDate':'2028-02-29'}),'2029-02-28');
 assert.equal(fieldValue(get('AccountInput.Name'),{'AccountInput.FirstName':'Avery','AccountInput.LastName':'Demo'}),'Avery Demo');
 assert.equal(fieldValue(get('CoverageDLossOfUseOutput.IncludedLimit'),{'DwellingInput.Form':'HO6','CoverageCPersonalPropertyHO46Input.Limit':75000}),30000);
 assert.equal(fieldValue(get('PolicyPremiums.Premium'),{'PolicyPremiums.Premium':999}),undefined);
 assert.equal(fieldValue(get('WaterBackupAndSumpOverflowInput.Limit'),{}),10000);
 assert.equal(mapURL('google',{}),undefined);
 assert(mapURL('google',{'AccountInput.Address1':'1 A&B Lane','AccountInput.City':'Demo'}).includes('A%26B'));
 const body=pageValues(pages.find(p=>p.id==='dwellingcoverage'),{'RiskInput.UseDeductibleByPeril':true,'DwellingInput.Deductible':'500'});
 assert(!Object.hasOwn(body,'DwellingInput.Deductible'));
 assert(!Object.hasOwn(body,'CoverageAOutput.Premium'));
});
test('saving hydrated collection rows strips generated fields but preserves the new inputs',async()=>{
 const original=globalThis.fetch;let body;
 globalThis.fetch=async(_url,options)=>{body=JSON.parse(options.body);return {ok:true,text:async()=>JSON.stringify({id:'saved'})};};
 try{
  await quotesAPI.save({id:'test',version:1},'claimshistory',{},[{'LossInput.Source':'Applicant disclosure','LossOutputNonShredded.NumberDisplay':'1','LossInput.LossType':'Wind'}],false);
  assert.equal(body.rows[0]['LossInput.Source'],'Applicant disclosure');
  assert.equal(body.rows[0]['LossInput.LossType'],'Wind');
  assert(!Object.hasOwn(body.rows[0],'LossOutputNonShredded.NumberDisplay'));
 }finally{globalThis.fetch=original;}
});
