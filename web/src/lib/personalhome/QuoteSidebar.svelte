<script lang="ts">
 import Icon from './WorkspaceIcon.svelte';
 import type {Quote,Value} from './quote-model';
 let {values,quote,coapplicants=[]}:{values:Record<string,Value>;quote:Quote|null;coapplicants:Record<string,Value>[]}=$props();
 let person=$state(0);
 const selected=$derived(person>0?coapplicants[person-1]:undefined);
 const first=$derived(selected?.['PersonInput.FirstName']??values['AccountInput.FirstName']);
 const last=$derived(selected?.['PersonInput.LastName']??values['AccountInput.LastName']);
 const name=$derived([first,last].filter(Boolean).join(' ')||'New applicant');
 const initials=$derived([first,last].filter(Boolean).map(v=>String(v)[0]).join('')||'—');
 const show=(v:Value|undefined)=>v===null||v===undefined||v===''?'—':String(v);
 const money=(v:Value|undefined)=>typeof v==='number'?new Intl.NumberFormat('en-US',{style:'currency',currency:'USD',maximumFractionDigits:0}).format(v):'—';
</script>
<aside class="details-column" aria-label="Application details">
 <section class="detail-card">
  <h3><span class="icon"><Icon name="user"/></span>Application information</h3>
  <div class="card-body">
   <div class="person-tabs" aria-label="Applicant selection"><button class:active={!selected} aria-pressed={!selected} onclick={()=>person=0}>Primary</button>{#each coapplicants as co,i}<button class:active={person===i+1} aria-pressed={person===i+1} onclick={()=>person=i+1}>Co-applicant{coapplicants.length>1?' '+(i+1):''}</button>{/each}</div>
   <div class="identity"><span class="initials">{initials}</span><div><strong>{name}</strong><p>{selected?'Co-applicant':'Primary applicant'}</p></div></div>
   <dl><div><dt>First name</dt><dd>{show(first)}</dd></div><div><dt>Last name</dt><dd>{show(last)}</dd></div><div><dt>Middle initial</dt><dd>{show(selected?.['PersonInput.MiddleName']??(!selected?values['AccountInput.MiddleName']:undefined))}</dd></div><div><dt>Role</dt><dd>{selected?'Co-applicant':'Named insured'}</dd></div>{#if !selected}<div class="wide"><dt>Email address</dt><dd class="accent">{show(values['AccountInput.Email'])}</dd></div><div class="wide"><dt>Primary phone</dt><dd>{show(values['AccountInput.PrimaryPhone'])}</dd></div>{/if}</dl>
  </div>
 </section>
 <section class="detail-card">
  <h3><span class="icon"><Icon name="pin"/></span>Property address</h3>
  <div class="card-body"><div class="address"><Icon name="pin"/><div><strong>{values['LocationInput.Address1']||'Property address not entered'}</strong><p>{[values['LocationInput.City'],values['LocationInput.State'],values['LocationInput.ZipCode']].filter(Boolean).join(', ')||'Add details in Risk schedule'}</p></div></div><dl><div><dt>City</dt><dd>{show(values['LocationInput.City'])}</dd></div><div><dt>State</dt><dd>{show(values['LocationInput.State'])}</dd></div><div><dt>County</dt><dd>{show(values['LocationInput.County'])}</dd></div><div><dt>ZIP code</dt><dd>{show(values['LocationInput.ZipCode'])}</dd></div><div><dt>Usage</dt><dd>{show(values['DwellingInput.UseType'])}</dd></div><div><dt>Year built</dt><dd>{show(values['DwellingInput.YearBuilt'])}</dd></div></dl></div>
 </section>
 <section class="detail-card">
  <h3><span class="icon"><Icon name="shield"/></span>Coverage at a glance</h3>
  <div class="card-body"><dl><div><dt>Dwelling · A</dt><dd class="amount">{money(values['CoverageADwellingInput.Limit'])}</dd></div><div><dt>Personal property · C</dt><dd class="amount">{money(values[values['DwellingInput.Form']==='HO6'?'CoverageCPersonalPropertyHO46Input.Limit':'CoverageCPersonalPropertyHO3Input.Limit'])}</dd></div><div><dt>Loss of use · D</dt><dd>{money(values['CoverageDLossOfUseInput.Limit'])}</dd></div><div><dt>All-peril deductible</dt><dd>{show(values['DwellingInput.Deductible'])}</dd></div></dl><div class="rating-note"><span class="dot"></span>{quote?.status==='ready_for_rating'?'Application awaiting rating':'Premium not yet calculated'}</div></div>
 </section>
</aside>
<style>
 .details-column{display:grid;gap:18px;align-content:start;min-width:0}.detail-card{border:1px solid var(--border);border-radius:10px;background:var(--surface);box-shadow:0 2px 4px #18244004;overflow:hidden}h3{font-size:14px;padding:16px;display:flex;align-items:center;gap:10px;border-bottom:1px solid var(--border)}.icon{display:grid;place-items:center;padding:5px;background:var(--accent-soft);color:var(--accent);border-radius:5px}.card-body{padding:16px}.person-tabs{display:flex;gap:4px;flex-wrap:wrap;padding:3px;background:var(--surface-2);border:1px solid var(--border);border-radius:6px}.person-tabs button{border:0;background:none;color:var(--text-muted);font:500 12px var(--font);padding:6px 12px;border-radius:5px;cursor:pointer}.person-tabs button.active{background:var(--accent);color:var(--accent-text)}.identity{display:flex;gap:12px;align-items:center;margin:18px 0}.initials{width:42px;height:42px;display:grid;place-items:center;background:var(--accent);color:var(--accent-text);border-radius:50%;font-size:16px}.identity p,.address p{color:var(--text-muted);font-size:12px;margin-top:2px}strong{font-size:13px}dl{display:grid;grid-template-columns:1fr 1fr;gap:18px 12px;margin:0}dl>div{min-width:0}.wide{grid-column:1/-1}dt{font-size:10px;letter-spacing:.4px;text-transform:uppercase;color:var(--text-muted);font-weight:600;margin-bottom:4px}dd{margin:0;font-size:13px;overflow-wrap:anywhere}.accent,.amount{color:var(--accent)}.amount{font-weight:700;font-size:17px}.address{display:flex;align-items:center;gap:10px;padding:12px;background:var(--surface-2);border:1px solid var(--border);border-radius:6px;margin-bottom:18px}.address :global(svg){flex-shrink:0;color:var(--accent)}.rating-note{font-size:12px;color:var(--text-muted);border-top:1px solid var(--border);margin-top:18px;padding-top:14px;display:flex;align-items:center;gap:7px}.dot{width:6px;height:6px;border-radius:50%;background:var(--warning)}
</style>
