<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type AccountView,type PaymentResult} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {usd,us,parseMoney,moneyText} from '../format';
 import {flash,flashLinked,today} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import CardGateway from './CardGateway.svelte';
 import AllocationGrid from './AllocationGrid.svelte';
 let {view,reload}:{view:AccountView;reload:()=>void}=$props();
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 let method=$state(''), entry=$state(today()), check=$state(''), nameOnCheck=$state(''), postmark=$state(today());
 // svelte-ignore state_referenced_locally
 let mode=$state<'account'|'policies'>('account'), amountText=$state(usd(view.ledger.outstanding)), alloc=$state<Record<string,string>>({});
 let error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false), gateway=$state(false);
 const amount=$derived(parseMoney(amountText));
 const allocations=$derived(Object.entries(alloc).map(([term_id,v])=>({term_id,amount:parseMoney(v)??0})).filter(a=>a.amount>0));
 const valid=$derived(!!method&&amount!==null&&amount>0);
 function body(){return {version:view.account.version,method,amount,entry_date:entry,check_number:method==='Check'?check:'',name_on_check:method==='Check'?nameOnCheck:'',postmark_date:method==='Check'?postmark:'',allocation:mode,allocations:mode==='policies'?allocations:[]};}
 async function submit(e:Event){
  e.preventDefault();error='';errors={};
  if(!valid){errors={amount:amount===null?'Enter a payment amount.':'',method:method?'':'Select a payment method.'};return;}
  if(method==='Credit/Debit Card'){gateway=true;return;}
  busy=true;
  try{const r=await billing.pay(n,body());flashLinked(r.message,n,href(acct(n)));go('');}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}
 }
 async function payByCard(card:Record<string,unknown>):Promise<PaymentResult>{return billing.pay(n,{...body(),card});}
 function finished(r:PaymentResult){gateway=false;flash(r.message);reload();go(acct(n));}
</script>
<Page title="Make A Payment" subtitle="Record a payment and choose how it is allocated." date={false} cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <form class="b-stack" onsubmit={submit}>
  <Card title="Payment Details" icon="dollar">
   <div class="b-form">
    <div class="b-field" class:invalid={!!errors.method}><label for="mp-method">Payment Method</label><select id="mp-method" bind:value={method}><option value="">(Select)</option>{#each codes.paymentMethods as m}<option>{m}</option>{/each}</select>{#if errors.method}<span class="err">{errors.method}</span>{/if}</div>
    <div class="b-field" class:invalid={!!errors.entry_date}><label for="mp-entry">Entry Date</label><input id="mp-entry" type="date" bind:value={entry} max={today()}/>{#if errors.entry_date}<span class="err">{errors.entry_date}</span>{/if}</div>
    {#if method!=='Credit/Debit Card'}
     <div class="b-field"><label for="mp-check">Check Number</label><input id="mp-check" bind:value={check} maxlength="30" disabled={method!=='Check'&&method!==''}/></div>
     <div class="b-field"><label for="mp-name">Name On Check</label><input id="mp-name" bind:value={nameOnCheck} maxlength="100" disabled={method!=='Check'&&method!==''}/></div>
     <div class="b-field" class:invalid={!!errors.postmark_date}><label for="mp-post">Postmark Date</label><input id="mp-post" type="date" bind:value={postmark} max={today()} disabled={method!=='Check'&&method!==''}/>{#if errors.postmark_date}<span class="err">{errors.postmark_date}</span>{/if}</div>
    {/if}
   </div>
  </Card>
  <Card title="Payment Allocation" subtitle="Apply the payment to specific policies, or to the account as a whole." icon="transfer">
   <fieldset class="b-radios"><legend class="sr">Payment allocation</legend><label><input type="radio" bind:group={mode} value="policies"/>Pay Specific Policies / Arrangements</label><label><input type="radio" bind:group={mode} value="account"/>Pay To Account</label></fieldset>
   {#if mode==='account'}
    <div class="b-form">
     <div class="b-field" class:invalid={!!errors.amount}><div class="b-field-head"><label for="mp-amt">Payment Amount</label>{#if method!=='Credit/Debit Card'}<button type="button" class="b-link" onclick={()=>amountText=usd(view.ledger.outstanding)}>Outstanding Balance {usd(view.ledger.outstanding)}</button>{/if}</div><input id="mp-amt" inputmode="decimal" placeholder="$0.00" bind:value={amountText}/>{#if errors.amount}<span class="err">{errors.amount}</span>{/if}</div>
    </div>
   {:else}
    <div class="b-form">
     <div class="b-field" class:invalid={!!errors.amount}><label for="mp-amt2">Payment Amount</label><input id="mp-amt2" inputmode="decimal" placeholder="$0.00" bind:value={amountText}/>{#if errors.amount}<span class="err">{errors.amount}</span>{/if}</div>
    </div>
    <AllocationGrid {view} total={amount??0} bind:alloc {errors}/>
   {/if}
   <div class="b-actions"><a class="btn" href={href(acct(n))}>Exit Payment Processing</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!valid}>Submit Payment</button></div>
  </Card>
 </form>
</Page>
{#if gateway&&amount!==null}<CardGateway {view} {amount} pay={payByCard} onfinish={finished} oncancel={()=>gateway=false}/>{/if}
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>
