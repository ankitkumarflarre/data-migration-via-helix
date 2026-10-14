<script lang="ts">
 import Status from '../ui/Status.svelte';
 import type {Payment} from '../api';
 import {usd,us} from '../format';
 let {payment:p}:{payment:Payment}=$props();
 const card=$derived(p.method==='Credit/Debit Card');
</script>
<div class="b-grid"><table><thead><tr><th>Amount</th><th>Date Entered</th><th>Status</th><th>Payment Method</th>{#if card}<th>Card Number</th><th>Name on Card</th>{:else}<th>Check #</th><th>Name on Check</th><th>Postmark Date</th>{/if}</tr></thead>
 <tbody><tr><td>{usd(p.amount)}</td><td>{us(p.entry_date)}</td><td><Status value={p.status}/></td><td>{p.method}</td>{#if card}<td>{p.card_last4?'***'+p.card_last4.slice(-1):''}</td><td>{p.name_on_card??''}</td>{:else}<td>{p.check_number??''}</td><td>{p.name_on_check??''}</td><td>{us(p.postmark_date)}</td>{/if}</tr></tbody></table></div>
