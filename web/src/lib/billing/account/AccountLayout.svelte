<script lang="ts">
 import {onMount} from 'svelte';
 import {billing,errorText,type AccountView,type TermView} from '../api';
 import {href,acct,type Match} from '../routes';
 import {usd} from '../format';
 import Panel from '../ui/Panel.svelte';
 import Banner from '../ui/Banner.svelte';
 import Placeholder from '../ui/Placeholder.svelte';
 import Status from '../ui/Status.svelte';
 import {initials} from '../format';
 import AccountSummary from './AccountSummary.svelte';
 import Schedule from './Schedule.svelte';
 import AccountSettings from './AccountSettings.svelte';
 import MakePayment from './MakePayment.svelte';
 import SuspenseAllocation from './SuspenseAllocation.svelte';
 import PaymentsList from './PaymentsList.svelte';
 import Suspense from './Suspense.svelte';
 import Disburse from './Disburse.svelte';
 import Eft from './Eft.svelte';
 import Holds from './Holds.svelte';
 import Tasks from './Tasks.svelte';
 import AccountTransactions from './AccountTransactions.svelte';
 import TermSummary from './TermSummary.svelte';
 import TermForms from './TermForms.svelte';
 let {match}:{match:Match}=$props();
 // svelte-ignore state_referenced_locally
 const number=match.params.account, termID=match.params.term??'';
 // svelte-ignore state_referenced_locally
 const comp=match.route.component, pattern=match.route.pattern;
 let view=$state<AccountView|null>(null), term=$state<TermView|null>(null), error=$state('');
 async function reload(){
  error='';
  try{const [v,t]=await Promise.all([billing.account(number),termID?billing.term(termID):Promise.resolve(null)]);view=v;term=t;}catch(e){error=errorText(e);}
 }
 onMount(()=>{reload();billing.touch(number).catch(()=>{});});
 // Duck Creek narrows the left menu on focused forms (#058, #064, #066, #085).
 const payment=['MakePayment','SuspenseAllocation','PaymentsList'].includes(comp);
 const single:Record<string,string>={'account/:account/settings':'Select Billing Account Options','account/:account/tasks/new':'New Task','account/:account/transactions':'Billing Transactions','account/:account/transactions/misc':'Add Miscellaneous Billing Item','account/:account/term/:term/plan':'Term Plan Setup','account/:account/term/:term/renewal-plan':'Renewal Plan Setup','account/:account/term/:term/pif':'Paid In Full Discount','account/:account/term/:term/internal-collections':'Send To Internal Collections'};
 const focused=single[pattern];
 const menu=[['','Account'],['schedule','Installment Schedule'],['settings','Account Settings'],['pay','Payment Processing'],['suspense','Items In Suspense'],['eft','EFT Maintenance'],['holds','Hold Activity'],['tasks','Account Tasks'],['transactions','Billing Transactions']];
 const on=(sub:string)=>sub===''?comp==='AccountSummary':sub==='pay'?payment:sub==='suspense'?comp==='Suspense'||comp==='Disburse':match.path.startsWith(acct(number,sub))&&!termID;
 const termBase=$derived(acct(number,'term/'+termID));
 const actions=$derived([['internal-collections','Send To Internal Collections'],['transfer','Transfer Policy Term'],['collections','Transfer To Collections'],['plan','View/Edit Policy Term Settings'],['renewal-plan','Change renewal plan'],['hold','Hold Billing for Policy Term'],['pif','Paid In Full Discount']]);
</script>

<div class="bw-body">
 <aside class="b-side" aria-label="Account navigation">
  <div class="b-menu">
   <header><span class="b-avatar" aria-hidden="true">{initials(view?.name??'')}</span><div><strong>{view?.name??'…'}</strong><small>{number}</small>{#if view}<span><Status value={view.account.status}/></span>{/if}</div></header>
   <nav aria-label="Account">
    {#if focused}<a class="on" href={href(match.path)} aria-current="page">{focused}</a>
    {:else if payment}{#each [['pay','Make A Payment','MakePayment'],['suspense-allocation','Suspense Allocation','SuspenseAllocation'],['payments','Payments List','PaymentsList']] as [sub,label,c]}<a href={href(acct(number,sub))} class:on={comp===c} aria-current={comp===c?'page':undefined}>{label}</a>{/each}
    {:else}{#each menu as [sub,label]}<a href={href(acct(number,sub))} class:on={on(sub)} aria-current={on(sub)?'page':undefined}>{label}</a>{/each}{/if}
   </nav>
  </div>
  {#if !focused&&!payment&&view}
   {#if termID&&term}
    <Panel title="Policy Term Actions"><div class="links">{#each actions as [sub,label]}<a href={href(termBase+'/'+sub)}>{label}</a>{/each}</div></Panel>
    <Panel title="Term Details"><dl><dt>Billing Frequency:</dt><dd>{term.term.billing_frequency}</dd><dt>Master Company:</dt><dd>{term.term.master_company}</dd><dt>Line of Business:</dt><dd>{term.term.line_of_business}</dd><dt>Equity Date:</dt><dd>{term.term.equity_date?term.term.equity_date.replace(/^(\d{4})-(\d{2})-(\d{2})$/,(_,y,m,d)=>`${+m}/${+d}/${y}`):''}</dd><dt>Currency:</dt><dd>USD</dd></dl></Panel>
    <Panel title="Term Ledger"><dl><dt>Outstanding Balance:</dt><dd>{usd(term.ledger.outstanding)}</dd><dt>Current Amount Due:</dt><dd>{usd(term.ledger.current_due)}</dd><dt>Past-Due Amount:</dt><dd>{usd(term.ledger.past_due)}</dd><dt>Receivable Amount:</dt><dd>{usd(term.ledger.receivable)}</dd><dt>Paid on Term:</dt><dd>{usd(term.ledger.paid)}</dd><dt>Written Off/Waived:</dt><dd>{usd(term.ledger.written_off)}</dd></dl></Panel>
    <Panel title="Agent">{#if term.term.agent}<div>{term.term.agent.name}<br/>{term.term.agent.address}</div>{:else}<div class="b-note">No agent.</div>{/if}</Panel>
   {:else}
    {#if comp==='Schedule'}<Panel title="Schedule History" open={false}>{#if view.schedule_history.length}{#each view.schedule_history as h}<div>{h.date.replace(/^(\d{4})-(\d{2})-(\d{2})$/,(_,y,m,d)=>`${+m}/${+d}/${y}`)}: {h.text}</div>{/each}{:else}<div class="b-note">No schedule changes.</div>{/if}</Panel>{/if}
    <Panel title="Account Ledger"><dl><dt>Outstanding Balance:</dt><dd>{usd(view.ledger.outstanding)}</dd><dt>Current Amount Due:</dt><dd>{usd(view.ledger.current_due)}</dd><dt>Past-Due Amount:</dt><dd>{usd(view.ledger.past_due)}</dd><dt>In Suspense Amount:</dt><dd>{usd(view.ledger.in_suspense)}</dd><dt>Pending Disbursements:</dt><dd>{usd(view.ledger.pending_disbursements)}</dd></dl></Panel>
    <Panel title="Account Details"><dl><dt>Target Due Days:</dt><dd>{view.account.target_due_days}</dd><dt>Account Type:</dt><dd>{view.account.account_type}</dd><dt>Account Class:</dt><dd>{view.account.account_class}</dd><dt>Collection Method:</dt><dd>{view.account.collection_method}</dd><dt>Disbursement Hold:</dt><dd>{view.account.disbursement_hold}</dd><dt>NSF Counter:</dt><dd>{view.account.nsf_counter}</dd><dt>Currency:</dt><dd>{view.account.currency}</dd></dl></Panel>
    {#each [['Payee',view.payee,'change-payee'],['Payor',view.payor,'change-payor']] as const as [label,p,change]}
     <Panel title={label}>{#if p}<a href={href('party/'+p.id)}>{p.name}</a><div>{p.line1}<br/>{p.line2}<br/>{p.email}</div>{/if}<a href={href(`parties/search?for=${change}&account=${encodeURIComponent(number)}`)}>Change {label}</a></Panel>
    {/each}
   {/if}
  {/if}
 </aside>
 <div class="main">
  {#if error}<Banner kind="error">{error}</Banner>{/if}
  {#if view&&(!termID||term)}
   {#if comp==='AccountSummary'}<AccountSummary {view} {reload}/>
   {:else if comp==='Schedule'}<Schedule {view}/>
   {:else if comp==='AccountSettings'}<AccountSettings {view}/>
   {:else if comp==='MakePayment'}<MakePayment {view} {reload}/>
   {:else if comp==='SuspenseAllocation'}<SuspenseAllocation {view} {reload}/>
   {:else if comp==='PaymentsList'}<PaymentsList {view}/>
   {:else if comp==='Suspense'}<Suspense {view} {reload}/>
   {:else if comp==='Disburse'}<Disburse {view} query={match.query}/>
   {:else if comp==='Eft'}<Eft/>
   {:else if comp==='Holds'}<Holds {view} {term} holdID={match.params.hold??''}/>
   {:else if comp==='Tasks'}<Tasks {view} {reload} adding={pattern.endsWith('/new')}/>
   {:else if comp==='AccountTransactions'}<AccountTransactions {view}/>
   {:else if comp==='TermSummary'&&term}<TermSummary {view} {term}/>
   {:else if comp==='TermForms'&&term}<TermForms {view} {term} kind={match.path.split('/').pop()??''}/>
   {:else}<Placeholder title={match.route.title} back={href(termID?termBase:acct(number))}/>{/if}
  {/if}
 </div>
</div>
<style>.main{display:grid;gap:16px;min-width:0}</style>
