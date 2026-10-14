<script lang="ts">
 import type {Snippet} from 'svelte';
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import {iconFor} from './icons';
 // Dashboard widget: title + subtitle header, tools on the right, padded body.
 let {title='',subtitle='',icon,children,tools}:{title?:string;subtitle?:string;icon?:string;children:Snippet;tools?:Snippet}=$props();
</script>
<section class="b-card" aria-label={title||undefined}>
 {#if title}<header><div class="b-card-title"><span class="b-ico soft"><Icon name={icon??iconFor(title)} size={14}/></span><div><h2>{title}</h2>{#if subtitle}<p>{subtitle}</p>{/if}</div></div>{#if tools}<div class="b-card-tools">{@render tools()}</div>{/if}</header>{/if}
 <div class="b-card-body">{@render children()}</div>
</section>
