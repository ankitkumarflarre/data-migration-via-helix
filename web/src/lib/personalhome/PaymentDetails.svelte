<script lang="ts">
 import Icon from './WorkspaceIcon.svelte';
 import QuoteField from './QuoteField.svelte';
 import InstallmentSchedule from './InstallmentSchedule.svelte';
 import type {FormField,Value} from './quote-model';
 let {fields,values,errors,busy,dirty,policyNumber,onchange,onsave,onreset,onprevious,onnext}:{fields:FormField[];values:Record<string,Value>;errors:Record<string,string>;busy:boolean;dirty:boolean;policyNumber:string;onchange:(key:string,value:Value)=>void;onsave:()=>void;onreset:()=>void;onprevious:()=>void;onnext:()=>void}=$props();
 let showSchedule=$state(false);
 const radios=$derived(fields.filter(f=>f.control==='radio'));
 const inputs=$derived(fields.filter(f=>f.control!=='radio'));
</script>

<section class="payment">
 <h3>Payment</h3>
 {#each radios as field}
  <fieldset class="choice" class:invalid={!!errors[field.key]} aria-describedby={errors[field.key]?'pay-'+field.key+'-error':undefined}>
   <legend>{field.label}{#if field.required}<span class="required"> *</span>{/if}</legend>
   {#each field.options??[] as option}<label class="radio"><input type="radio" name={field.key} value={option} checked={values[field.key]===option} disabled={busy} onchange={()=>onchange(field.key,option)}/>{option}</label>{/each}
   {#if errors[field.key]}<span class="error" id={'pay-'+field.key+'-error'}>{errors[field.key]}</span>{/if}
  </fieldset>
 {/each}
 <fieldset class="grid" disabled={busy}>{#each inputs as field}<QuoteField {field} id={'pay-'+field.key} value={values[field.key]} context={values} invalid={errors[field.key]} onchange={value=>onchange(field.key,value)}/>{/each}</fieldset>
 <div class="row"><button class="btn primary" disabled={busy} onclick={onsave}>{busy?'Saving…':'Save'}</button><button class="btn" disabled={busy||!dirty} onclick={onreset}>Reset</button></div>
 {#if showSchedule}<InstallmentSchedule id="payment-schedule" {policyNumber} plan={String(values['Billing.PaymentPlan']??'')} due={String(values['Billing.InvoiceDueDate']??'')} effective={String(values['PolicyInput.EffectiveDate']??'')} expiration={String(values['PolicyInput.ExpirationDate']??'')}/>
 {/if}
 <div class="nav"><button class="btn" disabled={busy} onclick={onprevious}>‹ Previous</button><span class="spacer"></span><button class="btn" aria-expanded={showSchedule} aria-controls="payment-schedule" onclick={()=>showSchedule=!showSchedule}><Icon name="calendar" size={16}/>Schedule</button><button class="btn" disabled={busy} onclick={onnext}>Next ›</button></div>
</section>

<style>
 .payment{display:grid;gap:18px}h3{font-size:16px;margin:0}h4{font-size:13px;margin:0 0 10px}
 .choice{border:0;margin:0;padding:0;display:grid;gap:12px}.choice legend{font-size:13px;padding:0;margin-bottom:12px}.required,.error{color:var(--danger)}.error{font-size:12px}
 .radio{display:inline-flex;align-items:center;gap:10px;font-size:14px;cursor:pointer}.radio input{width:20px;height:20px;margin:0;accent-color:var(--accent)}
 .grid :global(label){font-size:13px;font-weight:400}.grid :global(input),.grid :global(select){min-height:44px;font-size:14px}
 .grid{border:0;margin:0;padding:0;min-width:0;display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:18px 28px}
 .row,.nav{display:flex;gap:12px;align-items:center;flex-wrap:wrap}.nav{border-top:1px solid var(--border);padding-top:18px}.nav .btn{display:inline-flex;align-items:center;gap:8px}.spacer{flex:1}
 @media(max-width:540px){.grid{grid-template-columns:1fr}.nav .spacer{display:none}.nav .btn{flex:1;justify-content:center}}
</style>
