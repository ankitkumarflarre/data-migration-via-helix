<script lang="ts">
 import type {Snippet} from 'svelte';
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import {iconFor} from './icons';
 let {title,open=true,soft=false,icon,children}:{title:string;open?:boolean;soft?:boolean;icon?:string;children:Snippet}=$props();
 // svelte-ignore state_referenced_locally
 let shown=$state(open);
 const id='s-'+Math.random().toString(36).slice(2,8);
</script>
<section class="b-section" class:soft><h3><button type="button" aria-expanded={shown} aria-controls={id} onclick={()=>shown=!shown}><span class="b-ico"><Icon name={icon??iconFor(title)} size={14}/></span>{title}<span class="chev" aria-hidden="true">{shown?'−':'+'}</span></button></h3>{#if shown}<div class="b-section-body" id={id}>{@render children()}</div>{/if}</section>
