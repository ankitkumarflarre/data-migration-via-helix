<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type PartyView} from '../api';
 import {href,go} from '../routes';
 import {codes} from '../codes';
 import {us} from '../format';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let {party,mode}:{party:PartyView;mode:'correct'|'replace'}=$props();
 // svelte-ignore state_referenced_locally
 const row=party.rows.name.find(r=>r.current)??party.rows.name.at(-1);
 let f=$state<Record<string,string>>({...(row?.fields??{})}), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 // svelte-ignore state_referenced_locally
 const org=party.kind==='Organization';
 async function save(e:Event){e.preventDefault();if(!row)return;busy=true;error='';errors={};try{await billing.saveRow(party.id,'name',{version:party.version,mode,row_id:row.id,fields:Object.fromEntries(Object.entries(f).filter(([,v])=>v))});flash(mode==='replace'?`Name replaced from ${us(party.system_date)}.`:'Name corrected.');go('party/'+party.id);}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}}
</script>
<Page title={(mode==='replace'?'Replace':'Correct')+' Name & Contact Information'} date={false} required cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Name & contact" icon="user">
 <form class="b-stack" onsubmit={save}>
  <div class="b-form">
   {#if org}<div class="b-field" class:invalid={!!errors.organization}><label for="pn-org" class="b-star">Organization</label><input id="pn-org" bind:value={f.organization}/>{#if errors.organization}<span class="err">{errors.organization}</span>{/if}</div>
   {:else}
    <div class="b-field"><label for="pn-prefix">Prefix</label><select id="pn-prefix" bind:value={f.prefix}><option value="">(Select)</option>{#each codes.assumed.namePrefixes as p}<option>{p}</option>{/each}</select></div>
    <div class="b-field" class:invalid={!!errors.first}><label for="pn-first" class="b-star">First</label><input id="pn-first" bind:value={f.first}/>{#if errors.first}<span class="err">{errors.first}</span>{/if}</div>
    <div class="b-field"><label for="pn-mid">Middle</label><input id="pn-mid" bind:value={f.middle}/></div>
    <div class="b-field" class:invalid={!!errors.last}><label for="pn-last" class="b-star">Last</label><input id="pn-last" bind:value={f.last}/>{#if errors.last}<span class="err">{errors.last}</span>{/if}</div>
    <div class="b-field"><label for="pn-suffix">Suffix</label><select id="pn-suffix" bind:value={f.suffix}><option value="">(Select)</option>{#each codes.assumed.nameSuffixes as p}<option>{p}</option>{/each}</select></div>
   {/if}
   <div class="b-field" class:invalid={!!errors.correspondence}><label for="pn-corr" class="b-star">Correspondence Name</label><input id="pn-corr" bind:value={f.correspondence}/>{#if errors.correspondence}<span class="err">{errors.correspondence}</span>{/if}</div>
   <div class="b-field"><label for="pn-contact">Preferred method of contact</label><select id="pn-contact" bind:value={f.contact}><option value="">(Select)</option>{#each codes.assumed.contactMethods as c}<option>{c}</option>{/each}</select></div>
   <div class="b-field"><span class="b-label">Effective Date</span><output>{us(mode==='replace'?party.system_date:row?.effective)}</output></div>
   <div class="b-field"><span class="b-label">Expiration Date</span><output>12/31/9999</output></div>
   <div class="b-field full"><label for="pn-com">Comments</label><textarea id="pn-com" rows="3" bind:value={f.comments}></textarea></div>
  </div>
  <div class="b-actions"><a class="btn" href={href('party/'+party.id)}>Back</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy}>Save</button></div>
 </form>
 </Card>
</Page>

