<script lang="ts">
 import {onMount} from 'svelte';
 import {billing,errorText,type PartyView} from '../api';
 import {href,type Match} from '../routes';
 import Banner from '../ui/Banner.svelte';
 import Placeholder from '../ui/Placeholder.svelte';
 import Status from '../ui/Status.svelte';
 import {initials} from '../format';
 import PartyDetails from './PartyDetails.svelte';
 import PartyName from './PartyName.svelte';
 import PartyRows from './PartyRows.svelte';
 import Demographics from './Demographics.svelte';
 import {kinds} from './kinds';
 let {match}:{match:Match}=$props();
 // svelte-ignore state_referenced_locally
 const id=match.params.party, comp=match.route.component;
 let party=$state<PartyView|null>(null), error=$state('');
 async function reload(){try{party=await billing.party(id);}catch(e){error=errorText(e);}}
 onMount(reload);
 const label=$derived(comp==='PartyName'?(match.params.mode==='replace'?'Replace':'Correct')+' Name & Contact Information':comp==='PartyRows'?kinds[match.params.kind]?.title??'':comp==='Demographics'?'Demographics':comp==='Placeholder'?match.route.title:'');
</script>
<div class="bw-body">
 <aside class="b-side"><div class="b-menu"><header><span class="b-avatar" aria-hidden="true">{initials(party?.name??'')}</span><div><strong>{party?.name??'…'}</strong>{#if label&&party?.accounts[0]}<small>{party.accounts[0]}</small><span><Status value="Active"/></span>{:else}<small>People &amp; Places</small>{/if}</div></header>{#if label}<nav><a class="on" href={href(match.path)}>{label}</a></nav>{/if}</div></aside>
 <div class="b-stack">
  {#if error}<Banner kind="error">{error}</Banner>{/if}
  {#if party}
   {#if comp==='PartyDetails'}<PartyDetails {party}/>
   {:else if comp==='PartyName'}<PartyName {party} mode={match.params.mode==='replace'?'replace':'correct'}/>
   {:else if comp==='Demographics'}<Demographics {party}/>
   {:else if comp==='PartyRows'&&kinds[match.params.kind]}<PartyRows {party} kind={match.params.kind} {reload}/>
   {:else}<Placeholder title={match.route.title} back={href('party/'+id)}/>{/if}
  {/if}
 </div>
</div>
