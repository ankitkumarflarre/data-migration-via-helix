<script lang="ts">
 import {href} from '../routes';
 let {current}:{current:string}=$props();
 const items=[['admin/errors','View Error Queue'],['admin/holds','Manage Hold Events'],['admin/disbursement','Disbursement Request'],['admin/transactions','Billing Transactions'],['admin/settings','Workspace Settings']];
</script>
<aside class="b-side"><div class="b-menu"><nav aria-label="Administration">{#each items as [p,label]}<a href={href(p)} class:on={current===p} aria-current={current===p?'page':undefined}>{label}</a>{/each}</nav></div></aside>
