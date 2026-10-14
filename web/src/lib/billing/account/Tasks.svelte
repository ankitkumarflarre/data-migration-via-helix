<script lang="ts">
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Card from '../ui/Card.svelte';
 import Status from '../ui/Status.svelte';
 import {billing,errorText,fieldErrors,type AccountView,type Task} from '../api';
 import {href,acct,go} from '../routes';
 import {codes} from '../codes';
 import {us,created} from '../format';
 import {flash,today} from '../state.svelte';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let {view,reload,adding}:{view:AccountView;reload:()=>void;adding:boolean}=$props();
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 let f=$state({title:'',type:codes.taskTypes[0],due_date:today(),description:'',queue:'',assignee:''});
 let error=$state(''), errors=$state<Record<string,string>>({}), busy=$state(false);
 const blank={category:'',state:'',field:codes.taskSearchFields[0],value:''};
 let more=$state<{mine:boolean;pickup:boolean}>({mine:false,pickup:false}), mineF=$state({...blank}), pickF=$state({...blank}), applied=$state({mine:{...blank},pickup:{...blank}});
 let mineView=$state('all'), queue=$state(''), pickView=$state('all');
 const filter=(t:Task,q:typeof blank)=>(!q.category||t.category===q.category)&&(!q.state||t.status===q.state)&&(!q.value.trim()||t.account_number.toLowerCase().includes(q.value.trim().toLowerCase()));
 const mine=$derived(view.tasks.filter(t=>t.assignee==='Local User').filter(t=>mineView==='all'||t.due_date<=view.system_date).filter(t=>filter(t,applied.mine)));
 const pickup=$derived(view.tasks.filter(t=>!t.assignee&&(!queue||t.queue===queue)).filter(t=>pickView==='all'||t.due_date<=view.system_date).filter(t=>filter(t,applied.pickup)));
 async function create(e:Event){e.preventDefault();busy=true;error='';errors={};try{await billing.addTask(n,{version:view.account.version,...f});flash('Task created.');go(acct(n,'tasks'));}catch(err){error=errorText(err);errors=fieldErrors(err);}finally{busy=false;}}
 async function pick(t:Task){busy=true;error='';try{await billing.pickup(n,t.id,view.account.version);flash(`Task "${t.title}" was assigned to you.`);reload();go(acct(n,'tasks'));}catch(e){error=errorText(e);}finally{busy=false;}}
</script>
{#snippet filters(id:string,q:typeof blank,apply:()=>void,clear:()=>void)}
 <div class="b-form filters">
  <div class="b-field"><label for={id+'-cat'}>Category</label><select id={id+'-cat'} bind:value={q.category}><option value="">All</option>{#each codes.taskCategories as c}<option>{c}</option>{/each}</select></div>
  <div class="b-field"><label for={id+'-state'}>State</label><select id={id+'-state'} bind:value={q.state}><option value="">All</option>{#each codes.taskStates as s}<option>{s}</option>{/each}</select></div>
  <div class="b-field"><label for={id+'-field'}>Search</label><select id={id+'-field'} bind:value={q.field}>{#each codes.taskSearchFields as s}<option>{s}</option>{/each}</select></div>
  <div class="b-field"><label for={id+'-for'}>For</label><input id={id+'-for'} bind:value={q.value}/></div>
  <div class="b-actions full"><span class="spacer"></span><button class="btn sm" disabled={!q.category&&!q.state&&!q.value} onclick={clear}>Clear</button><button class="btn sm primary" onclick={apply}>Search</button></div>
 </div>
{/snippet}
{#snippet grid(rows:Task[],pickable:boolean)}
 <div class="b-grid"><table><thead><tr><th>Queue</th><th>Category</th><th>Title</th><th>Status</th><th>Reference #</th><th>Created</th><th>Due Date</th><th></th></tr></thead><tbody>
  {#each rows as t (t.id)}<tr><td>{t.queue}</td><td>{t.category}</td><td><span title={t.description}>{t.title}</span></td><td><Status value={t.status}/></td><td>{t.account_number}</td><td>{created(t.created)}</td><td>{us(t.due_date)}</td><td>{#if pickable}<button class="btn sm" disabled={busy} onclick={()=>pick(t)} aria-label={'Pick up task '+t.title}>Pick up</button>{/if}</td></tr>{:else}<tr><td colspan="8" class="empty">No results to display.</td></tr>{/each}
 </tbody></table></div>
{/snippet}
{#if adding}
 <Page title="New Task" subtitle="Create a follow-up task on this account." date={false} required cards>
  {#if error}<Banner kind="error">{error}</Banner>{/if}
  <Card title="Task" icon="check">
   <form class="b-stack" onsubmit={create}>
    <div class="b-form">
     <div class="b-field" class:invalid={!!errors.title}><label for="nt-title" class="b-star">Title</label><input id="nt-title" bind:value={f.title} maxlength="200"/>{#if errors.title}<span class="err">{errors.title}</span>{/if}</div>
     <div class="b-field"><span class="b-label">Account Number</span><span class="b-ro">{n}</span></div>
     <div class="b-field"><label for="nt-type">Task Type</label><select id="nt-type" bind:value={f.type}>{#each codes.taskTypes as t}<option>{t}</option>{/each}</select></div>
     <div class="b-field"><label for="nt-due">Due Date</label><input id="nt-due" type="date" bind:value={f.due_date}/></div>
     <div class="b-field" class:invalid={!!errors.queue}><label for="nt-queue" class="b-star">Queue</label><select id="nt-queue" bind:value={f.queue}><option value=""></option>{#each codes.taskQueues as q}<option>{q}</option>{/each}</select>{#if errors.queue}<span class="err">{errors.queue}</span>{/if}</div>
     <div class="b-field"><label for="nt-assignee">Assignee</label><select id="nt-assignee" bind:value={f.assignee}><option value="">(select)</option>{#each codes.assumed.taskAssignees as a}<option>{a}</option>{/each}</select></div>
     <div class="b-field full"><label for="nt-desc">Description</label><textarea id="nt-desc" rows="3" maxlength="2000" bind:value={f.description}></textarea></div>
    </div>
    <div class="b-actions"><a class="btn" href={href(acct(n,'tasks'))}>Cancel</a><span class="spacer"></span><button class="btn primary" type="submit" disabled={busy||!f.title.trim()||!f.queue}>Create Task</button></div>
   </form>
  </Card>
 </Page>
{:else}
 <Page title="Account Tasks" subtitle="Your tasks on this account and tasks waiting to be picked up." date={false} cards>
  {#snippet actions()}<a class="btn sm primary" href={href(acct(n,'tasks/new'))}><Icon name="plus" size={14}/>Add a Task</a>{/snippet}
  {#if error}<Banner kind="error">{error}</Banner>{/if}
  <Card title="My Tasks" icon="check">
   {#snippet tools()}<select aria-label="My tasks view" bind:value={mineView}><option value="all">View All Tasks</option><option value="due">View Tasks Due</option></select><button class="b-link" aria-expanded={more.mine} onclick={()=>more.mine=!more.mine}>More Options</button>{/snippet}
   {#if more.mine}{@render filters('mt',mineF,()=>applied.mine={...mineF},()=>{mineF={...blank};applied.mine={...blank};})}{/if}
   {@render grid(mine,false)}
  </Card>
  <Card title="Tasks Available for Pickup" icon="database">
   {#snippet tools()}<select aria-label="Queue" bind:value={queue}><option value="">View All Queues</option>{#each codes.taskQueues as q}<option>{q}</option>{/each}</select><select aria-label="Pickup tasks view" bind:value={pickView}><option value="all">View All Tasks</option><option value="due">View Tasks Due</option></select><button class="b-link" aria-expanded={more.pickup} onclick={()=>more.pickup=!more.pickup}>More Options</button>{/snippet}
   {#if more.pickup}{@render filters('pt',pickF,()=>applied.pickup={...pickF},()=>{pickF={...blank};applied.pickup={...blank};})}{/if}
   {@render grid(pickup,true)}
  </Card>
 </Page>
{/if}
<style>select{min-height:34px;font-size:12px;padding:4px 8px}.filters{padding:14px;border:1px solid var(--border);border-radius:8px;background:var(--surface-2)}.filters select{min-height:38px;font-size:14px}</style>