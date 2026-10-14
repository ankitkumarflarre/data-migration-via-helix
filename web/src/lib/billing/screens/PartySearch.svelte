<script lang="ts">
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type PartyHit} from '../api';
 import {href,go,acct} from '../routes';
 import {codes} from '../codes';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import NewParty from './NewParty.svelte';
 import {pickParty} from './NewAccount.svelte';
 let {query}:{query:URLSearchParams}=$props();
 // for=payor|payee (Create New Account) or change-payee|change-payor (account side panel)
 // svelte-ignore state_referenced_locally
 const purpose=query.get('for')??'', account=query.get('account')??'';
 const back=$derived(purpose==='disburse-payee'?href(acct(account,'disburse')+'?ids='+(query.get('ids')??'')):purpose.startsWith('change')?href(acct(account)):purpose?href('accounts/new'):href(''));
 let f=$state({target:'',by:'',last:'',first:'',middle:'',address:'',city:'',state:'',zip:'',license:'',phone:'',membership:'',phonetic:false});
 let hits=$state<PartyHit[]|null>(null), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false), adding=$state(false);
 const person=$derived(f.target==='Person');
 const ready=$derived(!!f.target&&!!f.by&&(
  (f.by==='Name and Address'||f.by==='Other')?!!f.last.trim()&&!!f.state:f.by==='License Number'?!!f.license.trim():f.by==='Phone Number'?/^\d{3}-\d{3}-\d{4}$/.test(f.phone):!!f.membership.trim()));
 async function search(e:Event){e.preventDefault();busy=true;error='';errors={};try{hits=await billing.searchParties(f);}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}}
 async function select(id:string,name:string){
  if(purpose==='payor'||purpose==='payee'){pickParty(purpose,id,name);go('accounts/new');return;}
  if(purpose==='disburse-payee'){go(acct(account,'disburse')+`?ids=${query.get('ids')??''}&payee=${id}`);return;}
  if(purpose.startsWith('change-')){
   const role=purpose.slice(7);busy=true;error='';
   try{const v=await billing.account(account);await billing.changeParty(account,{version:v.account.version,role,party_id:id});flash(`The account ${role} was changed to ${name}.`);go(acct(account));}catch(err){error=errorText(err);}finally{busy=false;}
   return;
  }
  go('party/'+id);
 }
</script>
<div class="narrow">
 {#if hits}
  <Page title="Search Results" subtitle={`We found ${hits.length} ${hits.length===1?'party':'parties'} that matched your search. Select a party or choose an option below.`} date={false} cards>
   {#if error}<Banner kind="error">{error}</Banner>{/if}
   <Card title="Existing Parties" icon="user">
    <div class="b-grid"><table><thead><tr><th>Name</th><th>Address</th><th>DOB</th><th>Entity Name</th><th>Party Type</th><th></th><th></th><th></th></tr></thead><tbody>
     {#each hits as h (h.id)}<tr><td title={h.name}>{h.name}</td><td title={h.address}>{h.address.length>24?h.address.slice(0,22)+'…':h.address}</td><td>{h.dob}</td><td>{h.entity_name}</td><td><span class="b-badge">{h.party_type}</span></td><td><a href={href('party/'+h.id)}>View</a></td><td><button class="b-link" disabled={busy} onclick={()=>select(h.id,h.name)}>Select</button></td><td><a href={href('party/'+h.id)} title="Open the party's 360 view">360</a></td></tr>{:else}<tr><td colspan="8" class="empty">No parties matched your search.</td></tr>{/each}
    </tbody></table></div>
    {#if adding}<NewParty kind={person?'Person':'Organization'} oncreated={(id,name)=>{adding=false;select(id,name);}} oncancel={()=>adding=false}/>{/if}
    <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span><button class="btn" onclick={()=>hits=null}>Revise Search</button><button class="btn primary" onclick={()=>adding=true}>Add New</button></div>
   </Card>
  </Page>
 {:else}
  <Page title="Search for a Person or Place" subtitle="Please enter all required (*) information." date={false} required cards>
   {#if error}<Banner kind="error">{error}</Banner>{/if}
   <Card title="Search criteria" icon="search">
    <form class="b-stack" onsubmit={search}>
     <div class="b-form">
      <div class="b-field"><label for="ps-target" class="b-star">I'm looking for a/an</label><select id="ps-target" bind:value={f.target}><option value=""></option>{#each codes.partySearchTargets as t}<option>{t}</option>{/each}</select></div>
      <div class="b-field"><label for="ps-by" class="b-star">I want to search by</label><select id="ps-by" bind:value={f.by}><option value=""></option>{#each codes.partySearchBy as b}<option>{b}</option>{/each}</select></div>
     </div>
     {#if f.by==='Name and Address'||f.by==='Other'}
      <div class="b-form">
       <div class="b-field" class:invalid={!!errors.last}><label for="ps-last" class="b-star">{person?'Last Name':'Organization or Last Name'}</label><input id="ps-last" bind:value={f.last}/>{#if errors.last}<span class="err">{errors.last}</span>{/if}</div>
       {#if person}<div class="b-field"><label for="ps-first">First Name</label><input id="ps-first" bind:value={f.first}/></div><div class="b-field"><label for="ps-mid">Middle Name</label><input id="ps-mid" bind:value={f.middle}/></div>{/if}
       <div class="b-field"><label for="ps-addr">Address</label><input id="ps-addr" bind:value={f.address}/></div>
       <div class="b-field"><label for="ps-city">City</label><input id="ps-city" bind:value={f.city}/></div>
       <div class="b-field" class:invalid={!!errors.state}><label for="ps-state" class="b-star">State</label><select id="ps-state" bind:value={f.state}><option value=""></option>{#each codes.states as s}<option>{s}</option>{/each}</select>{#if errors.state}<span class="err">{errors.state}</span>{/if}</div>
       <div class="b-field"><label for="ps-zip">Zip Code</label><input id="ps-zip" bind:value={f.zip} inputmode="numeric" maxlength="5"/></div>
       <div class="b-field"><span class="b-label">Matching</span><label class="b-check"><input id="ps-ph" type="checkbox" bind:checked={f.phonetic}/>Use phonetic search</label></div>
      </div>
     {:else if f.by==='License Number'}<div class="b-form"><div class="b-field"><label for="ps-lic" class="b-star">License Number</label><input id="ps-lic" bind:value={f.license}/></div></div>
     {:else if f.by==='Phone Number'}<div class="b-form"><div class="b-field" class:invalid={!!errors.phone}><label for="ps-phone" class="b-star">Phone Number</label><input id="ps-phone" bind:value={f.phone} placeholder="XXX-XXX-XXXX"/>{#if errors.phone}<span class="err">{errors.phone}</span>{/if}</div></div>
     {:else if f.by==='Membership ID'}<div class="b-form"><div class="b-field"><label for="ps-mem" class="b-star">Membership ID</label><input id="ps-mem" bind:value={f.membership}/></div></div>{/if}
     <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!ready}><Icon name="search" size={14}/>Search</button></div>
    </form>
   </Card>
  </Page>
 {/if}
</div>
<style>.narrow{min-width:0}</style>
