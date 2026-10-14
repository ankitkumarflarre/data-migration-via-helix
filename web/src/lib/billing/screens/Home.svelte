<script lang="ts">
 import {onMount} from 'svelte';
 import {billing,errorText,type HomeView} from '../api';
 import {href,go,acct} from '../routes';
 import {codes} from '../codes';
 import {longDate,us} from '../format';
 import {ws,takeFlash} from '../state.svelte';
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import Banner from '../ui/Banner.svelte';
 import Status from '../ui/Status.svelte';
 let home=$state<HomeView|null>(null), error=$state(''), by=$state('Account Number'), q=$state(''), searched=$state('');
 let allNotes=$state(false), tab=$state<'recent'|'tasks'>('recent');
 const flash=takeFlash();
 onMount(async()=>{try{home=await billing.home();}catch(e){error=errorText(e);}});
 const slug=(name:string)=>name.toLowerCase().replace(/ /g,'-');
 async function find(e:Event){e.preventDefault();searched='';try{const hits=await billing.findAccounts(q,by);if(hits.length)go(acct(hits[0].number));else searched=`No billing account matches ${q.trim()}.`;}catch(err){searched=errorText(err);}}
 const activities=[['payments/search','Search For A Payment','search'],['payments/enter','Enter A Single Payment','dollar'],['unidentified','Unidentified Payments','filter'],['accounts/new','Create A New Account','plus'],['statements','Import Statements','file']];
 const notes=$derived((allNotes?home?.notifications:home?.notifications.slice(0,3))??[]);
</script>

<section class="home" aria-label="Home">
 <div class="head"><span class="eyebrow">Billing · Daily experience</span><h1>Billing dashboard</h1>{#if ws.system}<p class="b-date">The current system date is {longDate(ws.system.date)}</p>{/if}</div>
 {#if flash}<Banner kind={flash.kind}>{flash.text}{#if flash.link}<a href={flash.link.href}>{flash.link.label}</a>{flash.link.after??''}{/if}</Banner>{/if}
 {#if error}<Banner kind="error">{error}</Banner>{/if}

 <div class="b-kpis">
  <button type="button" class="b-kpi" class:selected={tab==='recent'} onclick={()=>tab='recent'}><span class="b-ico soft"><Icon name="clock" size={14}/></span><span class="b-kpi-body"><span class="b-kpi-top"><span class="b-kpi-label">Recently Accessed</span><strong>{home?.recent.length??'–'}</strong></span><small>billing accounts you opened</small></span></button>
  <button type="button" class="b-kpi" class:selected={tab==='tasks'} onclick={()=>tab='tasks'}><span class="b-ico warning"><Icon name="check" size={14}/></span><span class="b-kpi-body"><span class="b-kpi-top"><span class="b-kpi-label">Tasks Due</span><strong>{home?.tasks_due.length??'–'}</strong></span><small>assigned and due today</small></span></button>
  <button type="button" class="b-kpi" class:selected={allNotes} aria-expanded={allNotes} onclick={()=>allNotes=!allNotes}><span class="b-ico success"><Icon name="bell" size={14}/></span><span class="b-kpi-body"><span class="b-kpi-top"><span class="b-kpi-label">Notifications</span><strong>{home?.notifications.length??'–'}</strong></span><small>{allNotes?'showing all':'latest three shown'}</small></span></button>
 </div>

 <div class="split">
  <div class="main">
   <section class="widget">
    <div class="widget-header">
     <div><h2>Billing accounts</h2><p>Find an account, or pick up where you left off</p></div>
     <form class="b-inline" role="search" onsubmit={find}>
      <label class="sr-only" for="home-by">Find account by</label><select id="home-by" bind:value={by}>{#each codes.accountSearchBy as o}<option>{o}</option>{/each}</select>
      <div class="b-search"><Icon name="search" size={14}/><label class="sr-only" for="home-q">searching for</label><input id="home-q" type="search" bind:value={q} placeholder="Search accounts…" autocomplete="off"/><button type="submit" aria-label="Search" disabled={!q.trim()}><Icon name="arrow" size={14}/></button></div>
     </form>
    </div>
    <div class="widget-body">
     {#if searched}<p class="b-err" role="alert">{searched}</p>{/if}
     <div class="bw-tabs" role="tablist" aria-label="Account lists">
      <button role="tab" aria-selected={tab==='recent'} class:on={tab==='recent'} onclick={()=>tab='recent'}>Recently Accessed <span>{home?.recent.length??0}</span></button>
      <button role="tab" aria-selected={tab==='tasks'} class:on={tab==='tasks'} onclick={()=>tab='tasks'}>Tasks Due <span>{home?.tasks_due.length??0}</span></button>
     </div>
     <div class="b-grid">
      {#if tab==='recent'}
       <table><thead><tr><th>Account No</th><th>Name</th><th>Address</th><th>Status</th></tr></thead><tbody>
        {#each home?.recent??[] as r (r.number)}<tr><td><a href={href(acct(r.number))}>{r.number}</a></td><td>{r.name}</td><td class="wrap">{r.address}</td><td><Status value={r.status}/></td></tr>
        {:else}<tr><td colspan="4" class="empty">No accounts opened yet. Search above or open one from a payment.</td></tr>{/each}
       </tbody></table>
      {:else}
       <table><thead><tr><th>Task</th><th>Account No</th><th>Category</th><th>Due Date</th><th>Status</th></tr></thead><tbody>
        {#each home?.tasks_due??[] as t (t.id)}<tr><td><a href={href(acct(t.account_number,'tasks'))}>{t.title}</a></td><td>{t.account_number}</td><td>{t.category}</td><td>{us(t.due_date)}</td><td><Status value={t.status}/></td></tr>
        {:else}<tr><td colspan="5" class="empty">You have no tasks due today.</td></tr>{/each}
       </tbody></table>
      {/if}
     </div>
    </div>
   </section>

   <section class="widget">
    <div class="widget-header"><div><h2>Activities</h2><p>Start a billing task</p></div></div>
    <div class="widget-body actions">{#each activities as [path,label,icon]}<a class="action" href={href(path)}><span class="b-ico soft"><Icon name={icon} size={14}/></span>{label}<span class="go" aria-hidden="true">›</span></a>{/each}</div>
   </section>

   <section class="widget">
    <div class="widget-header"><div><h2>Reports</h2><p>Billing reports from the reporting home</p></div><a class="b-link" href={href('reports')}>Reporting Home</a></div>
    <ul class="widget-body reports">{#each codes.reports as r}<li><a href={href('reports/'+slug(r))}><Icon name="chart" size={14}/>{r}</a></li>{/each}</ul>
   </section>
  </div>

  <aside class="rail" aria-label="Notifications">
   <div class="rail-title"><span>Notifications</span><button class="b-link" aria-expanded={allNotes} onclick={()=>allNotes=!allNotes}>{allNotes?'Show fewer':'View all'}</button></div>
   {#each notes as n}<article class="note"><div class="note-top"><b><Icon name="bell" size={12}/>{n.type||'Notification'}</b><time>{n.received}</time></div><p>{n.message}</p></article>
   {:else}<p class="b-note">No notifications.</p>{/each}
  </aside>
 </div>
</section>

<style>
 .home{display:grid;gap:16px;min-width:0}
 .head{display:grid;gap:4px}.eyebrow{font-size:10px;font-weight:600;letter-spacing:.8px;text-transform:uppercase;color:var(--text-muted)}h1{font-size:24px;font-weight:700;margin:0}
 .b-kpi{cursor:pointer}.b-kpi:hover{border-color:var(--accent)}.b-kpi.selected{background:var(--accent-soft);border-color:var(--accent)}
 .split{display:flex;gap:20px;align-items:flex-start}.main{flex:7 1 0;min-width:0;display:grid;gap:16px}
 .rail{flex:3 1 0;min-width:260px;display:grid;gap:10px}
 .rail-title{display:flex;justify-content:space-between;align-items:center;font-size:14px;font-weight:600;padding:2px 2px 0}
 .widget{background:var(--surface);border:1px solid var(--border);border-radius:12px;box-shadow:var(--shadow);overflow:hidden;min-width:0}
 .widget-header{display:flex;justify-content:space-between;align-items:center;flex-wrap:wrap;gap:12px;padding:14px 16px;border-bottom:1px solid var(--border)}
 .widget-header h2{font-size:16px;font-weight:700;margin:0}.widget-header p{font-size:13px;color:var(--text-muted);margin:2px 0 0}
 .widget-header .b-search{width:220px}.widget-header select{min-height:34px;font-size:12px;padding:4px 8px}
 .widget-body{padding:16px;display:grid;gap:12px;min-width:0}
 .bw-tabs button{display:inline-flex;align-items:center;gap:6px;border:1px solid var(--border);border-radius:6px;background:var(--surface);color:var(--text);font:500 13px var(--font);padding:6px 12px;cursor:pointer;white-space:nowrap}
 .bw-tabs button.on{background:var(--accent);border-color:var(--accent);color:var(--accent-text);font-weight:600}
 .bw-tabs button span{font-size:11px;opacity:.8}.bw-tabs{justify-self:start}
 .actions{grid-template-columns:repeat(auto-fill,minmax(240px,1fr))}
 .action{display:flex;align-items:center;gap:10px;padding:12px;border:1px solid var(--border);border-radius:8px;color:var(--text);text-decoration:none;font-weight:500;font-size:13px}
 .action:hover{border-color:var(--accent);color:var(--accent)}.action .go{margin-left:auto;color:var(--accent);font-size:18px;line-height:1}
 .reports{list-style:none;margin:0;grid-template-columns:repeat(auto-fill,minmax(240px,1fr));gap:4px 16px}
 .reports a{display:flex;align-items:center;gap:8px;padding:8px 4px;text-decoration:none;font-size:13px;font-weight:500;border-bottom:1px solid var(--border)}.reports a:hover{text-decoration:underline}
 .note{background:color-mix(in srgb,var(--experience-bg) 6%,var(--surface));border:1px solid var(--border);border-radius:8px;box-shadow:0 1px 3px rgb(0 0 0 / .04);padding:12px 14px;display:grid;gap:6px}
 .note-top{display:flex;justify-content:space-between;gap:8px;align-items:center}.note-top b{display:inline-flex;align-items:center;gap:6px;font-size:13px}.note-top time{font-size:11px;color:var(--text-faint);white-space:nowrap}
 .note p{font-size:13px;line-height:1.45;margin:0;color:var(--text-muted)}
 .sr-only{position:absolute;width:1px;height:1px;overflow:hidden;clip-path:inset(50%);white-space:nowrap}
 @media(max-width:1100px){.split{flex-direction:column}.rail{width:100%;min-width:0}}
 @media(max-width:600px){.widget-header .b-search{width:auto;flex:1 1 180px}.widget-header form{width:100%}}
</style>
