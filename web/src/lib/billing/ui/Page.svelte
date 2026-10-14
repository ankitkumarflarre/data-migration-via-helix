<script lang="ts">
 import type {Snippet} from 'svelte';
 import {longDate} from '../format';
 import {ws,takeFlash} from '../state.svelte';
 import Banner from './Banner.svelte';
 // A page is a dashboard-style header on the background followed by cards.
 // With cards=false the content goes into one card; with cards it is laid out
 // as given (the screen uses <Card> itself).
 let {title,subtitle='',date=true,required=false,note=false,cards=false,children,actions}:{title:string;subtitle?:string;date?:boolean;required?:boolean;note?:boolean;cards?:boolean;children:Snippet;actions?:Snippet}=$props();
 // The banner left by the previous screen is shown once, on this page.
 const flash=takeFlash();
 // The eyebrow names the area the page belongs to, like the quote screens.
 const parts=location.hash.replace(/^#billing\/?/,'').split(/[/?]/);
 const eyebrow=parts[0]==='account'&&parts[1]?'Billing account · '+decodeURIComponent(parts[1]):parts[0]==='party'?'Billing · People & Places':parts[0]==='admin'?'Billing · Administration':parts[0]==='batches'?'Billing · Batch cash management':'Billing';
</script>
<section class="b-stack" aria-label={title}>
 <header class="b-pagehead">
  <div class="b-title"><span class="eyebrow">{eyebrow}</span><h1>{title}</h1>
   {#if subtitle}<p class="b-date">{subtitle}</p>{/if}
   {#if date && ws.system}<p class="b-date">The current system date is {longDate(ws.system.date)}</p>{/if}
   {#if required}<p class="b-req">indicates a required field.</p>{/if}
  </div>
  {#if actions||note}<div class="b-tools">{@render actions?.()}{#if note}<button class="btn sm" disabled title="Notes are not specified in the walkthrough">Add A Note</button>{/if}</div>{/if}
 </header>
 {#if flash}<Banner kind={flash.kind}>{flash.text}{#if flash.link}<a href={flash.link.href}>{flash.link.label}</a>{flash.link.after??''}{/if}</Banner>{/if}
 {#if cards}{@render children()}{:else}<div class="b-card"><div class="b-card-body">{@render children()}</div></div>{/if}
</section>
