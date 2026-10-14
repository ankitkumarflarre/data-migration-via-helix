<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type AccountView} from '../api';
 import {href,acct,go} from '../routes';
 import {usd,us,parseMoney} from '../format';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import AllocationGrid from './AllocationGrid.svelte';
 let {view,reload}:{view:AccountView;reload:()=>void}=$props();
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 // svelte-ignore state_referenced_locally
 const free=view.suspense.filter(s=>s.status==='No Hold');
 let picked=$state<string[]>([]), calculated=$state(0), mode=$state<'account'|'policies'>('account'), alloc=$state<Record<string,string>>({}), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 const selectedTotal=$derived(free.filter(s=>picked.includes(s.id)).reduce((t,s)=>t+s.amount,0));
 const allocations=$derived(Object.entries(alloc).map(([term_id,v])=>({term_id,amount:parseMoney(v)??0})).filter(a=>a.amount>0));
 async function submit(){busy=true;error='';errors={};try{const m=await billing.allocateSuspense(n,{version:view.account.version,suspense_ids:picked,allocation:mode,allocations:mode==='policies'?allocations:[]});flash(m.message);reload();go(acct(n));}catch(e){error=errorText(e);errors=fieldErrors(e);}finally{busy=false;}}
</script>
<Page title="Suspense Allocation" subtitle="Apply money held in suspense to policies or to the account." date={false} cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Suspense items" icon="database">
  {#snippet tools()}<button class="btn sm" disabled={!free.length} onclick={()=>picked=free.map(s=>s.id)}>Select All</button><button class="btn sm" onclick={()=>calculated=selectedTotal}>Calculate Total Suspense Selected</button>{/snippet}
  {#if free.length}
   <div class="b-grid"><table><thead><tr><th><span class="sr">Select</span></th><th>Payment Source</th><th>Suspense Date</th><th>Suspense Reason</th><th class="num">Amount</th></tr></thead><tbody>
    {#each free as s (s.id)}<tr><td><input type="checkbox" aria-label={'Select '+usd(s.amount)} bind:group={picked} value={s.id}/></td><td>{s.payment_source}</td><td>{us(s.date)}</td><td>{s.reason}</td><td class="num">{usd(s.amount)}</td></tr>{/each}
   </tbody></table></div>
  {:else}<p class="b-note">No items are free to allocate.</p>{/if}
  <div class="b-kpis">
   <div class="b-kpi"><span class="b-ico soft">$</span><span class="b-kpi-body"><span class="b-kpi-top"><span class="b-kpi-label">Total Suspense Selected</span><strong>{usd(calculated)}</strong></span></span></div>
   <div class="b-kpi"><span class="b-ico warning">$</span><span class="b-kpi-body"><span class="b-kpi-top"><span class="b-kpi-label">Total Suspense Remaining</span><strong>{usd(view.ledger.in_suspense-calculated)}</strong></span></span></div>
  </div>
 </Card>
 <Card title="Suspense Allocation" icon="transfer">
  <fieldset class="b-radios"><legend class="sr">Allocation</legend><label><input type="radio" bind:group={mode} value="policies"/>Pay Specific Policies / Arrangements</label><label><input type="radio" bind:group={mode} value="account"/>Pay To Account</label></fieldset>
  {#if mode==='account'}<div class="b-form"><div class="b-field"><span class="b-label">Payment Amount</span><span class="b-ro">{usd(calculated)}</span></div></div>
  {:else}<AllocationGrid {view} total={calculated} bind:alloc {errors}/>{/if}
  {#if errors.suspense_ids}<p class="b-err">{errors.suspense_ids}</p>{/if}
  <div class="b-actions"><a class="btn" href={href(acct(n))}>Exit Payment Processing</a><span class="spacer"></span>{#if calculated>0&&calculated===selectedTotal}<button class="btn primary" disabled={busy} onclick={submit}>Submit Payment</button>{/if}</div>
 </Card>
</Page>
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>
