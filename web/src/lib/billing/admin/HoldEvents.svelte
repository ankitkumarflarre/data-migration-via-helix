<script lang="ts">
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Card from '../ui/Card.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText,type HoldEvent} from '../api';
 import {href,acct} from '../routes';
 import {us} from '../format';
 import {ws} from '../state.svelte';
 import AdminMenu from './AdminMenu.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let rows=$state<HoldEvent[]>([]), error=$state(''), notice=$state(''), review=$state(''), ending=$state('');
 onMount(load);
 async function load(){try{rows=await billing.holdEvents();}catch(e){error=errorText(e);}}
 const current=$derived(rows.filter(r=>!r.end||r.end>(ws.system?.date??'')));
 async function end(r:HoldEvent){error='';try{notice=(await billing.endHoldEvent(r.id,r.version)).message;ending='';await load();}catch(e){error=errorText(e);}}
</script>
<div class="bw-body"><AdminMenu current="admin/holds"/>
<Page title="Manage Hold Events" subtitle="Area hold events pause billing for every account attached to them." cards>
 {#snippet actions()}<a class="btn sm primary" href={href('admin/holds/new')}><Icon name="plus" size={14}/>Create Hold Event</a>{/snippet}
 {#if error}<Banner kind="error">{error}</Banner>{/if}{#if notice}<Banner>{notice}</Banner>{/if}
 <Card title="Current Hold Events" icon="shield">
  <div class="b-grid"><table><thead><tr><th>Event Name</th><th>Description</th><th>Type</th><th>Reason</th><th>Start Date</th><th>End Date</th><th></th><th></th></tr></thead><tbody>
   {#each current as r (r.id)}
    <tr><td>{r.name}</td><td>{r.description}</td><td>{r.type}</td><td>{r.reason}</td><td>{us(r.start)}</td><td>{us(r.end)}</td>
     <td>{#if ending===r.id}<button class="btn danger sm" onclick={()=>end(r)}>End on {us(ws.system?.date)}</button> <button class="b-link" onclick={()=>ending=''}>Keep</button>{:else}<button class="b-link" onclick={()=>ending=r.id}>End</button>{/if}</td>
     <td><button class="b-link" aria-expanded={review===r.id} onclick={()=>review=review===r.id?'':r.id}>Review</button></td></tr>
    {#if review===r.id}<tr class="desc"><td colspan="8">{#if r.accounts?.length}Accounts on this hold: {#each r.accounts as a,i}{i?', ':''}<a href={href(acct(a))}>{a}</a>{/each}{:else}No accounts are attached to this event yet.{/if}</td></tr>{/if}
   {:else}<tr><td colspan="8" class="empty">No current hold events.</td></tr>{/each}
  </tbody></table></div>
 </Card>
</Page></div>