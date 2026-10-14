<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText,fieldErrors,type AccountView} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {usd,parseMoney,moneyText,initials} from '../format';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let {view,query}:{view:AccountView;query:URLSearchParams}=$props();
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 // svelte-ignore state_referenced_locally
 const ids=(query.get('ids')??'').split(',').filter(Boolean);
 // svelte-ignore state_referenced_locally
 const items=view.suspense.filter(s=>ids.includes(s.id));
 const available=items.reduce((t,s)=>t+s.amount,0);
 // svelte-ignore state_referenced_locally
 let payeeID=$state(query.get('payee')??''), payeeName=$state(view.payee?.name??'');
 let amountText=$state(moneyText(available)), type=$state(codes.disbursementTypes[0]), reason=$state(''), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 const amount=$derived(parseMoney(amountText));
 const remaining=$derived(available-(amount??0));
 onMount(async()=>{if(payeeID){try{payeeName=(await billing.party(payeeID)).name;}catch{payeeID='';}}});
 // Refund stays disabled until a reason is chosen and the amount fits.
 const ready=$derived(!!reason&&amount!==null&&amount>0&&amount<=available);
 async function refund(){busy=true;error='';errors={};try{const r=await billing.disburse(n,{version:view.account.version,suspense_ids:ids,amount,type,reason,payee_id:payeeID});flash(r.message);go(acct(n,'suspense'));}catch(e){error=errorText(e);errors=fieldErrors(e);}finally{busy=false;}}
</script>
<Page title="Disburse Funds" subtitle="Refund the selected suspense items to a payee." note cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 {#if !items.length}<Banner kind="warn">Select items on Items In Suspense first. <a href={href(acct(n,'suspense'))}>Items In Suspense</a></Banner>
 {:else}
  <Card title="Payee" icon="user">
   <div class="b-inline"><span class="b-avatar" aria-hidden="true">{initials(payeeName)}</span><strong>{payeeName}</strong><span class="spacer"></span><a class="btn sm" href={href(`parties/search?for=disburse-payee&account=${encodeURIComponent(n)}&ids=${ids.join(',')}`)}>Select Another Payee</a></div>
  </Card>
  <Card title={`${usd(available)} Available For Disbursement`} icon="dollar">
   <div class="b-grid"><table><thead><tr><th>Direction</th><th class="num">Available Amount</th><th>Disbursement Group</th><th>Disbursement Amount</th><th class="num">Amount Remaining</th></tr></thead><tbody>
    <tr><td>{n}</td><td class="num">{usd(available)}</td><td>{codes.disbursementGroups[0]}</td><td><input aria-label="Disbursement amount" inputmode="decimal" bind:value={amountText} class:invalid={!!errors.amount||amount===null||remaining<0}/>{#if errors.amount}<div class="b-err">{errors.amount}</div>{/if}</td><td class="num">{usd(Math.max(remaining,0))}</td></tr>
   </tbody><tfoot><tr><td colspan="5"><div class="b-pager"><span>{usd(amount??0)} disbursed / {usd(Math.max(remaining,0))} remaining</span></div></td></tr></tfoot></table></div>
  </Card>
  <Card title="Disbursement" subtitle="Refunds are issued by check, whatever the original payment method." icon="transfer">
   <div class="b-form">
    <div class="b-field"><label for="db-type">Disbursement Type</label><select id="db-type" bind:value={type}>{#each codes.disbursementTypes as t}<option>{t}</option>{/each}</select></div>
    <div class="b-field" class:invalid={!!errors.reason}><label for="db-reason">Disbursement Reason</label><select id="db-reason" bind:value={reason}><option value="">Select Disbursement Reason</option>{#each codes.disbursementReasons as r}<option>{r}</option>{/each}</select>{#if errors.reason}<span class="err">{errors.reason}</span>{/if}</div>
   </div>
   <div class="b-actions"><a class="btn" href={href(acct(n,'suspense'))}>Cancel</a><span class="spacer"></span><button class="btn primary" disabled={busy||!ready} onclick={refund}>Refund</button></div>
  </Card>
 {/if}
 {#if !items.length}<div class="b-bar"><a class="btn" href={href(acct(n,'suspense'))}>Cancel</a></div>{/if}
</Page>