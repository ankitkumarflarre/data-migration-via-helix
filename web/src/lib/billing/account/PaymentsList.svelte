<script lang="ts">
 import Card from '../ui/Card.svelte';
 import Status from '../ui/Status.svelte';
 import type {AccountView} from '../api';
 import {href,acct} from '../routes';
 import {codes} from '../codes';
 import {usd,created} from '../format';
 import Page from '../ui/Page.svelte';
 import Pager from '../ui/Pager.svelte';
 let {view}:{view:AccountView}=$props();
 let status=$state(''), method=$state(''), page=$state(0);
 const rows=$derived(view.payments.filter(p=>(!status||p.status===status)&&(!method||p.method===method)));
</script>
<Page title="Payments List" subtitle="Select the Amount link for more detail." date={false} cards>
 <Card title="Payments" icon="dollar">
  {#snippet tools()}
   <label class="sr" for="pl-status">Status</label><select id="pl-status" bind:value={status}><option value="">All statuses</option>{#each codes.paymentStatuses as s}<option>{s}</option>{/each}</select>
   <label class="sr" for="pl-method">Payment Method</label><select id="pl-method" bind:value={method}><option value="">All methods</option>{#each codes.paymentMethods as m}<option>{m}</option>{/each}</select>
  {/snippet}
  <div class="b-grid"><table><thead><tr><th>Amount</th><th>Created</th><th>Payment Method</th><th>Reference #</th><th>Status</th><th>Created By</th></tr></thead><tbody>
   {#each rows.slice(page*10,page*10+10) as p (p.id)}<tr><td><a href={href('payment/'+p.id)}>{usd(p.amount)}</a></td><td>{created(p.created_at)}</td><td>{p.method}</td><td>{p.reference??p.transaction_number??''}</td><td><Status value={p.status}/></td><td>{p.created_by}</td></tr>{:else}<tr><td colspan="6" class="empty">No payments.</td></tr>{/each}
  </tbody><Pager total={rows.length} bind:page cols={6}/></table></div>
  <div class="b-actions"><a class="btn" href={href(acct(view.account.number))}>Exit Payment Processing</a></div>
 </Card>
</Page>
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}select{min-height:34px;font-size:12px;padding:4px 8px}</style>