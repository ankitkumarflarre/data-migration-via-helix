<script lang="ts">
 import type {Identity} from '../api';
 let {value=$bindable(),errors={},prefix,labels='entry'}:{value:Identity;errors?:Record<string,string>;prefix:string;labels?:'entry'|'detail'}=$props();
 // Each identifier carries a required marker, but the rule is "at least one".
 const rows=$derived(labels==='entry'
  ?[['last','Last Name',true],['account_ref','Account Reference',true],['first','First Name',false],['policy_ref','Policy / Arrangement Reference',true],['middle','Middle',false],['other','Other Information',true]] as const
  :[['last','Last Name:',false],['first','First Name:',false],['middle','Middle:',false],['account_ref','Account reference:',false],['policy_ref','Term reference:',false],['other','Other information:',false]] as const);
</script>
<p class="b-note">Enter at least one of the following pieces of information.</p>
<div class="b-form">
 {#each rows as [k,label,star]}
  <div class="b-field" class:invalid={!!errors[k]}><label for={prefix+k} class:b-star={star}>{label}</label><input id={prefix+k} bind:value={value[k]} maxlength="100" aria-invalid={!!errors[k]} aria-describedby={errors[k]?prefix+k+'-error':undefined}/>{#if errors[k]}<span class="err" id={prefix+k+'-error'}>{errors[k]}</span>{/if}</div>
 {/each}
</div>