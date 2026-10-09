<script lang="ts">
 import {fieldValue,mapURL} from './field-values';
 import {matches,type FormField,type Value} from './quote-model';
 let {field,value=null,context,invalid='',id,rowIndex=0,onchange}:{field:FormField;value?:Value;context:Record<string,Value>;invalid?:string;id:string;rowIndex?:number;onchange:(value:Value)=>void}=$props();
 const computed=$derived(fieldValue(field,context,rowIndex));
 const locked=$derived(!!field.disabledWhen&&matches(field.disabledWhen,context));
 const link=$derived(field.link?mapURL(field.link,context):undefined);
 const required=$derived(field.required&&matches(field.requiredWhen,context));
 function change(e:Event){const el=e.currentTarget as HTMLInputElement;if(field.type==='integer'||field.type==='number')onchange(el.value===''?null:Number(el.value));else onchange(el.value);}
</script>
<div data-source-ref={field.sourceRefs?.join(' ')} class="form-field" class:invalid={!!invalid}>
 <label for={id}>{field.label}{#if required}<span class="required"> *</span>{/if}</label>
 {#if field.link}{#if link}<a {id} href={link} target="_blank" rel="noopener noreferrer">Open {field.label} ↗</a>{:else}<output {id}>Enter a mailing address to view the map</output>{/if}
 {:else if field.readOnly}<output id={id}>{computed===true?'Yes':computed===false?'No':computed??field.unavailable??'Not provided'}</output>
 {:else if field.type==='boolean'}<select disabled={locked} {id} value={value===true?'true':value===false?'false':''} aria-invalid={!!invalid} aria-describedby={invalid?id+'-error':undefined} onchange={e=>onchange(e.currentTarget.value===''?null:e.currentTarget.value==='true')}><option value="">Select…</option><option value="true">Yes</option><option value="false">No</option></select>
 {:else if field.options}<select disabled={locked} {id} value={String(value??'')} aria-invalid={!!invalid} aria-describedby={invalid?id+'-error':undefined} onchange={change}><option value="">Select…</option>{#each field.options as option}<option value={option}>{option==='N'?'None / No':option==='Y'?'Yes':option}</option>{/each}</select>
 {:else if field.type==='textarea'}<textarea disabled={locked} {id} value={String(value??'')} rows="3" maxlength={field.maxLength??2000} aria-invalid={!!invalid} aria-describedby={invalid?id+'-error':undefined} oninput={change}></textarea>
 {:else}<input disabled={locked} {id} type={field.type==='date'?'date':field.type==='email'?'email':['integer','number'].includes(field.type)?'number':'text'} value={value??''} min={field.min} max={field.max} step={field.type==='integer'?1:'any'} maxlength={field.maxLength??2000} aria-invalid={!!invalid} aria-describedby={invalid?id+'-error':undefined} oninput={change}/>{/if}
 {#if field.note}<small class="note">{field.note}</small>{/if}
 {#if invalid}<span class="error" id={id+'-error'}>{invalid}</span>{/if}
</div>
<style>
 .note{font-size:11px;color:var(--text-muted);line-height:1.5}output{font-size:12px;overflow-wrap:anywhere}a{font-size:12px}.form-field{display:flex;flex-direction:column;gap:7px;min-width:0}label{font-size:11px;font-weight:500;line-height:1.4}input,select,textarea{width:100%;min-height:36px;border:1px solid var(--border-strong);border-radius:7px;padding:8px 10px;background:var(--surface);color:var(--text);font:400 12px var(--font)}textarea{resize:vertical}output{padding:10px;background:var(--surface-2);border-radius:7px}.required,.error{color:var(--danger)}.error{font-size:12px}.invalid input,.invalid select,.invalid textarea{border-color:var(--danger)}
</style>
