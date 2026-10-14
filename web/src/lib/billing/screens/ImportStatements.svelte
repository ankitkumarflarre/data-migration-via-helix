<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText} from '../api';
 import {href} from '../routes';
 import {us} from '../format';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let rows=$state<{id:string;name:string;imported?:string}[]>([]), picked=$state<string[]>([]), error=$state(''), notice=$state(''), busy=$state(false);
 onMount(load);
 async function load(){try{rows=await billing.statements();}catch(e){error=errorText(e);}}
 const open=$derived(rows.filter(r=>!r.imported));
 async function run(){busy=true;error='';try{notice=(await billing.importStatements(picked)).message;picked=[];await load();}catch(e){error=errorText(e);}finally{busy=false;}}
</script>
<Page title="Import Agent Statement" subtitle="From the list below, select the statements you wish to import." cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}{#if notice}<Banner>{notice}</Banner>{/if}
 <Card title="Agent statements" icon="file">
  <div class="b-grid"><table><thead><tr><th><input type="checkbox" aria-label="Select all statements" disabled={!open.length} checked={open.length>0&&picked.length===open.length} onchange={e=>picked=e.currentTarget.checked?open.map(r=>r.id):[]}/></th><th>Statement file</th></tr></thead><tbody>
   {#each rows as r (r.id)}<tr><td><input type="checkbox" aria-label={'Select '+r.name} disabled={!!r.imported} bind:group={picked} value={r.id}/></td><td>{r.name}{#if r.imported} <span class="b-badge ok">imported {us(r.imported)}</span>{/if}</td></tr>{:else}<tr><td colspan="2" class="empty">No agent statements are waiting to be imported. The agency-bill flow is not part of the walkthrough.</td></tr>{/each}
  </tbody></table></div>
  <div class="b-actions"><a class="btn" href={href('')}>Cancel</a><span class="spacer"></span><button class="btn primary" disabled={busy||!picked.length} onclick={run}>Import Statements</button></div>
 </Card>
</Page>