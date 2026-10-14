<script lang="ts">
 import {BillingError,errorText,type AccountView,type PaymentResult} from '../api';
 import {codes} from '../codes';
 import {usd} from '../format';
 import Modal from '../ui/Modal.svelte';
 // Simulated Global Payments (Heartland) portal (#067–#070). Nothing leaves
 // this machine; only the published test card numbers are accepted and only
 // the card brand, last four digits and name on card are stored.
 let {view,amount,pay,onfinish,oncancel}:{view:AccountView;amount:number;pay:(card:Record<string,unknown>)=>Promise<PaymentResult>;onfinish:(r:PaymentResult)=>void;oncancel:()=>void}=$props();
 // svelte-ignore state_referenced_locally
 const fee=Math.round(amount*codes.cardFeeBasisPoints/10000), names=(view.payor?.name??'').split(' ');
 let card=$state({number:'',name:'',month:'',year:'',cvc:''});
 // svelte-ignore state_referenced_locally
 let holder=$state({first:view.payor?.first||names.slice(0,-1).join(' '),middle:'',last:view.payor?.last||(names.at(-1)??''),address:view.payor?.line1??'',city:(view.payor?.line2??'').split(',')[0]??'',state:((view.payor?.line2??'').split(', ')[1]??'').split(' ')[0]??'',zip:((view.payor?.line2??'').match(/\d{5}$/)??[''])[0],country:'United States',email:view.payor?.email??'',phone:'',mobile:''});
 let step=$state<'form'|'terms'|'done'>('form'), touched=$state(false), error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false), result=$state<PaymentResult|null>(null);
 // svelte-ignore state_referenced_locally
 const year=Number(view.system_date.slice(0,4));
 const required:Record<string,string>=$derived({'card.number':card.number.trim(),'card.name':card.name.trim(),'card.month':card.month,'card.year':card.year,'card.cvc':card.cvc.trim(),first:holder.first.trim(),last:holder.last.trim(),address:holder.address.trim(),city:holder.city.trim(),state:holder.state,zip:holder.zip.trim(),email:holder.email.trim()});
 const missing=$derived(Object.entries(required).filter(([,v])=>!v).map(([k])=>k));
 const msg=(k:string)=>errors[k]||(touched&&missing.includes(k)?'This field is required.':'');
 function submit(){touched=true;errors={};if(missing.length===0)step='terms';}
 async function accept(){
  busy=true;error='';
  try{result=await pay({number:card.number,name:card.name,month:card.month,year:card.year,cvc:card.cvc,email:holder.email,accepted_terms:true});step='done';}
  catch(e){error=errorText(e);errors=e instanceof BillingError?e.fields:{};step='form';}finally{busy=false;}
 }
</script>
<Modal title="globalpayments · Secure Payment (simulated)" onclose={step==='done'&&result?()=>onfinish(result!):oncancel}>
 {#if step==='done'&&result?.receipt}
  <div class="ok" role="status"><strong>✔ Payment Successful</strong></div>
  {#if result.receipt.email}<div class="ok">Your receipt has been sent to {result.receipt.email}</div>{/if}
  <dl class="receipt"><dt>Transaction Number</dt><dd>{result.receipt.transaction_number}</dd><dt>Base Amount</dt><dd>{usd(result.receipt.base)}</dd><dt>Processing Fee</dt><dd>{usd(result.receipt.fee)}</dd><dt>Total Amount</dt><dd>{usd(result.receipt.total)}</dd></dl>
  <div class="b-actions col"><button class="btn" disabled title="Simulated gateway: no email is sent">Email</button><button class="btn primary" onclick={()=>onfinish(result!)}>Finish</button></div>
 {:else}
  {#if error}<div class="b-banner error" role="alert"><span class="ico">!</span><div>{error}</div></div>{/if}
  <p class="b-note">Simulated gateway: use a published test card such as {codes.testCards[0].number} (Visa). Billing posts the base amount only; the processing fee is the customer's.</p>
  <div class="cols">
   <section class="box"><h3>Payment Information</h3>
    <div class="tab">Credit Card</div>
    <div class="b-field" class:invalid={!!msg('card.number')}><label for="cg-num" class="b-star">Card Number <span class="brands">VISA · MasterCard · DISCOVER · AMEX</span></label><input id="cg-num" inputmode="numeric" autocomplete="off" bind:value={card.number} aria-invalid={!!msg('card.number')}/>{#if msg('card.number')}<span class="err">{msg('card.number')}</span>{/if}</div>
    <div class="b-field" class:invalid={!!msg('card.name')}><label for="cg-name" class="b-star">Name on Card</label><input id="cg-name" autocomplete="off" bind:value={card.name}/>{#if msg('card.name')}<span class="err">{msg('card.name')}</span>{/if}</div>
    <div class="b-field" class:invalid={!!msg('card.month')}><label for="cg-month" class="b-star">Month</label><select id="cg-month" bind:value={card.month}><option value="">-Select-</option>{#each Array.from({length:12},(_,i)=>String(i+1)) as m}<option value={m}>{m.padStart(2,'0')}</option>{/each}</select>{#if msg('card.month')}<span class="err">{msg('card.month')}</span>{/if}</div>
    <div class="pair">
     <div class="b-field" class:invalid={!!msg('card.year')}><label for="cg-year" class="b-star">Year</label><select id="cg-year" bind:value={card.year}><option value="">-Select-</option>{#each Array.from({length:12},(_,i)=>String(year+i)) as y}<option>{y}</option>{/each}</select>{#if msg('card.year')}<span class="err">{msg('card.year')}</span>{/if}</div>
     <div class="b-field" class:invalid={!!msg('card.cvc')}><label for="cg-cvc" class="b-star">Security Code</label><input id="cg-cvc" inputmode="numeric" autocomplete="off" maxlength="4" placeholder="CVC" bind:value={card.cvc}/>{#if msg('card.cvc')}<span class="err">{msg('card.cvc')}</span>{/if}</div>
    </div>
    <dl class="summary"><dt>Amount</dt><dd>{usd(amount)}</dd><dt>Processing Fee</dt><dd>{usd(fee)}</dd><dt>Total</dt><dd>{usd(amount+fee)}</dd></dl>
   </section>
   <section class="box"><h3>Card Holder Information</h3>
    {#each [['first','First Name',true],['middle','Middle Name',false],['last','Last Name',true],['address','Address',true],['city','City',true]] as const as [k,label,req]}
     <div class="b-field" class:invalid={!!msg(k)}><label for={'cg-'+k} class:b-star={req}>{label}</label><input id={'cg-'+k} bind:value={holder[k]}/>{#if msg(k)}<span class="err">{msg(k)}</span>{/if}</div>
    {/each}
    <div class="pair">
     <div class="b-field" class:invalid={!!msg('state')}><label for="cg-state" class="b-star">State</label><select id="cg-state" bind:value={holder.state}><option value=""></option>{#each codes.states as s}<option>{s}</option>{/each}</select>{#if msg('state')}<span class="err">{msg('state')}</span>{/if}</div>
     <div class="b-field" class:invalid={!!msg('zip')}><label for="cg-zip" class="b-star">Postal Code</label><input id="cg-zip" bind:value={holder.zip}/>{#if msg('zip')}<span class="err">{msg('zip')}</span>{/if}</div>
    </div>
    <div class="b-field"><label for="cg-country">Country</label><select id="cg-country" bind:value={holder.country}><option>United States</option></select></div>
    <div class="b-field" class:invalid={!!msg('email')}><label for="cg-email" class="b-star">Email</label><input id="cg-email" type="email" bind:value={holder.email}/>{#if msg('email')}<span class="err">{msg('email')}</span>{/if}</div>
    <div class="b-field"><label for="cg-phone">Phone Number</label><input id="cg-phone" type="tel" bind:value={holder.phone}/></div>
    <div class="b-field"><label for="cg-mobile">Mobile Phone Number</label><input id="cg-mobile" type="tel" bind:value={holder.mobile}/></div>
   </section>
  </div>
  <div class="b-actions split"><button class="btn" onclick={oncancel}>Cancel</button><button class="btn primary" disabled={busy} onclick={submit}>Submit Payment</button></div>
  <p class="brand">globalpayments</p>
  {#if step==='terms'}
   <div class="terms" role="alertdialog" aria-modal="true" aria-label="Payment terms">
    <div class="terms-box">
     <p>By accepting, I authorize this payment. If it is returned or declined by my card issuer, I remain responsible for the amount owed as though I had not attempted the payment, and I may be liable for additional fees and penalties to the extent of applicable law.</p>
     <h4>RECEIPT</h4><p>A receipt can be printed after the payment is accepted and serves as evidence of payment. If you provide an email address, a receipt is emailed to you after the payment is processed.</p>
     <h4>CONTACT</h4><p>If you wish to change the payment after submission, please contact SafePoint Insurance for assistance.</p>
     <div class="b-actions end"><button class="btn ghost" disabled={busy} onclick={()=>step='form'}>Decline</button><button class="btn primary" disabled={busy} onclick={accept}>{busy?'Processing…':'Accept'}</button></div>
    </div>
   </div>
  {/if}
 {/if}
</Modal>
<style>
 .cols{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:14px}.box{border:1px solid var(--border);border-radius:8px;padding:12px;display:grid;gap:10px;align-content:start}.box h3{font-size:15px;margin:0}
 .tab{justify-self:start;padding:6px 12px;border:1px solid var(--border);border-bottom:0;border-radius:6px 6px 0 0;font-size:13px}.brands{font-size:10px;color:var(--text-muted);margin-left:6px}
 .pair{display:grid;grid-template-columns:1fr 1fr;gap:10px}.summary,.receipt{display:grid;grid-template-columns:1fr auto;gap:6px 12px;margin:0;font-size:14px;border:1px solid var(--border);border-radius:6px;padding:10px}dd{margin:0;text-align:right}
 .split{justify-content:space-between}.end{justify-content:flex-end}.col{flex-direction:column;align-items:stretch}.brand{text-align:center;font-weight:800;font-size:20px;color:var(--info);margin:0}
 .ok{background:var(--success-soft);color:var(--success);padding:12px;border-radius:6px;font-size:15px}
 .terms{position:fixed;inset:0;background:rgb(0 0 0 / .35);display:grid;place-items:center;z-index:70;padding:16px}.terms-box{background:var(--surface);border-radius:8px;padding:18px;max-width:560px;display:grid;gap:8px;font-size:14px}.terms-box h4{margin:6px 0 0}
</style>
