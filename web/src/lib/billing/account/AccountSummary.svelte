<script lang="ts">
 import Card from '../ui/Card.svelte';
 import Status from '../ui/Status.svelte';
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import type {AccountView} from '../api';
 import {href,acct} from '../routes';
 import {usd,us,stamp} from '../format';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 import Pager from '../ui/Pager.svelte';
 import ActivityText from './ActivityText.svelte';
 let {view,reload}:{view:AccountView;reload:()=>void}=$props();
 let page=$state(0);
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 const shown=$derived(view.activities.slice(page*10,page*10+10));
 const groups=$derived([...new Set(view.terms.map(t=>t.reference))]);
 const termLinks=$derived(view.terms.map(t=>({id:t.id,policy:t.policy})));
</script>
<Page title="Account" note cards>
 {#snippet actions()}<button class="btn sm ghost" aria-label="Refresh" onclick={reload}>↻</button>{/snippet}
 {#each view.holds as h (h.id)}<Banner kind="warn">{h.banner} <a href={href(h.term_id?acct(n,'term/'+h.term_id+'/hold'):acct(n,'holds/'+h.id))}>Update</a></Banner>{/each}
 <div class="b-kpis">
  {#each [{label:'Outstanding Balance',value:view.ledger.outstanding,note:view.terms.length+' policy term'+(view.terms.length===1?'':'s'),icon:'dollar',tone:'soft'},{label:'Current Amount Due',value:view.ledger.current_due,note:view.next_invoice?'next invoice '+us(view.next_invoice.invoice_date):'nothing scheduled',icon:'calendar',tone:'success'},{label:'Past-Due Amount',value:view.ledger.past_due,note:view.ledger.past_due?'needs attention':'nothing past due',icon:'clock',tone:view.ledger.past_due?'danger':'warning'},{label:'In Suspense',value:view.ledger.in_suspense,note:'pending disbursements '+usd(view.ledger.pending_disbursements),icon:'shield',tone:'warning'}] as k}
   <div class="b-kpi" class:danger={k.tone==='danger'}><span class="b-ico {k.tone}"><Icon name={k.icon} size={14}/></span><span class="b-kpi-body"><span class="b-kpi-top"><span class="b-kpi-label">{k.label}</span><strong>{usd(k.value)}</strong></span><small>{k.note}</small></span></div>
  {/each}
 </div>
 <div class="b-tiles">
  <div class="b-tile"><h3>Most Recent Invoice</h3><table><thead><tr><th>Invoice Date</th><th>Docs</th><th>Amount</th><th>Due Date</th></tr></thead><tbody><tr>{#if view.last_invoice}<td>{us(view.last_invoice.invoice_date)}</td><td title="Invoice documents are not in the walkthrough">—</td><td>{usd(view.last_invoice.amount)}</td><td>{us(view.last_invoice.due_date)}</td>{:else}<td colspan="4">&nbsp;</td>{/if}</tr></tbody></table></div>
  <div class="b-tile"><h3>Last Payment Received</h3><table><thead><tr><th>Date</th><th>Amount</th><th>Method</th></tr></thead><tbody><tr>{#if view.last_payment}<td>{us(view.last_payment.date)}</td><td><a href={href('payment/'+view.last_payment.id)}>{usd(view.last_payment.amount)}</a></td><td>{view.last_payment.method==='Credit/Debit Card'?'Card':view.last_payment.method}</td>{:else}<td colspan="3">&nbsp;</td>{/if}</tr></tbody></table></div>
  <div class="b-tile"><h3>Next Scheduled Invoice</h3><table><thead><tr><th>Invoice Date</th><th>Amount</th><th>Due Date</th></tr></thead><tbody><tr>{#if view.next_invoice}<td>{us(view.next_invoice.invoice_date)}</td><td>{usd(view.next_invoice.amount)}</td><td>{us(view.next_invoice.due_date)}</td>{:else}<td colspan="3">&nbsp;</td>{/if}</tr></tbody></table></div>
 </div>
 <Card title="Recent Account Activity"><div class="b-grid"><table><thead><tr><th>Date</th><th>Activity Type</th><th>Created By</th><th class="num">Amount</th></tr></thead><tbody>
  {#each shown as a (a.id)}<tr class="has-desc"><td>{stamp(a.at)}</td><td>{a.type}</td><td>{a.created_by}</td><td class="num">{a.amount===null?'N/A':usd(a.amount)}</td></tr><tr class="desc"><td colspan="4"><ActivityText {a} terms={termLinks}/></td></tr>
  {:else}<tr><td colspan="4" class="empty">No account activity.</td></tr>{/each}
 </tbody><Pager total={view.activities.length} bind:page cols={4}/></table></div></Card>
 <Card title="Policies / Arrangements Associated With This Account"><div class="b-grid"><table><thead><tr><th>Reference</th><th>Term</th><th>Status</th><th class="num">Current Due</th><th class="num">Outstanding Balance</th></tr></thead><tbody>
  {#each groups as g}<tr class="group"><td colspan="5">⊟ Reference: {g}</td></tr>
   {#each view.terms.filter(t=>t.reference===g) as t (t.id)}<tr><td>{t.reference}</td><td><a href={href(acct(n,'term/'+t.id))}>{us(t.effective)} - {us(t.expiration)}</a></td><td><Status value={t.status}/>{#if t.on_hold} <Status value="On Hold"/>{/if}</td><td class="num">{usd(t.current_due)}</td><td class="num">{usd(t.outstanding)}</td></tr>{/each}
  {:else}<tr><td colspan="5" class="empty">No policies are billed on this account.</td></tr>{/each}
 </tbody></table></div></Card>
</Page>
