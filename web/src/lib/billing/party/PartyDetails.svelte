<script lang="ts">
 import type {PartyView} from '../api';
 import {href,acct} from '../routes';
 import {us} from '../format';
 import Page from '../ui/Page.svelte';
 import Section from '../ui/Section.svelte';
 import {oneLine,kinds} from './kinds';
 let {party:p}:{party:PartyView}=$props();
 const cur=(k:string)=>(p.rows[k]??[]).filter(r=>r.current);
 const name=$derived(cur('name')[0]?.fields??{});
 const addrKinds=$derived([...new Set(cur('address').map(r=>r.fields.kind))]);
 const d=$derived(p.demographics??{});
 // svelte-ignore state_referenced_locally
 const base='party/'+p.id;
</script>
<Page title="People & Places Details" subtitle={`${p.kind} associated with ${p.associated_with} · Party number ${p.number}`} date={false}>
 <Section title="Name Address & Contact Information" soft>
  <p>This party is a <strong>{p.kind}</strong> associated with <strong>{p.associated_with}</strong></p>
  <dl class="facts"><div><dt>{p.kind==='Person'?'Person Name':'Organization Name'}</dt><dd>{p.name}</dd></div><div><dt>Correspondence Name</dt><dd>{name.correspondence??''}</dd></div><div><dt>Preferred method of contact</dt><dd>{name.contact??''}</dd></div><div><dt>Country</dt><dd>United States</dd></div><div><dt>Party Number</dt><dd>{p.number}</dd></div></dl>
  <div class="b-actions"><a class="btn sm" href={href(base+'/name/correct')}>Correct Contact Information</a><a class="btn sm" href={href(base+'/name/replace')}>Replace Contact Information</a></div>
  {#each addrKinds as k}<p class="b-label">{k} Addresses</p>{#each cur('address').filter(r=>r.fields.kind===k) as r}<p>{oneLine(r.fields)} as of {us(r.effective)}</p>{/each}{/each}
  <a href={href(base+'/address')}>Work with addresses</a>
 </Section>
 <Section title="Phones" soft open={false}>{#each cur('phone') as r}<p>{r.fields.type}: {r.fields.number}{r.fields.primary==='Yes'?' (primary)':''}</p>{/each}<a href={href(base+'/phone')}>Work with phones</a></Section>
 <Section title="Email" soft open={false}>
  <p class="b-label">Active</p>
  {#if cur('email').length}<div class="b-grid"><table><thead><tr><th>Email Address</th><th>Email Type</th><th>Comments</th><th>Effective</th><th>Primary E-Mail</th></tr></thead><tbody>{#each cur('email') as r}<tr><td>{r.fields.address}</td><td>{r.fields.type}</td><td>{r.fields.comments??''}</td><td>{r.effective} To {r.expiration}</td><td>{r.fields.primary==='Yes'?'✓':''}</td></tr>{/each}</tbody></table></div>{/if}
  <a href={href(base+'/email')}>Work with email</a>
 </Section>
 <Section title="Locations" soft open={false}><p class="b-label">Physical Locations</p>{#each cur('location') as r}<p>{oneLine(r.fields)} · Physical Address Effective beginning {us(r.effective)}</p>{/each}<a href={href(base+'/location')}>Work with locations</a></Section>
 <Section title="Demographics & Other Names" soft open={false}>
  <p class="b-label">Demographics</p>
  <dl class="facts">{#each [['Preferred Language','language'],['Gender','gender'],['Ethnic Origin','ethnic_origin'],['DOB','dob'],['Marital Status','marital_status'],['Occupation','occupation'],['US Citizen?','us_citizen'],['Social Security #','ssn'],['Date of Death','date_of_death'],['Number Of Dependents','dependents'],['Profession','profession'],['Company Personnel','company_personnel'],['Other ID','other_id'],['Party Comments','comments']] as [l,k]}<div><dt>{l}</dt><dd>{k==='ssn'&&d[k]?'***-**-'+d[k].slice(-4):k==='dob'||k==='date_of_death'?us(d[k]):d[k]??''}</dd></div>{/each}<div><dt>OFAC Check Not Required</dt><dd>Unanswered</dd></div></dl>
  <a href={href(base+'/demographics')}>Work with demographics</a>
  <p class="b-label">Other Names</p>
  {#if cur('othername').length}<div class="b-grid"><table><thead><tr><th>Name</th><th>Type of Name</th><th>Comments</th><th>Effective</th></tr></thead><tbody>{#each cur('othername') as r}<tr><td>{[r.fields.first,r.fields.middle,r.fields.last].filter(Boolean).join(' ')}</td><td>{r.fields.type}</td><td>{r.fields.comments??''}</td><td>{r.effective} To {r.expiration}</td></tr>{/each}</tbody></table></div>{/if}
  <a href={href(base+'/othername')}>Work with names</a>
 </Section>
 <Section title="License & Membership Information" soft open={false}>
  <p class="b-label">Licenses</p>{#each cur('license') as r}<p>{kinds.license.columns.map(([,f])=>f(r.fields)).filter(Boolean).join(' · ')}</p>{/each}<a href={href(base+'/license')}>Work with licenses</a>
  <p class="b-label">Memberships</p>{#each cur('membership') as r}<p>{r.fields.type}: {r.fields.id}</p>{/each}<a href={href(base+'/membership')}>Work with memberships</a>
 </Section>
 <Section title="People & Places Relationships" soft open={false}><p class="b-note">Billing accounts: {#each p.accounts as a,i}{i?', ':''}<a href={href(acct(a))}>{a}</a>{:else}none{/each}. Other relationships are not shown in the walkthrough. <a href={href(base+'/relationships')}>Work with relationships</a></p></Section>
 <div class="b-actions"><button class="btn" onclick={()=>history.back()}>Cancel</button><span class="spacer"></span><button class="btn primary" onclick={()=>history.back()}>Finish</button></div>
</Page>
<style>.facts{display:grid;grid-template-columns:repeat(auto-fill,minmax(180px,1fr));gap:10px;margin:0}dt{font-weight:600;color:var(--text-muted);font-size:11px;text-transform:uppercase;letter-spacing:.5px}dd{margin:2px 0 0;font-size:14px}p{margin:0}</style>
