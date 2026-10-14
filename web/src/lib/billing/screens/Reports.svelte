<script lang="ts">
 import Card from '../ui/Card.svelte';
 import {onMount} from 'svelte';
 import {billing,errorText,type Report} from '../api';
 import {go} from '../routes';
 import {codes} from '../codes';
 import {csv} from '../format';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let {id}:{id:string}=$props();
 const slug=(name:string)=>name.toLowerCase().replace(/ /g,'-');
 let report=$state<Report|null>(null), error=$state(''), format=$state(''), history=$state<string[]>(readHistory());
 function readHistory():string[]{try{const v=JSON.parse(sessionStorage.getItem('billingReportHistory')||'[]');return Array.isArray(v)?v:[];}catch{return [];}}
 onMount(async()=>{
  if(!id)return;
  try{report=await billing.report(id);history=[id,...history.filter(h=>h!==id)].slice(0,8);try{sessionStorage.setItem('billingReportHistory',JSON.stringify(history));}catch{/* storage unavailable */}}catch(e){error=errorText(e);}
 });
 const nameOf=(s:string)=>codes.reports.find(r=>slug(r)===s)??s;
 function exportAs(f:string){
  format='';
  if(!report)return;
  if(f==='PDF'){window.print();return;}
  const rows:string[][]=[[report.title],[report.date],...report.meta.map(m=>[m.name,m.value])];
  for(const s of report.sections){rows.push([]);if(s.heading)rows.push([s.heading,s.heading_right??'']);rows.push(s.columns);rows.push(...s.rows);}
  const url=URL.createObjectURL(new Blob([csv(rows)],{type:'text/csv'}));
  const a=document.createElement('a');a.href=url;a.download=report.id+'.csv';a.click();setTimeout(()=>URL.revokeObjectURL(url),1000);
 }
</script>
<Page title="Reporting Home" subtitle="Run a billing report, revisit one from history, or export it." date={false} cards>
 <Card title="Choose a report" icon="chart">
  <div class="b-form no-print">
   <div class="b-field"><label for="rh-rep">Reports</label><select id="rh-rep" value={id} onchange={e=>go('reports/'+e.currentTarget.value)}><option value=""></option>{#each codes.reports as r}<option value={slug(r)}>{r}</option>{/each}</select></div>
   <div class="b-field"><label for="rh-hist">Report History</label><select id="rh-hist" value="" onchange={e=>{if(e.currentTarget.value)go('reports/'+e.currentTarget.value);}}><option value=""></option>{#each history as h}<option value={h}>{nameOf(h)}</option>{/each}</select></div>
   <div class="b-field"><label for="rh-export">Export</label><select id="rh-export" bind:value={format} disabled={!report} onchange={()=>exportAs(format)}><option value=""></option>{#each codes.reportExportFormats as f}<option value={f} disabled={!codes.reportExportEnabled.includes(f)}>{f}{codes.reportExportEnabled.includes(f)?'':' (not available)'}</option>{/each}</select></div>
  </div>
  {#if !id}<p class="b-note">Choose a report from the Reports list.</p>{/if}
 </Card>
 {#if error}<Banner kind="error">{error}</Banner>{/if}
 {#if report}
  <section class="b-card"><div class="b-card-body">
  <article class="report">
   <h3>{report.title}</h3><p class="center">{report.date}</p>
   <dl>{#each report.meta as m}<dt>{m.name}</dt><dd>{m.value}</dd>{/each}</dl>
   {#each report.sections as s}
    <div class="b-grid"><table>
     {#if s.heading}<caption><div class="cap"><span>{s.heading}</span><span class="right">{s.heading_right}</span></div></caption>{/if}
     <thead><tr>{#each s.columns as c,i}<th class:num={s.numeric[i]}>{c}</th>{/each}</tr></thead>
     <tbody>{#each s.rows as r,ri}<tr class={s.kinds[ri]}>{#each r as cell,i}<td class:num={s.numeric[i]}>{cell}</td>{/each}</tr>{:else}<tr><td colspan={s.columns.length} class="empty"><em>{s.empty}</em></td></tr>{/each}</tbody>
    </table></div>
   {/each}
   <p class="center"><strong>*** End of Report ***</strong></p>
   <p class="foot"><span>{report.title}</span><span>{report.date}</span><span>Page 1 of 1</span></p>
  </article>
  </div></section>
 {/if}
</Page>
<style>
 .report{display:grid;gap:12px;max-width:1000px;margin:0 auto;width:100%}.report h3{text-align:center;font-size:20px;font-weight:400;margin:0}.center{text-align:center;margin:0}
 dl{display:grid;grid-template-columns:140px 1fr;gap:6px 12px;margin:0;font-size:13px}dd{margin:0}
 caption .cap{display:flex;justify-content:space-between;gap:12px}.right{font-weight:700}.foot{display:flex;justify-content:space-between;font-size:12px;margin:0}
</style>
