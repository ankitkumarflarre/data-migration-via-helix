<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type PartyView} from '../api';
 import {href,go} from '../routes';
 import {codes} from '../codes';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let {party}:{party:PartyView}=$props();
 // svelte-ignore state_referenced_locally
 let f=$state<Record<string,string>>({us_citizen:'Yes',dependents:'0',...party.demographics}), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 const a=codes.assumed;
 const selects:[string,string,string[]][]=[['language','Preferred Language',a.languages],['gender','Gender',a.genders],['ethnic_origin','Ethnic Origin',a.ethnicOrigins],['marital_status','Marital Status',a.maritalStatuses],['other_id','Other ID',a.otherIds],['profession','Profession',a.professions]];
 async function save(e:Event){e.preventDefault();busy=true;error='';errors={};try{await billing.saveDemographics(party.id,party.version,Object.fromEntries(Object.entries(f).filter(([,v])=>v!=='')));flash('Demographics saved.');go('party/'+party.id);}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}}
</script>
<Page title="Demographics" date={false} cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Demographics" icon="user">
 <form class="b-stack" onsubmit={save}>
  <div class="b-form">
   {#each selects as [k,label,opts]}<div class="b-field"><label for={'dm-'+k}>{label}</label><select id={'dm-'+k} bind:value={f[k]}><option value="">(Select)</option>{#each opts as o}<option>{o}</option>{/each}</select></div>{/each}
   <div class="b-field" class:invalid={!!errors.dob}><label for="dm-dob">DOB</label><input id="dm-dob" type="date" bind:value={f.dob}/>{#if errors.dob}<span class="err">{errors.dob}</span>{/if}</div>
   <div class="b-field"><label for="dm-occ">Occupation</label><input id="dm-occ" bind:value={f.occupation}/></div>
   <div class="b-field" class:invalid={!!errors.ssn}><label for="dm-ssn">Social Security #</label><input id="dm-ssn" autocomplete="off" bind:value={f.ssn}/>{#if errors.ssn}<span class="err">{errors.ssn}</span>{/if}</div>
   <div class="b-field"><span class="b-label" id="dm-cit">US Citizen?</span><span class="b-seg" role="radiogroup" aria-labelledby="dm-cit"><label><input type="radio" bind:group={f.us_citizen} value="No"/>No</label><label><input type="radio" bind:group={f.us_citizen} value="Yes"/>Yes</label></span></div>
   <div class="b-field" class:invalid={!!errors.date_of_death}><label for="dm-dod">Date of Death</label><input id="dm-dod" type="date" bind:value={f.date_of_death}/>{#if errors.date_of_death}<span class="err">{errors.date_of_death}</span>{/if}</div>
   <div class="b-field" class:invalid={!!errors.dependents}><label for="dm-dep">Number Of Dependents</label><input id="dm-dep" inputmode="numeric" bind:value={f.dependents}/>{#if errors.dependents}<span class="err">{errors.dependents}</span>{/if}</div>
   <div class="b-field"><span class="b-label">Company Personnel</span><label class="b-check"><input id="dm-cp" type="checkbox" checked={f.company_personnel==='Yes'} onchange={e=>f.company_personnel=e.currentTarget.checked?'Yes':'No'}/>Company personnel</label></div>
   <div class="b-field"><span class="b-label">OFAC Check Not Required</span><output>Unanswered</output></div>
   <div class="b-field full"><label for="dm-com">Party Comments</label><textarea id="dm-com" rows="3" bind:value={f.comments}></textarea></div>
  </div>
  <div class="b-actions"><a class="btn" href={href('party/'+party.id)}>Back</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy}>Save</button></div>
 </form>
 </Card>
</Page>

