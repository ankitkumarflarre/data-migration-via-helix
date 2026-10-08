<script lang="ts">
 import {matches,type FormField,type Value} from './quote-model';
 let {field,value=null,context,invalid='',id,onchange}:{field:FormField;value?:Value;context:Record<string,Value>;invalid?:string;id:string;onchange:(value:Value)=>void}=$props();
 const required=$derived(field.required&&matches(field.requiredWhen,context));
 function change(e:Event){const el=e.currentTarget as HTMLInputElement;if(field.type==='integer'||field.type==='number')onchange(el.value===''?null:Number(el.value));else onchange(el.value);}
</script>
<div class="form-field" class:invalid={!!invalid}>
 <label for={id}>{field.label}{#if required}<span class="required"> *</span>{/if}</label>
 {#if field.readOnly}<output id={id}>{value??field.default??'Not available'}</output>
 {:else if field.type==='boolean'}<select {id} value={value===true?'true':value===false?'false':''} aria-invalid={!!invalid} aria-describedby={invalid?id+'-error':undefined} onchange={e=>onchange(e.currentTarget.value===''?null:e.currentTarget.value==='true')}><option value="">Select…</option><option value="true">Yes</option><option value="false">No</option></select>
 {:else if field.options}<select {id} value={String(value??'')} aria-invalid={!!invalid} aria-describedby={invalid?id+'-error':undefined} onchange={change}><option value="">Select…</option>{#each field.options as option}<option value={option}>{option==='N'?'None / No':option==='Y'?'Yes':option}</option>{/each}</select>
 {:else if field.type==='textarea'}<textarea {id} value={String(value??'')} rows="3" maxlength={field.maxLength??2000} aria-invalid={!!invalid} aria-describedby={invalid?id+'-error':undefined} oninput={change}></textarea>
 {:else}<input {id} type={field.type==='date'?'date':field.type==='email'?'email':['integer','number'].includes(field.type)?'number':'text'} value={value??''} min={field.min} max={field.max} step={field.type==='integer'?1:'any'} maxlength={field.maxLength??2000} aria-invalid={!!invalid} aria-describedby={invalid?id+'-error':undefined} oninput={change}/>{/if}
 {#if invalid}<span class="error" id={id+'-error'}>{invalid}</span>{/if}
</div>
<style>
 .form-field{display:flex;flex-direction:column;gap:7px;min-width:0}label{font-size:13px;font-weight:600;line-height:1.4}input,select,textarea{width:100%;min-height:40px;border:1px solid var(--border-strong);border-radius:7px;padding:9px 11px;background:var(--surface);color:var(--text);font:inherit}textarea{resize:vertical}output{padding:10px;background:var(--surface-2);border-radius:7px}.required,.error{color:var(--danger)}.error{font-size:12px}.invalid input,.invalid select,.invalid textarea{border-color:var(--danger)}
</style>
