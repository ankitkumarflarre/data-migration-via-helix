<script lang="ts">
 import type {Snippet} from 'svelte';
 let {kind='info',children}:{kind?:'info'|'warn'|'error';children:Snippet}=$props();
</script>
<div class="b-banner {kind}" role={kind==='info'?'status':'alert'}><span class="ico" aria-hidden="true">{kind==='info'?'i':'!'}</span><div>{@render children()}</div></div>
