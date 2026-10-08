<script lang="ts">
  import { report, ruleset, templates } from './model';
  let query = $state('');
  let status = $state('all');
  const columns = $derived(report.columns.filter(c => (status === 'all' || c.status.startsWith(status)) && [c.column, c.header, c.locations, c.candidates].some(v => v.toLowerCase().includes(query.toLowerCase()))));
</script>
<section class="card">
  <h2>Entity mapping register</h2>
  <p class="muted">81 spreadsheet columns · 20 confirmed · 39 require review · 22 not found. Confirmed spreadsheet mappings do not automatically approve UI correspondences.</p>
  <div class="toolbar">
    <label>Search mappings <input bind:value={query} type="search" placeholder="Column, header or entity" /></label>
    <label>Status <select bind:value={status}><option value="all">All statuses</option><option value="Confirmed">Confirmed</option><option value="Review">Requires review</option><option value="Not Found">Not found</option></select></label>
    <span class="muted">{columns.length} columns</span>
  </div>
  {#each columns as column (column.column)}
    {@const rule = ruleset.rules.find(r => r.excel_column === column.column)}
    <details>
      <summary><span class="mono">{column.column}</span> <strong>{column.header || '(Blank source header)'}</strong> <span class:confirmed={!!rule} class="status">{column.status}</span></summary>
      {#if rule}
        <p><b>Executable target:</b> <code>{rule.target.variant}.{rule.target.field}</code></p>
        <p><b>Transform:</b> <code>{JSON.stringify(rule.transform)}</code></p>
        {#if 'attention' in rule && rule.attention}<p class="warning">{rule.attention}</p>{/if}
        {#if 'note' in rule}<p>{rule.note}</p>{/if}
        <p class="muted">Rule {rule.id} · {rule.alternatives.length} alternatives. Change targets in Migrate → Review &amp; override.</p>
      {:else}<p class="warning">No executable migration rule for this column.</p>{/if}
      <p>{column.note}</p>
      {#if column.locations}<details><summary>Original report locations (legacy SQL names)</summary><pre>{column.locations}</pre></details>{/if}
      {#if column.candidates}<details><summary>Unapproved report candidates</summary><pre>{column.candidates}</pre></details>{/if}
    </details>
  {/each}
  {#if !columns.length}<p>No mappings match this search.</p>{/if}
</section>
<section class="card">
  <h2>Entity prerequisites and templates</h2>
  <p class="muted">Issuer and product are shared per job. Policyholder → location → dwelling asset → wind verification are created in dependency order. Policy references the policyholder, issuer and product.</p>
  <p class="warning">Dwelling, section_icoverages, line, location_address and loss_ratio_analysis have no policy reference; the migrator's ledger provides the link. Territory is a name match to a reporting entity, not a confirmed rating-territory model.</p>
  {#each templates.records as record}
    <details><summary><code>{record.variant}</code> <span class="status">{record.scope}</span></summary>
      <p>Lookup key: <code>{'key' in record ? record.key : 'Ledger only'}</code></p>
      <pre>{JSON.stringify('fields' in record ? record.fields : {}, null, 2)}</pre>
    </details>
  {/each}
</section>
<style>
  .card { padding: 24px; margin-bottom: 20px; }
  .toolbar { display: flex; gap: 16px; flex-wrap: wrap; align-items: end; margin: 20px 0; }
  label { display: grid; gap: 6px; }
  details { border-top: 1px solid var(--border); padding: 12px 0; }
  summary { cursor: pointer; overflow-wrap: anywhere; }
  summary strong { margin: 0 12px; }
  .status { font-size: 12px; color: var(--text-muted); }
  .confirmed { color: var(--success); }
  .warning { color: var(--warning); }
  pre { white-space: pre-wrap; overflow-wrap: anywhere; font-size: 12px; background: var(--surface-2); padding: 12px; }
  code { overflow-wrap: anywhere; }
</style>
