<script lang="ts">
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Card from '../ui/Card.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText,fieldErrors,type PaymentView,type Identity} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {usd} from '../format';
 import {flash,flashLinked} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import PaymentStrip from './PaymentStrip.svelte';
 import IdentityFields from './IdentityFields.svelte';
 let {id}:{id:string}=$props();
 let p=$state<PaymentView|null>(null), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 let mode=$state<'known'|'unidentified'>('known'), by=$state('Account Number'), number=$state(''), reason=$state(codes.reverseReasons[0]), confirm=$state(false);
 let ident=$state<Identity>({last:'',first:'',middle:'',account_ref:'',policy_ref:'',other:''});
 onMount(load);
 async function load(){try{p=await billing.payment(id);ident={...ident,...(p.identity??{})};number=p.identity?.account_ref??'';}catch(e){error=errorText(e);}}
 async function run(fn:()=>Promise<void>){busy=true;error='';errors={};try{await fn();}catch(e){error=errorText(e);errors=fieldErrors(e);}finally{busy=false;}}
 const apply=(e:Event)=>{e.preventDefault();run(async()=>{const r=await billing.applyUnidentified(p!.id,p!.version,number);flashLinked(r.message,r.payment.account_number!,href(acct(r.payment.account_number!)));go('unidentified');});};
 const update=(e:Event)=>{e.preventDefault();run(async()=>{await billing.updateReference(p!.id,{version:p!.version,...ident});flash('Payment reference updated.');await load();});};
 const reverse=()=>run(async()=>{await billing.reverse(p!.id,p!.version,reason);flash(`${usd(p!.amount)} Payment was reversed. Reason: ${reason.replace(/^RP-/,'')}`);go('unidentified');});
</script>
<Page title="Unidentified Payment" subtitle="Identify the account this payment belongs to, update its reference, or reverse it." cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 {#if p}
  <Card title="Payment Details" icon="dollar"><PaymentStrip payment={p}/></Card>
  {#if p.status!=='Active'||!p.unidentified}<Banner kind="warn">This payment is {p.unidentified?p.status.toLowerCase():'no longer unidentified'}. <a href={href('payment/'+p.id)}>View payment details</a></Banner>
  {:else}
   <Card title="Payment Reference" icon="search">
    <fieldset class="b-radios"><legend class="sr">Payment reference</legend>
     <label><input type="radio" bind:group={mode} value="known"/>I know the account, policy, arrangement or name</label>
     <label><input type="radio" bind:group={mode} value="unidentified"/>This is an unidentified payment, not associated with an account, policy or arrangement</label>
    </fieldset>
    {#if mode==='known'}
     <form class="b-stack" onsubmit={apply}>
      <div class="b-form">
       <div class="b-field"><label for="ud-by">Search for Account by</label><select id="ud-by" bind:value={by}>{#each codes.accountSearchBy as o}<option>{o}</option>{/each}</select></div>
       <div class="b-field" class:invalid={!!errors.account}><label for="ud-num">Enter Number</label><input id="ud-num" bind:value={number}/>{#if errors.account}<span class="err">{errors.account}</span>{/if}</div>
      </div>
      <div class="b-actions"><span class="b-note">Applies the payment to the account found.</span><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!number.trim()}><Icon name="search" size={14}/>Search</button></div>
     </form>
    {:else}
     <form class="b-stack" onsubmit={update}><IdentityFields bind:value={ident} {errors} prefix="ud-" labels="detail"/><div class="b-actions"><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy}>Update Reference</button></div></form>
    {/if}
   </Card>
   <Card title="Reverse This Payment" subtitle="Reversal removes the payment from the unidentified list." icon="transfer">
    <div class="b-form">
     <div class="b-field"><label for="ud-reason">Reverse reason</label><select id="ud-reason" bind:value={reason}>{#each codes.reverseReasons as r}<option>{r}</option>{/each}</select></div>
    </div>
    <div class="b-actions">{#if confirm}<span>Reverse {usd(p.amount)}?</span><span class="spacer"></span><button class="btn sm" onclick={()=>confirm=false}>Keep</button><button class="btn danger sm" disabled={busy} onclick={reverse}>Reverse Payment</button>{:else}<span class="spacer"></span><button class="btn danger sm" onclick={()=>confirm=true}>Reverse Payment</button>{/if}</div>
   </Card>
  {/if}
 {/if}
 <div class="b-bar"><a class="btn" href={href('unidentified')}>Close</a></div>
</Page>
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>
