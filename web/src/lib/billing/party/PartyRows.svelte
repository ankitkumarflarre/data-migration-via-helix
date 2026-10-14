<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type PartyView,type PartyRow} from '../api';
 import {href} from '../routes';
 import {us} from '../format';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import {kinds} from './kinds';
 let {party,kind,reload}:{party:PartyView;kind:string;reload:()=>Promise<void>}=$props();
 // svelte-ignore state_referenced_locally
 const k=kinds[kind];
 let mode=$state<''|'add'|'correct'|'replace'>(''), row=$state<PartyRow|null>(null), f=$state<Record<string,string>>({}), eff=$state(''), exp=$state('');
 let error=$state(''), notice=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false), removing=$state('');
 const rows=$derived(party.rows[kind]??[]);
 const current=$derived(rows.filter(r=>r.current));
 const groups=$derived(k.addressGroups?[...new Set(current.map(r=>r.fields.kind))]:['']);
 function open(m:'add'|'correct'|'replace',r:PartyRow|null=null){mode=m;row=r;errors={};error='';notice='';f=r?{...r.fields}:{country:'United States'};if(k.ownDates){eff=r?.effective??'';exp=r&&r.expiration!=='9999-12-31'?r.expiration:'';}}
 async function send(body:Record<string,unknown>,done:string){busy=true;error='';errors={};try{await billing.saveRow(party.id,kind,{version:party.version,...body});await reload();notice=done;mode='';removing='';}catch(e){error=errorText(e);errors=fieldErrors(e);}finally{busy=false;}}
 function save(e:Event){e.preventDefault();const fields=Object.fromEntries(Object.entries(f).filter(([key,v])=>v!==''&&k.fields.some(x=>x.key===key)));send({mode,row_id:row?.id??'',fields,effective:eff,expiration:exp},mode==='replace'?`Replaced. The new row is effective ${us(party.system_date)}; the old one ended that day.`:mode==='correct'?'Corrected. The effective date is unchanged.':'Saved.');}
 const showDate=$derived(mode==='correct'?row?.effective??'':party.system_date);
</script>
<Page title={k.title} date={false} required={!!mode} cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}{#if notice}<Banner>{notice}</Banner>{/if}
 {#if mode}
  <Card title={mode==='add'?k.add:mode==='replace'?'Replace':'Correct'} icon="rules">
  <form class="b-stack" onsubmit={save} aria-label={k.add}>
   {#if mode==='replace'}<p class="b-note">Replace ends the current row on {us(party.system_date)} and saves these values as a new row from that date.</p>{:else if mode==='correct'}<p class="b-note">Correct fixes the current row; its effective date stays {us(row?.effective)}.</p>{/if}
   <div class="b-form">
    {#each k.fields as fd}
     <div class="b-field" class:invalid={!!errors[fd.key]}>
      <label for={'pr-'+fd.key} class:b-star={fd.required}>{fd.label}</label>
      {#if fd.type==='checkbox'}<input id={'pr-'+fd.key} type="checkbox" class="box" checked={f[fd.key]==='Yes'} onchange={e=>f[fd.key]=e.currentTarget.checked?'Yes':'No'}/>
      {:else if fd.options}<select id={'pr-'+fd.key} bind:value={f[fd.key]}><option value="">(Select)</option>{#each fd.options as o}<option>{o}</option>{/each}</select>
      {:else if fd.type==='textarea'}<textarea id={'pr-'+fd.key} rows="2" bind:value={f[fd.key]}></textarea>
      {:else}<input id={'pr-'+fd.key} bind:value={f[fd.key]} placeholder={fd.placeholder}/>{/if}
      {#if errors[fd.key]}<span class="err">{errors[fd.key]}</span>{/if}
     </div>
    {/each}
    {#if k.ownDates}
     <div class="b-field" class:invalid={!!errors.effective}><label for="pr-eff">Effective Date</label><input id="pr-eff" type="date" bind:value={eff}/>{#if errors.effective}<span class="err">{errors.effective}</span>{/if}</div>
     <div class="b-field" class:invalid={!!errors.expiration}><label for="pr-exp">Expiration Date</label><input id="pr-exp" type="date" bind:value={exp}/>{#if errors.expiration}<span class="err">{errors.expiration}</span>{/if}</div>
    {:else}
     <div class="b-field"><span class="b-label">Effective Date</span><output>{us(showDate)}</output></div><div class="b-field"><span class="b-label">Expiration Date</span><output>12/31/9999</output></div>
    {/if}
   </div>
   {#if kind==='address'&&row}<p class="b-note">Associations: This is the {row.fields.kind} Address as of {us(row.effective)}. <span title="Associations beyond the address type are not in the walkthrough">Update associations</span></p>{/if}
   <div class="b-actions"><button class="btn" type="button" onclick={()=>mode=''}>Cancel</button><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy}>Save</button></div>
  </form>
  </Card>
 {/if}
 <Card title={k.saved}>
  {#snippet tools()}{#if !mode}<button class="btn sm primary" onclick={()=>open('add')}>{k.add}</button>{/if}{/snippet}
  {#if !current.length}<p class="b-label">No Records Found</p>{/if}
  {#each groups as g}
   {@const gr=current.filter(r=>!k.addressGroups||r.fields.kind===g)}
   {#if gr.length}
    {#if g}<p class="b-label">{g} Addresses</p>{/if}
    <div class="b-grid"><table><thead><tr>{#each k.columns as [c]}<th>{c}</th>{/each}<th>{k.addressGroups?'as of':'Effective'}</th>{#each k.modes as m}<th><span class="sr">{m}</span></th>{/each}</tr></thead><tbody>
     {#each gr as r (r.id)}<tr>{#each k.columns as [,fn]}<td class="wrap">{fn(r.fields)}</td>{/each}<td>{k.addressGroups?us(r.effective):`${r.effective} To ${r.expiration}`}</td>
      {#each k.modes as m}<td>{#if m==='remove'}{#if removing===r.id}<button class="btn sm danger" disabled={busy} onclick={()=>send({mode:'remove',row_id:r.id},`Removed. The name ended on ${us(party.system_date)}.`)}>Confirm</button>{:else}<button class="b-link" onclick={()=>removing=r.id}>Remove</button>{/if}{:else}<button class="b-link" onclick={()=>open(m,r)}>{m==='correct'?(kind==='email'?'Edit':'Correct'):'Replace'}</button>{/if}</td>{/each}</tr>{/each}
    </tbody></table></div>
   {/if}
  {/each}
  {#if rows.length>current.length}<details><summary class="b-note">History ({rows.length-current.length} ended)</summary><ul class="b-list">{#each rows.filter(r=>!r.current) as r}<li>{k.columns.map(([,fn])=>fn(r.fields)).join(' · ')} — {r.effective} To {r.expiration}</li>{/each}</ul></details>{/if}
 </Card>
 <div class="b-bar"><a class="btn" href={href('party/'+party.id)}>Back</a></div>
</Page>
<style>.box{width:16px;height:16px;accent-color:var(--accent)}.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>
