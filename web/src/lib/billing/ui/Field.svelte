<script lang="ts">
 import type {Snippet} from 'svelte';
 let {label,id,error='',required=false,children}:{label:string;id:string;error?:string;required?:boolean;children:Snippet}=$props();
</script>
<div class="b-field" class:invalid={!!error}><label for={id} class:b-star={required}>{label}</label>{@render children()}{#if error}<span class="err" id={id+'-error'}>{error}</span>{/if}</div>
