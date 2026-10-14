<script lang="ts">
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Card from '../ui/Card.svelte';
 import Status from '../ui/Status.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText,fieldErrors,type Batch} from '../api';
 import {href,go} from '../routes';
 import {codes} from '../codes';
 import {usd,us,parseMoney} from '../format';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import Pager from '../ui/Pager.svelte';
 let {path}:{path:string}=$props();
 // svelte-ignore state_referenced_locally
 const mode=path==='batches/new'?'new':path==='batches/processed'?'processed':'awaiting';
 let rows=$state<Batch[]>([]), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false), page=$state(0);
 let nb=$state({number:'',control:'',currency:'',method:'Check',org:'',description:''});
 let q=$state({number:'',user:'',deposited:'',created:''}), searchedQ=$state<typeof q|null>(null), picked=$state('');
 onMount(load);
 async function load(){try{rows=await billing.batches();}catch(e){error=errorText(e);}}
 const awaiting=$derived(rows.filter(b=>!['Deposited','Reversed'].includes(b.status)));
 const processed=$derived(searchedQ?rows.filter(b=>['Deposited','Reversed'].includes(b.status)&&(!searchedQ!.number||b.number.includes(searchedQ!.number.trim()))&&(!searchedQ!.user||b.user.toLowerCase().includes(searchedQ!.user.trim().toLowerCase()))&&(!searchedQ!.deposited||b.deposited_on===searchedQ!.deposited)&&(!searchedQ!.created||b.entry_date===searchedQ!.created)):[]);
 const ready=$derived(!!nb.number.trim()&&parseMoney(nb.control)!==null&&!!nb.currency&&!!nb.method&&!!nb.org);
 const anyQ=$derived(Object.values(q).some(v=>v.trim()!==''));
 async function create(e:Event){e.preventDefault();busy=true;error='';errors={};try{const b=await billing.createBatch({number:nb.number,control_total:parseMoney(nb.control),currency:nb.currency,default_method:nb.method,org_unit:nb.org,description:nb.description});flash(`Batch ${b.number} was created.`);go('batches');}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}}
 async function reverse(){const b=rows.find(r=>r.number===picked);if(!b)return;busy=true;error='';try{const m=await billing.reverseBatch(b.number,b.version);flash(m.message);picked='';await load();error='';}catch(err){error=errorText(err);}finally{busy=false;}}
</script>
<div class="bw-body">
 <aside class="b-side"><div class="b-menu"><nav aria-label="Batch Cash Management"><a href={href('batches')} class:on={mode==='awaiting'}>Awaiting Deposit</a><a href={href('batches/new')} class:on={mode==='new'}>Start A New Batch</a><a href={href('batches/processed')} class:on={mode==='processed'}>Processed Batches</a></nav></div></aside>
 {#if mode==='awaiting'}
  <Page title="Batch Cash Management" subtitle="Payment batches entered and waiting to be deposited." cards>
   {#snippet actions()}<a class="btn sm primary" href={href('batches/new')}><Icon name="plus" size={14}/>Start A New Batch</a>{/snippet}
   {#if error}<Banner kind="error">{error}</Banner>{/if}
   <Card title="Batches Awaiting Deposit" subtitle="Batch entry, deposit and the batch detail screen are not shown in the walkthrough." icon="database">
    <div class="b-grid"><table><thead><tr><th>Batch Number</th><th>Entry Date</th><th>User Id</th><th>Status</th><th class="num">Control Total</th><th class="num">Batch Total</th><th>Currency</th></tr></thead><tbody>
     {#each awaiting.slice(page*10,page*10+10) as b (b.number)}<tr class="has-desc"><td>{b.number}</td><td>{us(b.entry_date)}</td><td>{b.user}</td><td><Status value={b.status}/></td><td class="num">{usd(b.control_total)}</td><td class="num">{usd(b.batch_total)}</td><td>{b.currency}</td></tr><tr class="desc"><td colspan="7">{b.description}</td></tr>{:else}<tr><td colspan="7" class="empty">No batches are awaiting deposit.</td></tr>{/each}
    </tbody><Pager total={awaiting.length} bind:page cols={7}/></table></div>
   </Card>
  </Page>
 {:else if mode==='new'}
  <Page title="Start A New Payment Batch" subtitle="Payments entered in the batch must add up to the control total." cards>
   {#if error}<Banner kind="error">{error}</Banner>{/if}
   <Card title="Batch details" icon="card">
    <form class="b-stack" onsubmit={create}>
     <div class="b-form">
      <div class="b-field" class:invalid={!!errors.number}><label for="nb-num">Batch Number</label><input id="nb-num" bind:value={nb.number} maxlength="20"/>{#if errors.number}<span class="err">{errors.number}</span>{/if}</div>
      <div class="b-field" class:invalid={!!errors.control_total}><label for="nb-ctl">Control Total</label><input id="nb-ctl" inputmode="decimal" placeholder="$0.00" bind:value={nb.control}/>{#if errors.control_total}<span class="err">{errors.control_total}</span>{/if}</div>
      <div class="b-field"><label for="nb-cur">Currency</label><select id="nb-cur" bind:value={nb.currency}><option value="">(select)</option>{#each codes.batchCurrencies as c}<option>{c}</option>{/each}</select></div>
      <div class="b-field"><label for="nb-method">Default Payment Method</label><select id="nb-method" bind:value={nb.method}>{#each codes.paymentMethods as m}<option>{m}</option>{/each}</select></div>
      <div class="b-field"><label for="nb-org">Processing Org. Unit</label><select id="nb-org" bind:value={nb.org}><option value="">(select)</option>{#each codes.processingOrgUnits as o}<option>{o}</option>{/each}</select></div>
      <div class="b-field"><label for="nb-desc">Description</label><input id="nb-desc" bind:value={nb.description} maxlength="200"/></div>
     </div>
     <div class="b-actions"><a class="btn" href={href('batches')}>Cancel</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!ready}>Create</button></div>
    </form>
   </Card>
  </Page>
 {:else}
  <Page title="Processed Batches" subtitle="Search for processed batches by entering information in one or more of the fields below." cards>
   {#if error}<Banner kind="error">{error}</Banner>{/if}
   <Card title="Search criteria" icon="filter">
    <div class="b-form">
     <div class="b-field"><label for="pb-num">Batch Number</label><input id="pb-num" bind:value={q.number}/></div>
     <div class="b-field"><label for="pb-user">User Id</label><input id="pb-user" bind:value={q.user}/></div>
     <div class="b-field"><label for="pb-dep">Date Deposited</label><input id="pb-dep" type="date" bind:value={q.deposited}/></div>
     <div class="b-field"><label for="pb-cre">Date Created</label><input id="pb-cre" type="date" bind:value={q.created}/></div>
    </div>
    <div class="b-actions"><span class="spacer"></span><button class="btn" onclick={()=>{searchedQ={number:'',user:'',deposited:'',created:''};picked='';}}>Display All Items</button><button class="btn primary" disabled={!anyQ} onclick={()=>{searchedQ={...q};picked='';}}><Icon name="search" size={14}/>Search</button></div>
   </Card>
   <Card title="Processed Batches" subtitle="Select a deposited batch to reverse it." icon="database">
    <div class="b-grid"><table><thead><tr><th><span class="sr">Select</span></th><th>Batch Number</th><th>Date Created</th><th>Date Deposited</th><th>User Id</th><th>Status</th><th class="num">Batch Total</th><th>Currency</th></tr></thead><tbody>
     {#each processed as b (b.number)}<tr><td><input type="radio" name="pb" aria-label={'Select batch '+b.number} bind:group={picked} value={b.number} disabled={b.status!=='Deposited'}/></td><td>{b.number}</td><td>{us(b.entry_date)}</td><td>{us(b.deposited_on)}</td><td>{b.user}</td><td><Status value={b.status}/></td><td class="num">{usd(b.batch_total)}</td><td>{b.currency}</td></tr>{:else}<tr><td colspan="8" class="empty">No results to display.</td></tr>{/each}
    </tbody></table></div>
    <div class="b-actions"><a class="btn" href={href('batches')}>Cancel</a><span class="spacer"></span><button class="btn danger" disabled={busy||!picked} onclick={reverse}>Reverse</button></div>
   </Card>
  </Page>
 {/if}
</div>
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>
