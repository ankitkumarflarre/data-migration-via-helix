<script lang="ts">
 import Status from '../ui/Status.svelte';
 import Card from '../ui/Card.svelte';
 import {billing,errorText} from '../api';
 import {stamp} from '../format';
 import {ws,loadSystem} from '../state.svelte';
 import AdminMenu from './AdminMenu.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let error=$state(''), notice=$state(''), busy=$state(false);
 const svc=$derived(ws.system?.settings.disbursement_service);
 async function act(action:'start'|'stop'){if(!ws.system)return;busy=true;error='';notice='';try{ws.system=await billing.service(ws.system.version,action);notice=action==='start'?'A disbursement interface request was added to the async queue. It runs with the next due processes.':'The disbursement interface service was stopped and the queued request removed.';}catch(e){error=errorText(e);await loadSystem();}finally{busy=false;}}
</script>
<div class="bw-body"><AdminMenu current="admin/disbursement"/>
<Page title="Disbursement Interface Service" subtitle="Restart the disbursement interface if it has stopped for any reason." cards>
 {#if error}<Banner kind="warn">{error}</Banner>{/if}{#if notice}<Banner>{notice}</Banner>{/if}
 <Card title="Interface service" icon="transfer">
  {#snippet tools()}{#if svc}<Status value={svc.status}/>{/if}{/snippet}
  <p class="b-note">Clicking 'Start Service' adds a new Async Queue request to process disbursements. If there is already a request on the queue, this will result in duplicate requests and the interface processing will run more frequently than expected. Make sure you know the status of the interface service before you use this function.</p>
  {#if svc}<div class="b-form"><div class="b-field"><span class="b-label">Status</span><span class="b-ro">{svc.status}</span></div>{#if svc.queued}<div class="b-field"><span class="b-label">Request queued</span><span class="b-ro">{stamp(svc.queued_at??'')}</span></div>{/if}{#if svc.last_run}<div class="b-field"><span class="b-label">Last run</span><span class="b-ro">{stamp(svc.last_run)}</span></div>{/if}</div>{/if}
  <div class="b-actions"><span class="spacer"></span><button class="btn" disabled={busy} onclick={()=>act('stop')}>Stop Service</button><button class="btn primary" disabled={busy} onclick={()=>act('start')}>Start Service</button></div>
 </Card>
</Page></div>