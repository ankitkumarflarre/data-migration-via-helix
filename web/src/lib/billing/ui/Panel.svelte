<script lang="ts">
 import type {Snippet} from 'svelte';
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import {iconFor} from './icons';
 let {title,open=true,icon,children}:{title:string;open?:boolean;icon?:string;children:Snippet}=$props();
 // svelte-ignore state_referenced_locally
 let shown=$state(open);
 // svelte-ignore state_referenced_locally
 const id='p-'+title.toLowerCase().replace(/\W+/g,'-');
</script>
<section class="b-panel"><button type="button" aria-expanded={shown} aria-controls={id} onclick={()=>shown=!shown}><span class="b-ico soft"><Icon name={icon??iconFor(title)} size={14}/></span>{title}<span class="chev" aria-hidden="true">{shown?'−':'+'}</span></button>{#if shown}<div class="b-panel-body" id={id}>{@render children()}</div>{/if}</section>
