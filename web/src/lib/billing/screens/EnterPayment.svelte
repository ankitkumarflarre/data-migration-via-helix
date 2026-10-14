<script lang="ts">
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type AccountRef,type Identity} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {parseMoney,usd} from '../format';
 import {flash,today} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import IdentityFields from './IdentityFields.svelte';
 let mode=$state<'known'|'unidentified'>('known');
 let by=$state('Account Number'), using=$state(''), hits=$state<AccountRef[]|null>(null), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 let ident=$state<Identity>({last:'',first:'',middle:'',account_ref:'',policy_ref:'',other:''});
 let method=$state(''), amount=$state(''), entry=$state(today()), check=$state(''), nameOnCheck=$state('');
 const identified=$derived(!!(ident.last.trim()||ident.account_ref.trim()||ident.policy_ref.trim()||ident.other.trim()));
 async function search(e:Event){e.preventDefault();error='';hits=null;try{hits=await billing.findAccounts(using,by);}catch(err){error=errorText(err);}}
 async function enter(e:Event){
  e.preventDefault();error='';errors={};
  const cents=parseMoney(amount);
  if(cents===null){errors={amount:'Enter a payment amount.'};return;}
  busy=true;
  try{const p=await billing.enterUnidentified({...ident,method,amount:cents,entry_date:entry,check_number:check,name_on_check:nameOnCheck});flash(`Unidentified payment for ${usd(p.amount)} entered.`);go('unidentified');}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}
 }
</script>
<Page title="Enter A Single Payment" subtitle="Apply a payment to a billing account, or record it as unidentified." date={false} required={mode==='unidentified'} cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 <Card title="Payment Reference" subtitle="Do you know which account this payment belongs to?" icon="dollar">
  <fieldset class="b-radios"><legend class="sr">Payment reference</legend>
   <label><input type="radio" bind:group={mode} value="known"/>I know the account, policy, arrangement or name</label>
   <label><input type="radio" bind:group={mode} value="unidentified"/>This is an unidentified payment, not associated with an account, policy or arrangement</label>
  </fieldset>
 </Card>
 {#if mode==='known'}
  <Card title="Find the account" subtitle="Search, then open Payment Processing for the account." icon="search">
   <form class="b-stack" onsubmit={search}>
    <div class="b-form">
     <div class="b-field"><label for="ep-by">Search by</label><select id="ep-by" bind:value={by}>{#each codes.accountSearchBy as o}<option>{o}</option>{/each}</select></div>
     <div class="b-field"><label for="ep-using">Using</label><input id="ep-using" bind:value={using} autocomplete="off"/></div>
    </div>
    <div class="b-actions"><a class="btn" href={href('')}>Cancel</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={!using.trim()}><Icon name="search" size={14}/>Account Search</button></div>
   </form>
  </Card>
  {#if hits}
   <Card title="Search Results" subtitle="Select Payment Processing to enter the payment on that account.">
    <div class="b-grid"><table><thead><tr><th>Account Number</th><th>Payor Name</th><th>Address</th><th></th></tr></thead><tbody>
     {#each hits as h}<tr><td>{h.number}</td><td>{h.name}</td><td title={h.address}>{h.address.length>16?h.address.slice(0,14)+', …':h.address}</td><td><a href={href(acct(h.number,'pay'))}>Payment Processing</a></td></tr>{:else}<tr><td colspan="4" class="empty">No billing account matches {using}.</td></tr>{/each}
    </tbody></table></div>
   </Card>
  {/if}
 {:else}
  <form class="b-stack" onsubmit={enter}>
   <Card title="Payer identity" subtitle="How the payment can be traced later." icon="user"><IdentityFields bind:value={ident} {errors} prefix="ep-"/></Card>
   <Card title="Payment Details" icon="dollar">
    <div class="b-form">
     <div class="b-field" class:invalid={!!errors.method}><label for="ep-method" class="b-star">Payment Method</label><select id="ep-method" bind:value={method}><option value="">(Select)</option>{#each codes.paymentMethods.filter(m=>m!=='Credit/Debit Card') as m}<option>{m}</option>{/each}</select>{#if errors.method}<span class="err">{errors.method}</span>{/if}</div>
     <div class="b-field" class:invalid={!!errors.amount}><label for="ep-amt" class="b-star">Payment Amount</label><input id="ep-amt" inputmode="decimal" bind:value={amount} placeholder="$0.00"/>{#if errors.amount}<span class="err">{errors.amount}</span>{/if}</div>
     <div class="b-field" class:invalid={!!errors.entry_date}><label for="ep-entry">Entry Date</label><input id="ep-entry" type="date" bind:value={entry}/>{#if errors.entry_date}<span class="err">{errors.entry_date}</span>{/if}</div>
     {#if method==='Check'}<div class="b-field"><label for="ep-check">Check Number</label><input id="ep-check" bind:value={check} maxlength="30"/></div><div class="b-field"><label for="ep-name">Name On Check</label><input id="ep-name" bind:value={nameOnCheck} maxlength="100"/></div>{/if}
    </div>
    <div class="b-actions"><a class="btn" href={href('')}>Cancel</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!identified||!method||!amount}>Submit Payment</button></div>
   </Card>
  </form>
 {/if}
</Page>
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>
