<script lang="ts">
 import Card from '../ui/Card.svelte';
 import Status from '../ui/Status.svelte';
 import type {AccountView,TermView} from '../api';
 import {href,acct} from '../routes';
 import {usd,us} from '../format';
 import Page from '../ui/Page.svelte';
 import Banner from '../ui/Banner.svelte';
 let {view,term:t}:{view:AccountView;term:TermView}=$props();
 // svelte-ignore state_referenced_locally
 const n=view.account.number;
 let open=$state(true);
</script>
<Page title="Policy Term Summary" note cards>
 {#if t.hold}<Banner kind="warn">{t.hold.banner} <a href={href(acct(n,'term/'+t.term.id+'/hold'))}>Update</a></Banner>{/if}
 <Card title="Summary"><div class="b-grid"><table><thead><tr><th>Policy Number</th><th>Eff/Exp Dates</th><th>Status</th><th>Aggregation Reference</th></tr></thead><tbody><tr><td>{t.term.policy_number}</td><td>{us(t.term.effective)} - {us(t.term.expiration)}</td><td><Status value={t.term.status}/></td><td>{t.term.aggregation_reference}</td></tr></tbody></table></div></Card>
 <div class="b-tiles">
  <div class="b-tile"><h3>Most Recent Invoice</h3><table><thead><tr><th>Invoice Date</th><th>Docs</th><th>Amount</th><th>Due Date</th></tr></thead><tbody><tr>{#if t.last_invoice}<td>{us(t.last_invoice.invoice_date)}</td><td>—</td><td>{usd(t.last_invoice.amount)}</td><td>{us(t.last_invoice.due_date)}</td>{:else}<td colspan="4">&nbsp;</td>{/if}</tr></tbody></table></div>
  <div class="b-tile"><h3>Next Installment</h3><table><thead><tr><th>Date</th><th>Amount</th><th>Due Date</th></tr></thead><tbody><tr>{#if t.next_installment}<td><a href={href(acct(n,'schedule'))}>{us(t.next_installment.invoice_date)}</a></td><td>{usd(t.next_installment.amount)}</td><td>{us(t.next_installment.due_date)}</td>{:else}<td colspan="3">&nbsp;</td>{/if}</tr></tbody></table></div>
 </div>
 <Card title="Additional Data"><div class="b-grid"><table><thead><tr><th>Name</th><th>Value</th></tr></thead><tbody>{#each t.term.additional_data??[] as d}<tr><td>{d.name}</td><td class="wrap">{d.value}</td></tr>{:else}<tr><td colspan="2" class="empty">&nbsp;</td></tr>{/each}</tbody></table></div></Card>
 <Card title="Policy Term Activity"><div class="b-grid"><table><thead><tr><th>Date</th><th>Transaction</th><th>Receivable</th><th>Receivable Sub Type</th><th class="num">Amount</th><th>Bill In Full</th></tr></thead><tbody>
  {#each t.term.receivables as r (r.id)}<tr class="has-desc"><td>{us(r.date)}</td><td>{r.transaction}</td><td>{r.receivable}</td><td>{r.sub_type}</td><td class="num">{usd(r.amount)}</td><td>{r.bill_in_full?'Yes':''}</td></tr><tr class="desc"><td colspan="6">Billing Item Effective Dates - {us(r.effective)} - {us(r.expiration)}</td></tr>{/each}
 </tbody></table></div></Card>
 <Card title="Other Terms Associated With This Policy"><div class="b-grid"><table><thead><tr><th>Term</th><th>Status</th><th class="num">Current Due</th><th class="num">Outstanding Balance</th><th class="num">Potential Commission</th></tr></thead><tbody>
  <tr class="group"><td colspan="5"><button class="b-link" aria-expanded={open} onclick={()=>open=!open}>{open?'⊟':'⊞'}</button> Term: {us(t.term.effective)} - {us(t.term.expiration)}</td></tr>
  {#if open}{#each t.other_terms as o (o.id)}<tr><td><a href={href(acct(n,'term/'+o.id))}>{us(o.effective)} - {us(o.expiration)}</a></td><td><Status value={o.status}/></td><td class="num">{usd(o.current_due)}</td><td class="num">{usd(o.outstanding)}</td><td class="num">{o.id===t.term.id?usd(t.potential_commission):''}</td></tr>{/each}{/if}
 </tbody></table></div></Card>
</Page>
