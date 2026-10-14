<script lang="ts">
 import {billing,errorText,fieldErrors} from '../api';
 import {codes} from '../codes';
 let {oncreated,oncancel,kind='Person'}:{oncreated:(id:string,name:string)=>void;oncancel:()=>void;kind?:string}=$props();
 // svelte-ignore state_referenced_locally
 let k=$state(kind), name=$state<Record<string,string>>({first:'',middle:'',last:'',organization:''}), address=$state<Record<string,string>>({line1:'',line2:'',city:'',state:'',zip:''});
 let error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 async function save(e:Event){
  e.preventDefault();busy=true;error='';errors={};
  const n=Object.fromEntries(Object.entries(name).filter(([key,v])=>v.trim()&&(k==='Organization'?key==='organization':key!=='organization')));
  try{const p=await billing.createParty({kind:k,name:n,address:Object.fromEntries(Object.entries(address).filter(([,v])=>v.trim()))});oncreated(p.id,p.name);}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}
 }
</script>
<form class="box" onsubmit={save} aria-label="Create a new party">
 <div class="b-field-head"><strong>Create a new party</strong></div>
 <p class="b-note">The walkthrough does not open the new-party form; this asks for a name and a legal address only.</p>
 {#if error}<p class="b-err" role="alert">{error}</p>{/if}
 <div class="b-form">
  <div class="b-field"><label for="np-kind">Party Type</label><select id="np-kind" bind:value={k}><option>Person</option><option>Organization</option></select></div>
  {#if k==='Organization'}<div class="b-field" class:invalid={!!errors.organization}><label for="np-org" class="b-star">Organization Name</label><input id="np-org" bind:value={name.organization}/>{#if errors.organization}<span class="err">{errors.organization}</span>{/if}</div>
  {:else}
   <div class="b-field" class:invalid={!!errors.first}><label for="np-first" class="b-star">First</label><input id="np-first" bind:value={name.first}/>{#if errors.first}<span class="err">{errors.first}</span>{/if}</div>
   <div class="b-field"><label for="np-mid">Middle</label><input id="np-mid" bind:value={name.middle}/></div>
   <div class="b-field" class:invalid={!!errors.last}><label for="np-last" class="b-star">Last</label><input id="np-last" bind:value={name.last}/>{#if errors.last}<span class="err">{errors.last}</span>{/if}</div>
  {/if}
  <div class="b-field" class:invalid={!!errors.line1}><label for="np-a1" class="b-star">Address</label><input id="np-a1" bind:value={address.line1}/>{#if errors.line1}<span class="err">{errors.line1}</span>{/if}</div>
  <div class="b-field"><label for="np-a2">Address 2</label><input id="np-a2" bind:value={address.line2}/></div>
  <div class="b-field" class:invalid={!!errors.city}><label for="np-city" class="b-star">City</label><input id="np-city" bind:value={address.city}/>{#if errors.city}<span class="err">{errors.city}</span>{/if}</div>
  <div class="b-field" class:invalid={!!errors.state}><label for="np-state" class="b-star">State</label><select id="np-state" bind:value={address.state}><option value="">(Select)</option>{#each codes.states as s}<option>{s}</option>{/each}</select>{#if errors.state}<span class="err">{errors.state}</span>{/if}</div>
  <div class="b-field" class:invalid={!!errors.zip}><label for="np-zip" class="b-star">Zip Code</label><input id="np-zip" inputmode="numeric" maxlength="5" bind:value={address.zip}/>{#if errors.zip}<span class="err">{errors.zip}</span>{/if}</div>
 </div>
 <div class="b-actions"><button class="btn" type="button" onclick={oncancel}>Cancel</button><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy}>Save Party</button></div>
</form>
<style>.box{display:grid;gap:14px;padding:16px;border:1px solid var(--section-border);border-radius:8px;background:var(--surface-2)}</style>
