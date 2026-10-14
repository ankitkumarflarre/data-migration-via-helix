<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors} from '../api';
 import {codes} from '../codes';
 import {usd,parseMoney,moneyText} from '../format';
 import {ws,loadSystem,changed} from '../state.svelte';
 import AdminMenu from './AdminMenu.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 const s=ws.system?.settings;
 let limit=$state(moneyText(s?.auto_authorize_limit??100000)), pifOn=$state(!!s?.pif), pct=$state(String((s?.pif?.discount_basis_points??200)/100)), days=$state(String(s?.pif?.evaluation_days??30)), plans=$state<string[]>([...(s?.offered_plans??['Premium Finance Annual'])]);
 let error=$state(''), errors=$state<Record<string,string>>({}), notice=$state(''), busy=$state(false);
 async function save(e:Event){
  e.preventDefault();if(!ws.system)return;error='';errors={};notice='';
  const cents=parseMoney(limit), bp=Math.round(Number(pct)*100);
  if(cents===null){errors={auto_authorize_limit:'Enter an amount.'};return;}
  busy=true;
  try{ws.system=await billing.saveSettings({version:ws.system.version,auto_authorize_limit:cents,pif:pifOn?{discount_basis_points:bp,evaluation_days:Number(days)}:null,offered_plans:plans});notice='Settings saved.';changed();}catch(err){error=errorText(err);errors=fieldErrors(err);await loadSystem();}finally{busy=false;}
 }
</script>
<div class="bw-body"><AdminMenu current="admin/settings"/>
<Page title="Workspace Settings" subtitle="Configuration the walkthrough depends on but does not show. It applies to this local workspace only." cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}{#if notice}<Banner>{notice}</Banner>{/if}
 <form class="b-stack" onsubmit={save}>
  <Card title="Disbursements" icon="transfer">
   <div class="b-form">
    <div class="b-field wide" class:invalid={!!errors.auto_authorize_limit}><label for="as-limit">Auto-authorize disbursements up to</label><input id="as-limit" inputmode="decimal" bind:value={limit}/><small class="b-note">At or under this amount, System authorizes the refund at once (currently {usd(ws.system?.settings.auto_authorize_limit??0)}). Above it, an approver decides on Items In Suspense.</small>{#if errors.auto_authorize_limit}<span class="err">{errors.auto_authorize_limit}</span>{/if}</div>
   </div>
  </Card>
  <Card title="Paid In Full configuration" icon="dollar">
   {#snippet tools()}<label class="b-check"><input id="as-pif" type="checkbox" bind:checked={pifOn}/>Established</label>{/snippet}
   {#if pifOn}
    <div class="b-form">
     <div class="b-field" class:invalid={!!errors['pif.discount_basis_points']}><label for="as-pct">Discount (% of term premium)</label><input id="as-pct" type="number" min="0.01" max="100" step="0.01" bind:value={pct}/>{#if errors['pif.discount_basis_points']}<span class="err">{errors['pif.discount_basis_points']}</span>{/if}</div>
     <div class="b-field" class:invalid={!!errors['pif.evaluation_days']}><label for="as-days">Evaluation period (days from effective date)</label><input id="as-days" type="number" min="1" max="366" step="1" bind:value={days}/>{#if errors['pif.evaluation_days']}<span class="err">{errors['pif.evaluation_days']}</span>{/if}</div>
    </div>
   {:else}<p class="b-note">No Paid In Full discount is configured.</p>{/if}
  </Card>
  <Card title="Payment plans" subtitle="Plans offered on Term Plan Setup and Renewal Plan Setup." icon="calendar">
   <fieldset class="b-radios"><legend class="sr">Payment plans offered</legend>{#each codes.paymentPlans as p}<label><input type="checkbox" bind:group={plans} value={p}/>{p}</label>{/each}</fieldset>
   {#if errors.offered_plans}<span class="b-err">{errors.offered_plans}</span>{/if}
   <div class="b-actions"><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!plans.length}>Save Changes</button></div>
  </Card>
 </form>
</Page></div>
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>