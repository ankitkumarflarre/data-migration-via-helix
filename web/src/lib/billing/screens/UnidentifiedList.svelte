<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText,type PaymentView} from '../api';
 import {href} from '../routes';
 import {codes} from '../codes';
 import {usd,us,parseMoney} from '../format';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import Pager from '../ui/Pager.svelte';
 let rows=$state<PaymentView[]>([]), error=$state(''), page=$state(0);
 let after=$state(''), before=$state(''), on=$state(''), gt=$state(''), lt=$state(''), eq=$state(''), currency=$state(''), methods=$state<string[]>([]);
 onMount(async()=>{try{rows=await billing.unidentified();}catch(e){error=errorText(e);}});
 const matchesMethod=(m:string)=>methods.length===0||methods.some(f=>f===m||((f==='Credit Card'||f==='Debit Card')&&m==='Credit/Debit Card'));
 const filtered=$derived(rows.filter(p=>{const g=parseMoney(gt),l=parseMoney(lt),e=parseMoney(eq);return(!after||p.entry_date>after)&&(!before||p.entry_date<before)&&(!on||p.entry_date===on)&&(g===null||!gt||p.amount>g)&&(l===null||!lt||p.amount<l)&&(e===null||!eq||p.amount===e)&&(!currency||p.currency===currency)&&matchesMethod(p.method);}));
 const shown=$derived(filtered.slice(page*10,page*10+10));
 $effect(()=>{filtered.length;if(page*10>=filtered.length)page=0;});
 function clear(){after=before=on=gt=lt=eq=currency='';methods=[];}
</script>
<Page title="Unidentified Payments" subtitle="Payments that could not be matched to an account, policy or arrangement." date={false} cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Filter Unidentified Payments List" icon="filter">
  {#snippet tools()}<button class="btn sm" onclick={clear}>Clear Filters</button>{/snippet}
  <div class="b-form">
   <div class="b-field"><label for="ul-after">Entry date after</label><input id="ul-after" type="date" bind:value={after}/></div>
   <div class="b-field"><label for="ul-before">Entry date before</label><input id="ul-before" type="date" bind:value={before}/></div>
   <div class="b-field"><label for="ul-on">Entry date equal to</label><input id="ul-on" type="date" bind:value={on}/></div>
   <div class="b-field"><label for="ul-gt">Amount greater than</label><input id="ul-gt" inputmode="decimal" placeholder="$0.00" bind:value={gt}/></div>
   <div class="b-field"><label for="ul-lt">Amount less than</label><input id="ul-lt" inputmode="decimal" placeholder="$0.00" bind:value={lt}/></div>
   <div class="b-field"><label for="ul-eq">Amount equal to</label><input id="ul-eq" inputmode="decimal" placeholder="$0.00" bind:value={eq}/></div>
   <div class="b-field"><label for="ul-cur">Currency</label><select id="ul-cur" bind:value={currency}><option value="">(Select Currency)</option><option value="USD">US Dollars ($)</option></select></div>
   <fieldset class="b-field wide methods"><legend class="b-label">Payment Method</legend>{#each codes.paymentMethodFilters as m}<label class="b-check"><input type="checkbox" bind:group={methods} value={m}/>{m}</label>{/each}</fieldset>
  </div>
 </Card>
 <Card title="Unidentified payments" subtitle="Open Details to identify, re-reference or reverse a payment.">
  {#snippet tools()}<span class="b-badge info">{filtered.length} payments</span>{/snippet}
  <div class="b-grid"><table><thead><tr><th>Entry Date</th><th>User Id</th><th class="num">Amount</th><th>Currency</th><th>Payment Method</th><th>Identifier Type</th><th>Identifier</th><th></th></tr></thead><tbody>
   {#each shown as p (p.id)}<tr><td>{us(p.entry_date)}</td><td>{p.created_by}</td><td class="num">{usd(p.amount)}</td><td>US Dollars ($)</td><td>{p.method}</td><td>{p.identifier_type}</td><td>{p.identifier}</td><td><a href={href('unidentified/'+p.id)}>Details</a></td></tr>{:else}<tr><td colspan="8" class="empty">No unidentified payments.</td></tr>{/each}
  </tbody><Pager total={filtered.length} bind:page cols={8}/></table></div>
  <div class="b-actions"><a class="btn" href={href('')}>Close</a></div>
 </Card>
</Page>
<style>
 .methods{border:0;margin:0;padding:0;display:flex;flex-wrap:wrap;gap:8px 18px;align-items:center}.methods legend{width:100%;margin-bottom:6px}
</style>
