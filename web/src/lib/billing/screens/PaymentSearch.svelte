<script lang="ts">
 import Status from '../ui/Status.svelte';
 import {billing,errorText,fieldErrors,type PaymentView} from '../api';
 import {href,go} from '../routes';
 import {codes} from '../codes';
 import {usd,us,parseMoney} from '../format';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import Card from '../ui/Card.svelte';
 import Field from '../ui/Field.svelte';
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 let f=$state({account:'',policy:'',dateMode:'range',from:'',to:'',on:'',amountMode:'range',min:'',max:'',exact:'',method:'',currency:'',other:''});
 let results=$state<PaymentView[]|null>(null), error=$state(''), errors=$state<Record<string,string>>({});
 // Search appears once at least one piece of information is entered.
 const any=$derived([f.account,f.policy,f.from,f.to,f.on,f.min,f.max,f.exact,f.method,f.currency,f.other].some(v=>v.trim()!==''));
 async function search(e:Event){
  e.preventDefault();error='';errors={};results=null;
  const q:Record<string,string>={};
  if(f.account.trim())q.account=f.account.trim();if(f.policy.trim())q.policy=f.policy.trim();
  if(f.dateMode==='exact'&&f.on){q.from=f.on;q.to=f.on;}else{if(f.from)q.from=f.from;if(f.to)q.to=f.to;}
  const money=(k:string,v:string)=>{if(!v.trim())return;const c=parseMoney(v);if(c===null)errors={...errors,[k]:'Enter an amount.'};else q[k]=String(c);};
  if(f.amountMode==='exact'){money('min',f.exact);if(q.min)q.max=q.min;}else{money('min',f.min);money('max',f.max);}
  if(f.method)q.method=f.method;if(f.currency)q.currency=f.currency;if(f.other.trim())q.other=f.other.trim();
  if(Object.keys(errors).length)return;
  try{const r=await billing.searchPayments(q);if(r.length===1)go('payment/'+r[0].id);else results=r;}catch(err){error=errorText(err);errors=fieldErrors(err);}
 }
</script>
<Page title="Search For A Payment" subtitle="Find all payments associated by account, policy, date, amount or method." date={false} cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Search criteria" subtitle="Enter at least one piece of information." icon="filter">
  <form class="b-stack" onsubmit={search}>
   <div class="b-form">
    <Field label="Account #" id="ps-acct"><input id="ps-acct" bind:value={f.account} autocomplete="off"/></Field>
    <Field label="Policy / Arrangement #" id="ps-pol"><input id="ps-pol" bind:value={f.policy} autocomplete="off"/></Field>
    <div class="b-field"><label for="ps-method">Payment Method</label><select id="ps-method" bind:value={f.method}><option value="">(Select Method)</option>{#each codes.paymentMethodFilters as m}<option>{m}</option>{/each}</select></div>
    <div class="b-field wide">
     <div class="b-field-head"><span class="b-label" id="ps-date-l">Date</span><span class="b-seg" role="radiogroup" aria-labelledby="ps-date-l"><label><input type="radio" bind:group={f.dateMode} value="range"/>Range</label><label><input type="radio" bind:group={f.dateMode} value="exact"/>Exact</label></span></div>
     {#if f.dateMode==='range'}<div class="b-range"><input type="date" aria-label="Date from" bind:value={f.from}/><span>to</span><input type="date" aria-label="Date to" bind:value={f.to}/></div>{:else}<input type="date" aria-label="Date" bind:value={f.on}/>{/if}
    </div>
    <div class="b-field wide" class:invalid={!!errors.min||!!errors.max}>
     <div class="b-field-head"><span class="b-label" id="ps-amt-l">Amount</span><span class="b-seg" role="radiogroup" aria-labelledby="ps-amt-l"><label><input type="radio" bind:group={f.amountMode} value="range"/>Range</label><label><input type="radio" bind:group={f.amountMode} value="exact"/>Exact</label></span></div>
     {#if f.amountMode==='range'}<div class="b-range"><input aria-label="Amount from" inputmode="decimal" placeholder="$0.00" class:invalid={!!errors.min} bind:value={f.min}/><span>to</span><input aria-label="Amount to" inputmode="decimal" placeholder="$0.00" class:invalid={!!errors.max} bind:value={f.max}/></div>{:else}<input aria-label="Amount" inputmode="decimal" placeholder="$0.00" class:invalid={!!errors.min} bind:value={f.exact}/>{/if}
     {#if errors.min||errors.max}<span class="err">{errors.min||errors.max}</span>{/if}
    </div>
    <div class="b-field"><label for="ps-cur">Currency</label><select id="ps-cur" bind:value={f.currency}><option value="">(Select Currency)</option><option value="USD">US Dollars ($)</option></select></div>
    <div class="b-field wide"><label for="ps-other">Other Info</label><input id="ps-other" bind:value={f.other}/><small class="b-note">Example: First Name, Last Name, Check Number or Card Number</small></div>
   </div>
   {#if errors.query}<p class="b-err" role="alert">{errors.query}</p>{/if}
   <div class="b-actions"><a class="btn" href={href('')}>Cancel</a><span class="spacer"></span>{#if any}<button class="btn primary" type="submit"><Icon name="search" size={14}/>Search</button>{/if}</div>
  </form>
 </Card>
 {#if results}
  <Card title="Payments" subtitle="We found these payments that match your search criteria. Select the Amount link for more detail.">
   {#snippet tools()}<span class="b-badge info">{results?.length??0} found</span>{/snippet}
   <div class="b-grid"><table><thead><tr><th>Amount</th><th>Date Entered</th><th>Account #</th><th>Payment Method</th><th>Status</th><th>Created By</th></tr></thead><tbody>
    {#each results as p (p.id)}<tr><td><a href={href('payment/'+p.id)}>{usd(p.amount)}</a></td><td>{us(p.entry_date)}</td><td>{p.account_number||'Unidentified'}</td><td>{p.method}</td><td><Status value={p.status}/></td><td>{p.created_by}</td></tr>{:else}<tr><td colspan="6" class="empty">No payments match your search criteria.</td></tr>{/each}
   </tbody></table></div>
  </Card>
 {/if}
</Page>
