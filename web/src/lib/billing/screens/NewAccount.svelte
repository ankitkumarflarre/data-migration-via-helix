<script lang="ts" module>
 // The form survives the trip to Search for a Person or Place and back.
 const draft=$state({payor:'',payorName:'',same:true,payee:'',payeeName:'',accountClass:'',dueDays:'1',endOfMonth:false});
 export function pickParty(role:'payor'|'payee',id:string,name:string){if(role==='payor'){draft.payor=id;draft.payorName=name;}else{draft.payee=id;draft.payeeName=name;}}
 export function resetDraft(){Object.assign(draft,{payor:'',payorName:'',same:true,payee:'',payeeName:'',accountClass:'',dueDays:'1',endOfMonth:false});}
</script>
<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import NewParty from './NewParty.svelte';
 let {query}:{query:URLSearchParams}=$props();
 let error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false), creating=$state<''|'payor'|'payee'>('');
 const due=$derived(Number(draft.dueDays));
 const ready=$derived(!!draft.payor&&(draft.same||!!draft.payee)&&!!draft.accountClass&&Number.isInteger(due)&&due>=1&&due<=31);
 async function create(){
  busy=true;error='';errors={};
  try{const a=await billing.createAccount({payor_id:draft.payor,payee_id:draft.same?draft.payor:draft.payee,same_party:draft.same,account_class:draft.accountClass,target_due_days:due,end_of_month:draft.endOfMonth});resetDraft();flash(`Account ${a.number} was created.`);go(acct(a.number));}
  catch(e){error=errorText(e);errors=fieldErrors(e);}finally{busy=false;}
 }
 // svelte-ignore state_referenced_locally
 if(query.get('reset')==='1')resetDraft();
</script>
<Page title="Create New Account" subtitle="Choose the payor and payee, then set the billing options." date={false} required cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Parties" subtitle="Who pays the bills on this account, and who receives refunds." icon="user">
  <div class="parties">
   <div class="party"><span class="b-label b-star">Account Payor</span>{#if draft.payor}<strong>{draft.payorName}</strong>{:else}<span class="b-note">Not selected</span>{/if}<div class="b-actions"><a class="btn sm" href={href('parties/search?for=payor')}>Search For An Existing Party</a><button class="btn sm" onclick={()=>creating='payor'}>Create A New Party</button></div>{#if errors.payor_id}<p class="b-err">{errors.payor_id}</p>{/if}</div>
   {#if !draft.same}<div class="party"><span class="b-label b-star">Account Payee</span>{#if draft.payee}<strong>{draft.payeeName}</strong>{:else}<span class="b-note">Not selected</span>{/if}<div class="b-actions"><a class="btn sm" href={href('parties/search?for=payee')}>Search For An Existing Party</a><button class="btn sm" onclick={()=>creating='payee'}>Create A New Party</button></div>{#if errors.payee_id}<p class="b-err">{errors.payee_id}</p>{/if}</div>{/if}
  </div>
  <label class="b-check"><input type="checkbox" bind:checked={draft.same}/>Payee and payor are the same party</label>
  {#if creating}<NewParty oncreated={(id,name)=>{pickParty(creating as 'payor'|'payee',id,name);creating='';}} oncancel={()=>creating=''}/>{/if}
 </Card>
 <Card title="Billing Options" icon="rules">
  <div class="b-form">
   <div class="b-field" class:invalid={!!errors.account_class}><label for="na-class" class="b-star">Bill Account Class</label><select id="na-class" bind:value={draft.accountClass}><option value="">(select)</option>{#each codes.accountClasses as c}<option>{c}</option>{/each}</select>{#if errors.account_class}<span class="err">{errors.account_class}</span>{/if}</div>
   <div class="b-field" class:invalid={!!errors.target_due_days}><label for="na-due" class="b-star">Target Due Days</label><input id="na-due" type="number" min="1" max="31" step="1" bind:value={draft.dueDays}/>{#if errors.target_due_days}<span class="err">{errors.target_due_days}</span>{/if}</div>
   <div class="b-field"><span class="b-label">End of month</span><label class="b-check"><input id="na-eom" type="checkbox" bind:checked={draft.endOfMonth}/>Set target due day to end of month</label></div>
  </div>
  <div class="b-actions"><a class="btn" href={href('')} onclick={resetDraft}>Cancel</a><span class="spacer"></span>{#if ready}<button class="btn primary" disabled={busy} onclick={create}>Create Account</button>{/if}</div>
 </Card>
</Page>
<style>.parties{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:12px}.party{display:grid;gap:8px;align-content:start;padding:14px;border:1px solid var(--border);border-radius:8px;background:var(--surface-2)}.party strong{font-size:15px}</style>
