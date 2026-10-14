<script lang="ts">
 import Card from '../ui/Card.svelte';
 import Status from '../ui/Status.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText,type PaymentView} from '../api';
 import {href,acct} from '../routes';
 import {codes} from '../codes';
 import {usd,us} from '../format';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import PaymentStrip from './PaymentStrip.svelte';
 let {id}:{id:string}=$props();
 let p=$state<PaymentView|null>(null), error=$state(''), reason=$state(codes.reverseReasons[0]), busy=$state(false), confirm=$state(false);
 onMount(load);
 async function load(){try{p=await billing.payment(id);}catch(e){error=errorText(e);}}
 async function reverse(){if(!p)return;busy=true;error='';try{await billing.reverse(p.id,p.version,reason);flash(`${usd(p.amount)} Payment was reversed. Reason: ${reason.replace(/^RP-/,'')}`);confirm=false;await load();}catch(e){error=errorText(e);}finally{busy=false;}}
</script>
<Page title="Payment Details" cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 {#if p}
  <Card title="Payment" icon="dollar">{#snippet tools()}<Status value={p?.status??''}/>{/snippet}<PaymentStrip payment={p}/></Card>
  <Card title="Allocations" subtitle="Where this payment was applied." icon="transfer">
   <div class="b-grid"><table><thead><tr><th>Account #</th><th>Direction</th><th>Transaction Date</th><th>Status</th><th>Allocation Type</th><th class="num">Amount</th></tr></thead><tbody>
    {#each p.allocations as a (a.id)}<tr><td><a href={href(acct(a.account_number))}>{a.account_number}</a></td><td>{a.direction}</td><td>{us(a.date)}</td><td><Status value={a.status}/></td><td>{a.type}</td><td class="num">{usd(a.amount)}</td></tr>{:else}<tr><td colspan="6" class="empty">{p.unidentified?'This payment is unidentified and has not been allocated.':'No allocations.'}</td></tr>{/each}
   </tbody></table></div>
  </Card>
  {#if p.status==='Active'&&!p.unidentified}
   <Card title="Reverse This Payment" subtitle="What it paid becomes due again." icon="transfer">
    <div class="b-form"><div class="b-field"><label for="pd-reason">Reverse reason</label><select id="pd-reason" bind:value={reason}>{#each codes.reverseReasons as r}<option>{r}</option>{/each}</select></div></div>
    <div class="b-actions">{#if confirm}<span>Reverse {usd(p.amount)}? What it paid becomes due again.</span><span class="spacer"></span><button class="btn sm" onclick={()=>confirm=false}>Keep</button><button class="btn danger sm" disabled={busy} onclick={reverse}>Reverse Payment</button>{:else}<span class="spacer"></span><button class="btn danger sm" onclick={()=>confirm=true}>Reverse Payment</button>{/if}</div>
   </Card>
  {/if}
 {/if}
 <div class="b-bar"><button class="btn" onclick={()=>history.back()}>Cancel</button>{#if p?.unidentified}<a class="btn" href={href('unidentified/'+p.id)}>Unidentified Payment</a>{/if}</div>
</Page>