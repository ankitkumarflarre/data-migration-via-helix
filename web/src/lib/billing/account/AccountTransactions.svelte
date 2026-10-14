<script lang="ts">
 import Card from '../ui/Card.svelte';
 import type {AccountView} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import Page from '../ui/Page.svelte';
 let {view}:{view:AccountView}=$props();
 let tx=$state('');
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
</script>
<Page title="Billing Transactions" subtitle="Choose the billing transaction to add to this account." date={false} required cards>
 <Card title="Transaction" icon="transfer">
  <div class="b-form"><div class="b-field"><label for="bt-tx" class="b-star">Transaction</label><select id="bt-tx" bind:value={tx} onchange={()=>{if(tx)go(acct(n,'transactions/misc'));}}><option value="">(Select)</option>{#each codes.billingTransactions as t}<option>{t}</option>{/each}</select></div></div>
  <div class="b-actions"><a class="btn" href={href(acct(n))}>Cancel</a></div>
 </Card>
</Page>