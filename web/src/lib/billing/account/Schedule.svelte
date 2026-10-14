<script lang="ts">
 import type {AccountView} from '../api';
 import InstallmentSchedule from '../../personalhome/InstallmentSchedule.svelte';
 let {view}:{view:AccountView}=$props();
 // svelte-ignore state_referenced_locally
 let from=$state(view.terms[0]?.effective??''), to=$state(''), applied=$state({from:view.terms[0]?.effective??'',to:''});
 const dollars=(c:number)=>c/100;
 const rows=$derived(view.schedule.filter(r=>(!applied.from||r.due_date>=applied.from)&&(!applied.to||r.due_date<=applied.to)).map(r=>({key:r.id,install:r.install_date,due:r.due_date,term:r.term,items:r.items.map(i=>({recv:i.recv,trans:i.trans,amount:dollars(i.amount),cash:dollars(i.cash),credit:dollars(i.credit),writeOff:dollars(i.write_off),redist:dollars(i.redist),balance:dollars(i.balance)}))})));
</script>
<InstallmentSchedule id="billing-schedule" installments={rows} systemDate={view.system_date} collapsed footnote="Invoices are generated on the install date, 20 days before the due date. Schedule changes appear under Schedule History.">
 {#snippet tools()}<form class="b-inline" onsubmit={e=>{e.preventDefault();applied={from,to};}}><label for="sch-from">From</label><input id="sch-from" type="date" bind:value={from}/><label for="sch-to">To</label><input id="sch-to" type="date" bind:value={to}/><button class="btn sm" type="submit" aria-label="Refresh schedule">↻</button></form>{/snippet}
</InstallmentSchedule>
