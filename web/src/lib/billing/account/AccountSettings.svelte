<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type AccountView} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let {view}:{view:AccountView}=$props();
 // svelte-ignore state_referenced_locally
 let plan=$state(view.account.account_plan), day=$state(view.account.invoice_due_day?String(view.account.invoice_due_day):''), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 async function save(e:Event){e.preventDefault();busy=true;error='';errors={};try{const m=await billing.saveAccountSettings(n,{version:view.account.version,account_plan:plan,invoice_due_day:Number(day)});flash(m.message);go(acct(n));}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}}
</script>
<Page title="Select Billing Account Options" subtitle="Choose the account plan and the day invoices fall due." date={false} required cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Account options" icon="rules">
  <form class="b-stack" onsubmit={save}>
   <div class="b-form">
    <div class="b-field"><label for="as-plan">Account Plan</label><select id="as-plan" bind:value={plan}>{#each codes.accountPlans as p}<option>{p}</option>{/each}</select></div>
    <div class="b-field" class:invalid={!!errors.invoice_due_day}><label for="as-day" class="b-star">Invoice Due Day</label><input id="as-day" type="number" min="1" max="31" step="1" bind:value={day}/>{#if errors.invoice_due_day}<span class="err">{errors.invoice_due_day}</span>{/if}</div>
   </div>
   <div class="b-actions"><a class="btn" href={href(acct(n))}>Cancel</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!day}>Save Changes</button></div>
  </form>
 </Card>
</Page>