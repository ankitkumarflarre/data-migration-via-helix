<script lang="ts">
 import {href} from '../routes';
 import Icon from '../../personalhome/WorkspaceIcon.svelte';
 import AdminMenu from './AdminMenu.svelte';
 import Page from '../ui/Page.svelte';
</script>
<div class="bw-body"><AdminMenu current="admin"/><Page title="Administration" subtitle="Navigate to an administration function from the available list." cards>
 <div class="tiles">{#each [['admin/errors','View Error Queue','Errors raised by billing processes, with resubmit.','database'],['admin/holds','Manage Hold Events','Area hold events and the accounts on them.','shield'],['admin/disbursement','Disbursement Request','Start or stop the disbursement interface service.','transfer'],['admin/transactions','Billing Transactions','Add a billing transaction to an account.','dollar'],['admin/settings','Workspace Settings','Auto-authorize limit, Paid In Full and plans.','rules']] as [p,label,note,icon]}<a class="tile" href={href(p)}><span class="b-ico soft"><Icon name={icon} size={14}/></span><span><b>{label}</b><small>{note}</small></span><span class="go" aria-hidden="true">›</span></a>{/each}</div>
</Page></div>
<style>.tiles{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:12px}.tile{display:flex;gap:12px;align-items:center;padding:14px;background:var(--surface);border:1px solid var(--border);border-radius:12px;box-shadow:var(--shadow);text-decoration:none;color:var(--text)}.tile:hover{border-color:var(--accent)}.tile span:nth-child(2){display:grid;gap:2px;flex:1}.tile b{font-size:14px}.tile small{font-size:12px;color:var(--text-muted)}.go{color:var(--accent);font-size:18px}</style>