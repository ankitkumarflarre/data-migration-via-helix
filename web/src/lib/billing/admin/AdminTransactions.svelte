<script lang="ts">
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Card from '../ui/Card.svelte';
 import {billing} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import AdminMenu from './AdminMenu.svelte';
 import Page from '../ui/Page.svelte';
 let by=$state('Account Number'), q=$state(''), miss=$state('');
 async function search(e:Event){e.preventDefault();miss='';try{const h=await billing.findAccounts(q,by);if(h.length)go(acct(h[0].number,'transactions'));else miss=`No billing account matches ${q.trim()}.`;}catch(err){miss=err instanceof Error?err.message:String(err);}}
</script>
<div class="bw-body"><AdminMenu current="admin/transactions"/>
<Page title="Billing Transactions" subtitle="Find the account to add a billing transaction to." cards>
 <Card title="Find the account" icon="search">
  <form class="b-stack" onsubmit={search}>
   <div class="b-form">
    <div class="b-field"><label for="at-by">Search for Account by</label><select id="at-by" bind:value={by}>{#each codes.accountSearchBy as o}<option>{o}</option>{/each}</select></div>
    <div class="b-field"><label for="at-q">Enter Number</label><input id="at-q" bind:value={q}/></div>
   </div>
   {#if miss}<p class="b-err" role="alert">{miss}</p>{/if}
   <div class="b-actions"><a class="btn" href={href('admin')}>Cancel</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={!q.trim()}><Icon name="search" size={14}/>Search</button></div>
  </form>
 </Card>
</Page></div>