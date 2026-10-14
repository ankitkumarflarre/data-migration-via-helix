<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {billing,errorText,fieldErrors,type AccountView,type TermView} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {us} from '../format';
 import {flash} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 // Hold Activity For Account (#079–#081), Edit Hold Activity (#083) and Hold
 // Billing for Policy Term (#060). Holds start on the system date.
 let {view,term,holdID}:{view:AccountView;term:TermView|null;holdID:string}=$props();
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 const existing=$derived(holdID?view.holds.find(h=>h.id===holdID)??null:term?.hold??null);
 let kind=$state<'individual'|'event'>('individual'), type=$state(''), reason=$state(''), end=$state(''), eventID=$state(''), resume=$state(codes.holdResumeOptions[0]);
 let error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 const back=$derived(term?href(acct(n,'term/'+term.term.id)):href(acct(n)));
 const title=$derived(existing?`Edit Hold Activity for ${term?'Policy Term '+term.term.policy_number:'Account '+n}`:term?`Hold Billing for Policy Term ${term.term.policy_number} ${us(term.term.effective)} - ${us(term.term.expiration)}`:`Hold Activity For Account ${n}`);
 const ready=$derived(kind==='event'?!!eventID:!!type&&!!reason);
 $effect(()=>{if(existing)end=existing.end??'';});
 async function place(){busy=true;error='';errors={};try{const r=await billing.placeHold(n,{version:view.account.version,kind,term_id:term?.term.id??'',event_id:kind==='event'?eventID:'',type,reason,end});flash(r.message);go(term?acct(n,'term/'+term.term.id):acct(n));}catch(e){error=errorText(e);errors=fieldErrors(e);}finally{busy=false;}}
 async function update(action:'save'|'resume'){if(!existing)return;busy=true;error='';errors={};try{const m=await billing.updateHold(n,existing.id,{version:view.account.version,action,end,resume_option:resume});flash(m.message);go(term?acct(n,'term/'+term.term.id):acct(n));}catch(e){error=errorText(e);errors=fieldErrors(e);}finally{busy=false;}}
</script>
<Page {title} cards>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 {#if existing}
  <Card title="Hold" icon="shield">
   <div class="b-form">
    <div class="b-field"><span class="b-label">Hold Type</span><span class="b-ro">{existing.type}</span></div>
    <div class="b-field"><span class="b-label">Hold Reason</span><span class="b-ro">{existing.reason}</span></div>
    <div class="b-field"><span class="b-label">Start hold on</span><span class="b-ro">{us(existing.start)}</span></div>
    <div class="b-field" class:invalid={!!errors.end}><label for="eh-end">End hold on</label><input id="eh-end" type="date" bind:value={end} min={existing.start}/>{#if errors.end}<span class="err">{errors.end}</span>{/if}</div>
   </div>
  </Card>
  <Card title="Resume Options" subtitle={`Adjust due dates moves every open installment due on or after ${us(existing.start)} by the days the hold lasted.`} icon="calendar">
   <fieldset class="b-radios"><legend class="sr">Resume options</legend>{#each codes.holdResumeOptions as o}<label><input type="radio" bind:group={resume} value={o}/>{o}</label>{/each}</fieldset>
   <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span><button class="btn" disabled={busy} onclick={()=>update('save')}>Save</button><button class="btn primary" disabled={busy} onclick={()=>update('resume')}>Resume</button></div>
  </Card>
 {:else}
  {#if !term}
   <Card title="Hold Options" icon="shield">
    <fieldset class="b-radios"><legend class="sr">Hold options</legend><label><input type="radio" bind:group={kind} value="individual"/>This is an individual account hold</label><label><input type="radio" bind:group={kind} value="event"/>This hold is part of an area hold event</label></fieldset>
   </Card>
  {/if}
  {#if kind==='event'}
   <Card title="Hold Billing/Follow-up" subtitle="Attach this account to a current hold event." icon="calendar">
    <div class="b-grid"><table><thead><tr><th><span class="sr">Select</span></th><th>Event Name</th><th>Description</th><th>Type</th><th>Reason</th><th>Start Date</th><th>End Date</th></tr></thead><tbody>
     {#each view.hold_events as ev (ev.id)}<tr><td><input type="radio" name="hev" aria-label={'Attach to '+ev.name} bind:group={eventID} value={ev.id}/></td><td>{ev.name}</td><td>{ev.description}</td><td>{ev.type}</td><td>{ev.reason}</td><td>{us(ev.start)}</td><td>{us(ev.end)}</td></tr>{:else}<tr><td colspan="7" class="empty">No current hold events.</td></tr>{/each}
    </tbody></table></div>
    {#if errors.event_id}<p class="b-err">{errors.event_id}</p>{/if}
    <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span><button class="btn primary" disabled={busy||!ready} onclick={place}>Place Hold</button></div>
   </Card>
  {:else}
   <Card title="Hold details" icon="rules">
    <div class="b-form">
     <div class="b-field" class:invalid={!!errors.type}><label for="h-type">Hold Type</label><select id="h-type" bind:value={type}><option value="">Select Type</option>{#each codes.holdTypes as t}<option>{t}</option>{/each}</select>{#if errors.type}<span class="err">{errors.type}</span>{/if}</div>
     <div class="b-field" class:invalid={!!errors.reason}><label for="h-reason">Hold Reason</label><select id="h-reason" bind:value={reason}><option value="">Select Type</option>{#each codes.holdReasons as r}<option>{r}</option>{/each}</select>{#if errors.reason}<span class="err">{errors.reason}</span>{/if}</div>
     <div class="b-field"><span class="b-label">Start hold on</span><span class="b-ro">{us(view.system_date)}</span></div>
     <div class="b-field" class:invalid={!!errors.end}><label for="h-end">End hold on</label><input id="h-end" type="date" bind:value={end} min={view.system_date}/>{#if errors.end}<span class="err">{errors.end}</span>{:else}<small class="b-note">Leave blank for an indefinite hold.</small>{/if}</div>
    </div>
    <div class="b-actions"><a class="btn" href={back}>Cancel</a><span class="spacer"></span><button class="btn primary" disabled={busy||!ready} onclick={place}>Place Hold</button></div>
   </Card>
  {/if}
 {/if}
</Page>
<style>.sr{position:absolute;width:1px;height:1px;overflow:hidden;clip:rect(0 0 0 0)}</style>
