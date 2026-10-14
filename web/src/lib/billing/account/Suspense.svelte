<script lang="ts">
 import Card from '../ui/Card.svelte';
 import Status from '../ui/Status.svelte';
 import {billing,errorText,type AccountView} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {usd,us} from '../format';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let {view,reload}:{view:AccountView;reload:()=>void}=$props();
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 // svelte-ignore state_referenced_locally
 let picked=$state<string[]>(view.suspense.map(s=>s.id)), action=$state(''), error=$state(''), busy=$state(false), open=$state('');
 const live=['Disburse','Place Hold','Release Hold'];
 async function submit(){
  error='';
  if(action==='Disburse'){go(acct(n,'disburse')+'?ids='+picked.join(','));return;}
  busy=true;
  try{const m=await billing.suspenseAction(n,{version:view.account.version,suspense_ids:picked,action});flash(m.message);reload();go(acct(n,'suspense'));}catch(e){error=errorText(e);}finally{busy=false;}
 }
 async function decide(id:string,approve:boolean){busy=true;error='';try{await billing.decide(n,id,view.account.version,approve);flash(approve?'Disbursement authorized.':'Disbursement rejected. The funds were returned to suspense.');reload();go(acct(n,'suspense'));}catch(e){error=errorText(e);}finally{busy=false;}}
</script>
<Page title="Items In Suspense" subtitle="Select items, then choose an action to apply to them." note cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Suspense items" icon="database">
  {#snippet tools()}
   <label class="sr" for="su-action">Action</label>
   <select id="su-action" bind:value={action} disabled={!view.suspense.length}><option value="">Select An Action</option>{#each codes.suspenseActions as a}<option value={a} disabled={!live.includes(a)}>{a}{live.includes(a)?'':' (not specified)'}</option>{/each}</select>
   <button class="btn sm primary" disabled={busy||!action||!picked.length} onclick={submit}>Submit</button>
  {/snippet}
  <div class="b-grid"><table><thead><tr><th><input type="checkbox" aria-label="Select all items" disabled={!view.suspense.length} checked={view.suspense.length>0&&picked.length===view.suspense.length} onchange={e=>picked=e.currentTarget.checked?view.suspense.map(s=>s.id):[]}/></th><th>Payment Source</th><th>Refund Source</th><th>Reference</th><th>Suspense Date</th><th>Suspense Reason</th><th class="num">Amount</th><th>Status</th></tr></thead><tbody>
   {#each view.suspense as s (s.id)}<tr><td><input type="checkbox" aria-label={'Select '+usd(s.amount)+' from '+us(s.date)} bind:group={picked} value={s.id}/></td><td>{s.payment_source}</td><td>{s.refund_source}</td><td>{s.reference}</td><td>{us(s.date)}</td><td>{s.reason}</td><td class="num">{usd(s.amount)}</td><td><Status value={s.status}/></td></tr>{:else}<tr><td colspan="8" class="empty">No items in suspense.</td></tr>{/each}
  </tbody></table></div>
 </Card>
 {#if view.disbursements.length}
  <Card title="Disbursements" subtitle="Refund requests made from suspense." icon="transfer">
   <div class="b-grid"><table><thead><tr><th>Date Requested</th><th class="num">Amount</th><th>Status</th><th></th></tr></thead><tbody>
    {#each view.disbursements as d (d.id)}
     <tr><td>{us(d.request_date)}</td><td class="num">{usd(d.amount)}</td><td><Status value={d.status}/></td><td><button class="b-link" aria-expanded={open===d.id} onclick={()=>open=open===d.id?'':d.id}>View Details</button></td></tr>
     {#if open===d.id}<tr class="desc"><td colspan="4">
      Payee: {d.payee_name} · Method: {d.method} · Type: {d.type} · Reason: {d.reason} · Group: {d.group} · Requested by {d.user}{#if d.authorized_by} · Authorized by {d.authorized_by} on {us(d.authorized_on)}{/if}{#if d.issued_on} · Issued {us(d.issued_on)}{/if}
      {#if d.status==='Pending Authorization'}<div class="b-actions" style="margin-top:8px"><span class="b-note">Above the auto-authorization limit of {usd(view.auto_authorize_limit)}; an approver decides.</span><button class="btn sm" disabled={busy} onclick={()=>decide(d.id,false)}>Reject</button><button class="btn sm primary" disabled={busy} onclick={()=>decide(d.id,true)}>Authorize</button></div>{/if}
     </td></tr>{/if}
    {/each}
   </tbody></table></div>
  </Card>
 {/if}
</Page>
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}select{min-height:34px;font-size:12px;padding:4px 8px}</style>
