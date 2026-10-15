<script lang="ts">
 import {onMount} from 'svelte';
 import './billing.css';
 import Icon from '../personalhome/WorkspaceIcon.svelte';
 import {billing,errorText,BillingError} from './api';
 import {parse,go,href,acct,type Match} from './routes';
 import {ws,loadSystem,flash,changed} from './state.svelte';
 import {us} from './format';
 import Placeholder from './ui/Placeholder.svelte';
 import Home from './screens/Home.svelte';
 import PaymentSearch from './screens/PaymentSearch.svelte';
 import PaymentDetail from './screens/PaymentDetail.svelte';
 import EnterPayment from './screens/EnterPayment.svelte';
 import UnidentifiedList from './screens/UnidentifiedList.svelte';
 import UnidentifiedDetail from './screens/UnidentifiedDetail.svelte';
 import NewAccount from './screens/NewAccount.svelte';
 import PartySearch from './screens/PartySearch.svelte';
 import ImportStatements from './screens/ImportStatements.svelte';
 import Reports from './screens/Reports.svelte';
 import Admin from './admin/Admin.svelte';
 import ErrorQueue from './admin/ErrorQueue.svelte';
 import HoldEvents from './admin/HoldEvents.svelte';
 import DisbursementService from './admin/DisbursementService.svelte';
 import AdminTransactions from './admin/AdminTransactions.svelte';
 import AdminSettings from './admin/AdminSettings.svelte';
 import Batches from './admin/Batches.svelte';
 import AccountLayout from './account/AccountLayout.svelte';
 import PartyLayout from './party/PartyLayout.svelte';

 let match=$state<Match>(parse(location.hash));
 let error=$state(''), date=$state(''), busy=$state(false);
 let quickBy=$state('Account Number'), quickQ=$state('');
 const top=$derived(match.path.startsWith('admin')?'admin':match.path.startsWith('batches')?'batches':match.path.startsWith('party/')?'party':'home');

 onMount(()=>{
  const sync=()=>{if(location.hash.startsWith('#billing')){match=parse(location.hash);window.scrollTo({top:0});}};
  window.addEventListener('hashchange',sync);
  loadSystem().then(s=>date=s.date).catch(e=>error=errorText(e));
  return()=>window.removeEventListener('hashchange',sync);
 });

 async function quick(e:Event){
  e.preventDefault();
  if(!quickQ.trim())return;
  try{const hits=await billing.findAccounts(quickQ,quickBy);if(hits.length){quickQ='';go(acct(hits[0].number));}else{flash(`No billing account matches ${quickQ.trim()}.`,'warn');go('');match=parse(location.hash);}}catch(err){error=errorText(err);}
 }
 async function setDate(){
  if(!ws.system||!date)return;
  busy=true;error='';
  try{ws.system=await billing.setDate(ws.system.version,date);changed();}catch(err){error=errorText(err);if(err instanceof BillingError&&err.stale)await loadSystem();}finally{busy=false;}
 }
 async function runDue(){
  if(!ws.system)return;
  busy=true;error='';
  try{const r=await billing.runDue(ws.system.version);await loadSystem();flash(r.message);changed();match=parse(location.hash);}catch(err){error=errorText(err);}finally{busy=false;}
 }
 async function seed(){
  busy=true;error='';
  try{ws.system=await billing.seed();date=ws.system.date;flash('Loaded the walkthrough test data. System date set to '+us(ws.system.date)+'.');changed();go('');}catch(err){error=errorText(err);}finally{busy=false;}
 }
</script>

<div class="bw">
 <div class="bw-top">
  <nav class="bw-tabs" aria-label="Billing">
   <a href={href('')} class:on={top==='home'||top==='party'} aria-current={top==='home'?'page':undefined}><Icon name="home" size={14}/>Home</a>
   <a href={href('admin')} class:on={top==='admin'} aria-current={top==='admin'?'page':undefined}><Icon name="rules" size={14}/>Administration</a>
   <a href={href('batches')} class:on={top==='batches'} aria-current={top==='batches'?'page':undefined}><Icon name="card" size={14}/>Batch Cash Management</a>
  </nav>
  <!-- Home has its own account search in the Billing accounts card. -->
  {#if match.route.component!=='Home'}<form class="bw-quick" role="search" onsubmit={quick}>
   <label class="sr-only" for="bw-quick-by">Search by</label>
   <select id="bw-quick-by" bind:value={quickBy}><option>Account Number</option></select>
   <div class="b-search"><Icon name="search" size={14}/><label class="sr-only" for="bw-quick-q">Account number</label><input id="bw-quick-q" type="search" bind:value={quickQ} placeholder="Search account number…" autocomplete="off"/><button type="submit" aria-label="Search accounts"><Icon name="arrow" size={14}/></button></div>
  </form>{/if}
 </div>

 {#if error}<div class="b-banner error" role="alert"><span class="ico">!</span><div>{error}</div></div>{/if}
 {#if ws.system?.empty && match.route.component==='Home'}
  <div class="b-banner"><span class="ico">i</span><div>The billing store is empty. <button class="btn sm primary" disabled={busy} onclick={seed}>Load walkthrough test data</button> loads the accounts, payments, suspense items, batches, errors and parties shown in the Duck Creek screenshots (with test names and addresses). Or run <code>migrator billing-seed</code>.</div></div>
 {/if}

 {#key match.path+'?'+match.query.toString()+'#'+ws.tick}
  {#if match.path.startsWith('account/')}
   <AccountLayout {match}/>
  {:else if match.path.startsWith('party/')}
   <PartyLayout {match}/>
  {:else}
   <div class="bw-body wide">
    {#if match.route.component==='Home'}<Home/>
    {:else if match.route.component==='PaymentSearch'}<PaymentSearch/>
    {:else if match.route.component==='PaymentDetail'}<PaymentDetail id={match.params.payment}/>
    {:else if match.route.component==='EnterPayment'}<EnterPayment/>
    {:else if match.route.component==='UnidentifiedList'}<UnidentifiedList/>
    {:else if match.route.component==='UnidentifiedDetail'}<UnidentifiedDetail id={match.params.payment}/>
    {:else if match.route.component==='NewAccount'}<NewAccount query={match.query}/>
    {:else if match.route.component==='PartySearch'}<PartySearch query={match.query}/>
    {:else if match.route.component==='ImportStatements'}<ImportStatements/>
    {:else if match.route.component==='Reports'}<Reports id={match.params.report??''}/>
    {:else if match.route.component==='Admin'}<Admin/>
    {:else if match.route.component==='ErrorQueue'}<ErrorQueue/>
    {:else if match.route.component==='HoldEvents'}<HoldEvents/>
    {:else if match.route.component==='DisbursementService'}<DisbursementService/>
    {:else if match.route.component==='AdminTransactions'}<AdminTransactions/>
    {:else if match.route.component==='AdminSettings'}<AdminSettings/>
    {:else if match.route.component==='Batches'}<Batches path={match.path}/>
    {:else if match.route.component==='Placeholder'}<Placeholder title={match.route.title} back={href('admin/holds')}/>
    {:else}<Placeholder title="Page not found" back={href('')}/>{/if}
   </div>
  {/if}
 {/key}

 <div class="bw-foot">
  <span class="b-ico soft"><Icon name="calendar" size={14}/></span><label for="bw-date">System date</label>
  <input id="bw-date" type="date" bind:value={date} disabled={busy||!ws.system}/>
  <button class="btn sm" disabled={busy||!ws.system||date===ws.system?.date} onclick={setDate}>Set System Date</button>
  <button class="btn sm" disabled={busy||!ws.system} onclick={runDue} title="Generate invoices that are due, flag past-due installments and let a queued disbursement interface request issue authorized refunds">Run due processes</button>
  <span class="spacer"></span>
  <span>Local workspace · {ws.system?.user??''} · {match.route.title}</span>
 </div>
</div>

<style>
 .sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0);white-space:nowrap}
</style>
