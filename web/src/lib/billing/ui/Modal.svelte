<script lang="ts">
 import type {Snippet} from 'svelte';
 import {onMount} from 'svelte';
 let {title,onclose,children}:{title:string;onclose:()=>void;children:Snippet}=$props();
 let box:HTMLElement;
 onMount(()=>{const prev=document.activeElement as HTMLElement|null;box?.querySelector<HTMLElement>('input,select,textarea,button:not(.close)')?.focus();const key=(e:KeyboardEvent)=>{if(e.key==='Escape')onclose();};window.addEventListener('keydown',key);return()=>{window.removeEventListener('keydown',key);prev?.focus();};});
</script>
<div class="b-modal-back" role="presentation" onclick={e=>{if(e.target===e.currentTarget)onclose();}}>
 <div class="b-modal" role="dialog" aria-modal="true" aria-label={title} bind:this={box}><header><h2>{title}</h2><button class="btn sm ghost close" aria-label="Close" onclick={onclose}>✕</button></header><div class="b-modal-body">{@render children()}</div></div>
</div>
