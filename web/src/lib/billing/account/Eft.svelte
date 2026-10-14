<script lang="ts">
 import Card from '../ui/Card.svelte';
 import Page from '../ui/Page.svelte';
</script>
<Page title="Scheduled EFT Transactions" note cards>
 <Card title="EFT Transactions" subtitle="EFT enrolment (bank details, recurring card payments) is not shown in the walkthrough, so no EFT transactions can be scheduled yet." icon="calendar">
  <div class="b-grid"><table><thead><tr><th>Date</th><th>Activity Type</th><th>Created By</th><th class="num">Amount</th></tr></thead><tbody><tr><td colspan="4" class="empty">No results to display.</td></tr></tbody></table></div>
 </Card>
</Page>