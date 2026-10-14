<script lang="ts">
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type AccountView,type TermView} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {usd,us,parseMoney} from '../format';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 // Policy Term Actions: transfer (#056), collections (#057), term plan (#058),
 // renewal plan (#059) and Paid In Full Discount (#061).
 let {view,term:t,kind}:{view:AccountView;term:TermView;kind:string}=$props();
 // svelte-ignore state_referenced_locally
 const n=view.account.number, id=t.term.id, back=href(acct(n,'term/'+id));
 // svelte-ignore state_referenced_locally
 let target=$state(''), by=$state('Account Number'), plan=$state(kind==='plan'?t.term.payment_plan:t.term.renewal_plan), amount=$state('0.00'), reason=$state(codes.pifReasons[0]);
 let error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 async function run(fn:()=>Promise<{message:string}>,to=acct(n,'term/'+id)){busy=true;error='';errors={};try{flash((await fn()).message);go(to);}catch(e){error=errorText(e);errors=fieldErrors(e);}finally{busy=false;}}
 const plans=$derived([...new Set([...t.offered_plans])]);
</script>
{#if kind==='transfer'}
 <Page title={'Transfer Policy From Account '+n} subtitle={`Moves policy term ${t.term.policy_number} ${us(t.term.effective)} - ${us(t.term.expiration)} to the account found.`} note cards>
  {#if error}<Banner kind="error">{error}</Banner>{/if}
  <Card title="Target account" icon="search">
   <form class="b-stack" onsubmit={e=>{e.preventDefault();run(()=>billing.transfer(n,id,t.version,target),acct(target.trim()));}}>
    <div class="b-form">
     <div class="b-field"><label for="tt-by">Search for Account by</label><select id="tt-by" bind:value={by}>{#each codes.accountSearchBy as o}<option>{o}</option>{/each}</select></div>
     <div class="b-field" class:invalid={!!errors.target}><label for="tt-num">Enter Number</label><input id="tt-num" bind:value={target}/>{#if errors.target}<span class="err">{errors.target}</span>{/if}</div>
    </div>
    <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!target.trim()}><Icon name="search" size={14}/>Search</button></div>
   </form>
  </Card>
 </Page>
{:else if kind==='collections'}
 <Page title={'Transfer Policy From Account '+n} note cards>
  {#if error}<Banner kind="error">{error}</Banner>{/if}
  <Card title="Transfer To Collections" icon="transfer">
   {#if t.collections_block}<Banner kind="warn">{t.collections_block}</Banner>
   {:else}<p>Transfer policy term <strong>{t.term.policy_number}</strong> with an outstanding balance of <strong>{usd(t.ledger.outstanding)}</strong> to collections?</p>{/if}
   <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span>{#if !t.collections_block}<button class="btn primary" disabled={busy} onclick={()=>run(()=>billing.collections(n,id,t.version))}>Transfer To Collections</button>{/if}</div>
  </Card>
 </Page>
{:else if kind==='plan'||kind==='renewal-plan'}
 <Page title={kind==='plan'?'Term Plan Setup':'Renewal Plan Setup'} date={false} cards>
  {#if error}<Banner kind="error">{error}</Banner>{/if}
  <Card title={kind==='plan'?'Payment Plan':'Renewal Payment Plan'} subtitle={plans.includes(plan)?'':`Current plan: ${plan} (not offered; plans are set in Administration › Workspace Settings).`} icon="calendar">
   <fieldset class="b-radios"><legend class="sr">{kind==='plan'?'Payment Plan':'Renewal Payment Plan'}</legend>{#each plans as p}<label><input type="radio" bind:group={plan} value={p}/>{p}</label>{/each}</fieldset>
   <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span><button class="btn primary" disabled={busy||!plans.includes(plan)} onclick={()=>run(()=>kind==='plan'?billing.termPlan(n,id,t.version,plan):billing.renewalPlan(n,id,t.version,plan))}>Save Changes</button></div>
  </Card>
 </Page>
{:else if kind==='pif'}
 <Page title="Paid In Full Discount" date={false} required cards>
  {#if error}<Banner kind="error">{error}</Banner>{/if}
  <Card title="Discount" icon="dollar">
   <div class="b-form">
    <div class="b-field"><span class="b-label">Term</span><span class="b-ro">{t.term.policy_number} {t.term.effective} - {t.term.expiration}</span></div>
    <div class="b-field"><span class="b-label">Term Currency</span><span class="b-ro">US Dollars ($)</span></div>
    <div class="b-field"><span class="b-label">Potential Discount</span><span class="b-ro">{t.pif.configured?usd(t.pif.potential_discount):t.pif.message}</span></div>
    <div class="b-field"><span class="b-label">Evaluation Period</span><span class="b-ro">{t.pif.configured?`${us(t.pif.evaluation_start)} - ${us(t.pif.evaluation_end)}`:t.pif.message}</span></div>
    <div class="b-field" class:invalid={!!errors.amount}><label for="pif-amt" class="b-star">Discount Amount</label><input id="pif-amt" inputmode="decimal" placeholder="$0.00" bind:value={amount}/>{#if errors.amount}<span class="err">{errors.amount}</span>{/if}</div>
    <div class="b-field"><label for="pif-reason" class="b-star">Reason</label><select id="pif-reason" bind:value={reason}>{#each codes.pifReasons as r}<option>{r}</option>{/each}</select></div>
   </div>
   <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span>{#if t.pif.configured}<button class="btn primary" disabled={busy||!(parseMoney(amount)??0)} onclick={()=>run(()=>billing.pif(n,id,{version:t.version,amount:parseMoney(amount),reason}))}>Apply Discount</button>{/if}</div>
  </Card>
 </Page>
{/if}
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>