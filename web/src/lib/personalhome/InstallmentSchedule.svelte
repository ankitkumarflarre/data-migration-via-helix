<script lang="ts" module>
 export interface ScheduleLine {recv:string;trans:string;amount:number;cash:number;credit:number;writeOff:number;redist:number;balance:number}
 export interface ScheduleInstallment {key:string;install:string;due:string;term:string;items:ScheduleLine[]}
</script>
<script lang="ts">
 import type {Snippet} from 'svelte';
 import Icon from './WorkspaceIcon.svelte';
 // Without `installments` the quote's sample schedule is drawn from the plan
 // and due date; the billing workspace passes its real installments (amounts
 // in dollars) and the system date.
 let {plan='',due='',policyNumber='',effective='',expiration='',id,installments:given,systemDate,footnote,tools,collapsed=false}:{plan?:string;due?:string;policyNumber?:string;effective?:string;expiration?:string;id:string;installments?:ScheduleInstallment[];systemDate?:string;footnote?:string;tools?:Snippet;collapsed?:boolean}=$props();
 const months:Record<string,number>={Annual:12,'Semi-Annual':6,Quarterly:3,Monthly:1};
 // Sample figures modelled on the billing system's schedule: fees are charged
 // with the first installment and the premium is split evenly across the plan.
 const SAMPLE_PREMIUM=2262, SAMPLE_FEES=[25,2,23,226];
 const DAYS_BEFORE_DUE=30;
 // Rows start open for the quote and closed in billing (#062).
 let toggled=$state<string[]>([]);
 const isOpen=(key:string)=>collapsed===toggled.includes(key);
 const iso=(d:Date)=>d.toISOString().slice(0,10);
 // Until a plan and due date are chosen, show an annual schedule from the effective date.
 const assumed=$derived(!given&&(!months[plan]||!/^\d{4}-\d{2}-\d{2}$/.test(due)));
 const usedPlan=$derived(months[plan]?plan:'Annual');
 const usedDue=$derived(/^\d{4}-\d{2}-\d{2}$/.test(due)?due:effective);
 const term=$derived(`${policyNumber} ${us(effective)} - ${us(expiration)}`);
 const installments=$derived.by(():ScheduleInstallment[]=>{
  if(given)return given;
  const step=months[usedPlan], m=usedDue.match(/^(\d{4})-(\d{2})-(\d{2})$/);
  if(!m)return [];
  const [y,mo,d]=[+m[1],+m[2]-1,+m[3]], count=12/step;
  const share=Math.floor(SAMPLE_PREMIUM*100/count)/100, last=Math.round((SAMPLE_PREMIUM-share*(count-1))*100)/100;
  return Array.from({length:count},(_,i)=>{
   const end=new Date(Date.UTC(y,mo+i*step+1,0)).getUTCDate(), date=new Date(Date.UTC(y,mo+i*step,Math.min(d,end)));
   const amounts=[...(i===0?SAMPLE_FEES.map(a=>({recv:'FEE',amount:a})):[]),{recv:'PREM',amount:i===count-1?last:share}];
   return {key:iso(date),due:iso(date),install:iso(new Date(date.getTime()-DAYS_BEFORE_DUE*864e5)),term,items:amounts.map(x=>({recv:x.recv,trans:'NBUS',amount:x.amount,cash:0,credit:0,writeOff:0,redist:0,balance:x.amount}))};
  });
 });
 function us(s:string){const m=s.match(/^(\d{4})-(\d{2})-(\d{2})$/);return m?`${+m[2]}/${+m[3]}/${m[1]}`:'—';}
 const usd=(n:number)=>n.toLocaleString('en-US',{style:'currency',currency:'USD'});
 const sum=(items:ScheduleLine[],k:keyof Omit<ScheduleLine,'recv'|'trans'>)=>items.reduce((t,x)=>t+x[k],0);
 const shownDate=$derived((systemDate?new Date(systemDate+'T12:00:00'):new Date()).toLocaleDateString('en-US',{year:'numeric',month:'long',day:'numeric'}));
 const toggle=(key:string)=>toggled=toggled.includes(key)?toggled.filter(k=>k!==key):[...toggled,key];
</script>

{#snippet amounts(items:ScheduleLine[])}<td class="num">{usd(sum(items,'amount'))}</td><td class="num">{usd(sum(items,'cash'))}</td><td class="num">{usd(sum(items,'credit'))}</td><td class="num">{usd(sum(items,'writeOff'))}</td><td class="num">{usd(sum(items,'redist'))}</td><td class="num">{usd(sum(items,'balance'))}</td>{/snippet}

<section class="installments" {id} aria-label="Installment schedule">
 <header><h4>Installment Schedule</h4>{@render tools?.()}</header>
 <p class="system">The current system date is {shownDate}{assumed?' · Showing an annual schedule from the effective date until a payment plan and invoice due date are chosen.':''}</p>
 <div class="table-scroll">
  <table>
   <thead><tr><th>Install Date</th><th>Term</th><th>Due Date</th><th>Recv</th><th>Trans</th><th class="num">Item Amt</th><th class="num">Install Amt</th><th class="num">Cash</th><th class="num">Credit</th><th class="num">Write-Off</th><th class="num">ReDist</th><th class="num">Balance</th></tr></thead>
   <tbody>
    {#each installments as row (row.key)}
     {@const open=isOpen(row.key)}
     {@const termOpen=isOpen(row.key+'/term')}
     <tr class="install"><td><button class="tree" aria-expanded={open} aria-label={(open?'Collapse':'Expand')+' installment due '+us(row.due)} onclick={()=>toggle(row.key)}>{open?'⊟':'⊞'}</button>{us(row.install)}</td><td></td><td><b>{us(row.due)}</b></td><td></td><td></td><td></td>{@render amounts(row.items)}</tr>
     {#if open}
      <tr class="term"><td><span class="indent"></span><button class="tree" aria-expanded={termOpen} aria-label={(termOpen?'Collapse':'Expand')+' term '+row.term} onclick={()=>toggle(row.key+'/term')}>{termOpen?'⊟':'⊞'}</button></td><td class="nowrap">{row.term}</td><td></td><td></td><td></td><td></td>{@render amounts(row.items)}</tr>
      {#if termOpen}
       {#each row.items as item}<tr class="item"><td><span class="indent deep"></span><Icon name="file" size={14}/></td><td></td><td>{us(row.due)}</td><td>{item.recv}</td><td>{item.trans}</td><td class="num">{usd(item.amount)}</td>{@render amounts([item])}</tr>{/each}
      {/if}
     {/if}
    {:else}
     <tr><td colspan="12" class="empty">{given?'No installments in this date range.':'Enter an effective date on New quote to see the schedule.'}</td></tr>
    {/each}
   </tbody>
  </table>
 </div>
 <p class="footnote">{footnote??'Due dates follow the payment plan from the invoice due date. Amounts are sample figures for display, not a rated premium; the install date is shown 30 days before each due date. No payments have been received.'}</p>
</section>

<style>
 .installments{border:1px solid var(--border);border-radius:10px;background:var(--surface);overflow:hidden;min-width:0}
 header{display:flex;align-items:center;justify-content:space-between;gap:10px;flex-wrap:wrap;padding:12px 16px;background:var(--accent-soft);border-bottom:1px solid var(--border)}h4{margin:0;font-size:14px;color:var(--accent)}
 .system{margin:0;padding:10px 16px;font-size:12px;font-style:italic;color:var(--text-muted);border-bottom:1px solid var(--border)}
 .table-scroll{overflow-x:auto;max-height:420px;overflow-y:auto}
 table{width:100%;border-collapse:collapse;font-size:12px;min-width:1100px}
 th{position:sticky;top:0;z-index:1;background:var(--accent);color:var(--accent-text);font-weight:600;text-align:left;padding:8px 10px;white-space:nowrap}
 td{padding:7px 10px;border-bottom:1px solid var(--border);white-space:nowrap;vertical-align:middle}
 .num{text-align:right}.nowrap{white-space:nowrap}
 tr.install td{background:var(--surface-2)}tr.term td{color:var(--text-muted)}tr.item :global(svg){color:var(--accent);vertical-align:middle}
 .tree{border:0;background:none;color:var(--accent);font-size:15px;line-height:1;padding:0 6px 0 0;cursor:pointer}
 .indent{display:inline-block;width:16px}.indent.deep{width:36px}
 .empty{text-align:center;color:var(--text-muted);padding:20px}
 .footnote{margin:0;padding:10px 16px;font-size:11px;color:var(--text-muted)}
</style>
