<script lang="ts">
 import Card from '../ui/Card.svelte';
 import Status from '../ui/Status.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText,fieldErrors,type ErrorItem} from '../api';
 import {codes} from '../codes';
 import {stamp,us} from '../format';
 import AdminMenu from './AdminMenu.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import Modal from '../ui/Modal.svelte';
 let rows=$state<ErrorItem[]>([]), error=$state(''), notice=$state('');
 let start=$state(''), end=$state(''), type=$state('All'), status=$state('All'), applied=$state({start:'',end:'',type:'All',status:'All'});
 let details=$state<ErrorItem|null>(null), resubmit=$state<ErrorItem|null>(null), draft=$state<{sections:{name:string;value:string}[];invoice_id:string;async_id:string}>({sections:[],invoice_id:'',async_id:''}), errors=$state<Record<string,string>>({}), busy=$state(false);
 onMount(load);
 async function load(){try{rows=await billing.errors();}catch(e){error=errorText(e);}}
 const shown=$derived(rows.filter(r=>(!applied.start||r.at.slice(0,10)>=applied.start)&&(!applied.end||r.at.slice(0,10)<=applied.end)&&(applied.type==='All'||r.type===applied.type)&&(applied.status==='All'||r.status===applied.status)));
 function openResubmit(r:ErrorItem){resubmit=r;draft={sections:r.sections.map(s=>({...s})),invoice_id:r.invoice_id,async_id:r.async_id};errors={};}
 async function submit(){if(!resubmit)return;busy=true;errors={};try{const r=await billing.resubmit(resubmit.id,{version:resubmit.version,...draft});notice=r.status==='Resolved'?`Request resubmitted and processed. ${r.description}`:`Request resubmitted but failed again: ${r.description}`;resubmit=null;await load();}catch(e){errors={...fieldErrors(e),_:errorText(e)};}finally{busy=false;}}
</script>
<div class="bw-body"><AdminMenu current="admin/errors"/>
<Page title="Error Queue" subtitle="Errors raised by billing processes. Resubmit an open error once it is fixed." cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}{#if notice}<Banner>{notice}</Banner>{/if}
 <Card title="Filter Error List" icon="filter">
  <div class="b-form">
   <div class="b-field"><label for="eq-start">Start Date</label><input id="eq-start" type="date" bind:value={start}/></div>
   <div class="b-field"><label for="eq-end">End Date</label><input id="eq-end" type="date" bind:value={end}/></div>
   <div class="b-field"><label for="eq-type">Type</label><select id="eq-type" bind:value={type}><option>All</option>{#each codes.errorTypes as t}<option>{t}</option>{/each}</select></div>
   <div class="b-field"><label for="eq-status">Status</label><select id="eq-status" bind:value={status}><option>All</option>{#each codes.errorStatuses as s}<option>{s}</option>{/each}</select></div>
  </div>
  <div class="b-actions"><span class="spacer"></span><button class="btn primary" onclick={()=>applied={start,end,type,status}}>Refresh List</button></div>
 </Card>
 <Card title="Errors" icon="database">
  {#snippet tools()}<span class="b-badge info">{shown.length} shown</span>{/snippet}
  <div class="b-grid"><table><thead><tr><th>Date</th><th>Type</th><th>Reference</th><th>Status</th><th></th><th></th></tr></thead><tbody>
   {#each shown as r (r.id)}
    <tr class="has-desc"><td>{stamp(r.at).replace(/ (am|pm)$/,m=>m.toUpperCase())}</td><td title={r.type}>{r.type.length>30?r.type.slice(0,28)+'…':r.type}</td><td>{r.reference}</td><td><Status value={r.status}/></td><td><button class="b-link" onclick={()=>details=r}>View Details</button></td><td><button class="b-link" disabled={r.status!=='Open'} onclick={()=>openResubmit(r)}>Resubmit</button></td></tr>
    <tr class="desc"><td colspan="6">{r.summary}{#if r.attempts} · resubmitted {r.attempts}×{/if}</td></tr>
   {:else}<tr><td colspan="6" class="empty">No errors match the filter.</td></tr>{/each}
  </tbody></table></div>
 </Card>
</Page></div>
{#if details}
 <Modal title="Error Details" onclose={()=>details=null}>
  <div class="b-field"><span class="b-label">Error Description:</span><p>{details.description}</p></div>
  <div class="b-field"><span class="b-label">XML:</span><pre class="b-xml">{details.payload}</pre></div>
  {#if details.resolved_at}<p class="b-note">Resolved {us(details.resolved_at)}.</p>{/if}
  <div class="b-actions"><button class="btn" onclick={()=>details=null}>Close</button></div>
 </Modal>
{/if}
{#if resubmit}
 <Modal title="Resubmission Details" onclose={()=>resubmit=null}>
  {#if errors._}<Banner kind="error">{errors._}</Banner>{/if}
  <div class="b-field"><span class="b-label">Error Description:</span><p>{resubmit.description}</p></div>
  <p class="b-label">Payload:</p>
  {#each draft.sections as s,i}<div class="b-field" class:invalid={!!errors['sections.'+i]}><label for={'rs-'+i}>{s.name}</label><textarea id={'rs-'+i} rows="4" class="mono" bind:value={draft.sections[i].value}></textarea></div>{/each}
  <div class="b-form"><div class="b-field"><label for="rs-inv">invoiceID</label><input id="rs-inv" bind:value={draft.invoice_id}/></div><div class="b-field"><label for="rs-async">asyncID</label><input id="rs-async" bind:value={draft.async_id}/></div></div>
  <div class="b-actions"><button class="btn primary" disabled={busy} onclick={submit}>Submit</button><button class="btn" onclick={()=>resubmit=null}>Close</button></div>
 </Modal>
{/if}
<style>.mono{font-family:var(--mono);font-size:12px}</style>
