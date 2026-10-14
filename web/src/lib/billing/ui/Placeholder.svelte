<script lang="ts">
 import Page from './Page.svelte';
 let {title,back}:{title:string;back?:string}=$props();
</script>
<Page {title}>
 <div class="b-placeholder"><strong>Not yet specified.</strong> The Duck Creek walkthrough does not show this screen, so its fields and rules are unknown. It is listed as a gap to capture in a follow-up walkthrough (spec §8).</div>
 {#if back}<div class="b-actions"><a class="btn" href={back}>Back</a></div>{/if}
</Page>
