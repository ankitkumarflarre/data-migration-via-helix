<script lang="ts">
 import type {AccountView} from '../api';
 import {usd,us,parseMoney} from '../format';
 // Allocation Amt per policy plus the Account Level row; whatever is not
 // allocated to a policy stays in suspense at the account level.
 let {view,total,alloc=$bindable(),errors={}}:{view:AccountView;total:number;alloc:Record<string,string>;errors?:Record<string,string>}=$props();
 const allocated=$derived(Object.entries(alloc).filter(([k])=>k!=='').reduce((t,[,v])=>t+(parseMoney(v)??0),0));
 const remaining=$derived(total-allocated);
 const terms=$derived(view.terms.filter(t=>t.status!=='Cancelled'));
</script>
<div class="b-grid"><table><thead><tr><th>Allocation Amt</th><th>Reference</th><th>Effective Date</th><th>Expiration Date</th><th class="num">Current Due Amt</th><th class="num">Past Due Amount</th><th class="num">Outstanding Balance</th><th>Suspense State</th></tr></thead><tbody>
 {#each terms as t,i (t.id)}<tr><td><input aria-label={'Allocation amount for '+t.policy} inputmode="decimal" size="9" placeholder="0.00" bind:value={alloc[t.id]} class:invalid={!!errors['allocations.'+i]}/>{#if errors['allocations.'+i]}<div class="b-err">{errors['allocations.'+i]}</div>{/if}</td><td>{t.policy}</td><td>{us(t.effective)}</td><td>{us(t.expiration)}</td><td class="num">{usd(t.current_due)}</td><td class="num">{usd(t.past_due)}</td><td class="num">{usd(t.outstanding)}</td><td></td></tr>{/each}
 <tr><td class="num">{usd(Math.max(remaining,0))}</td><td>Account Level</td><td></td><td></td><td class="num">{usd(view.ledger.current_due)}</td><td class="num">{usd(view.ledger.past_due)}</td><td class="num">{usd(view.ledger.outstanding)}</td><td>No Hold</td></tr>
</tbody></table></div>
<div class="b-inline totals"><span class="b-badge info">Total Allocated {usd(allocated)}</span><span class="b-badge" class:bad={remaining<0}>Total Remaining {usd(remaining)}</span></div>
{#if errors.allocations}<p class="b-err" role="alert">{errors.allocations}</p>{/if}
<p class="b-note">Any remaining balance will stay on hold in suspense at the account level.</p>
