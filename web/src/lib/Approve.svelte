<script lang="ts">
  import { api, type Job } from './api';

  let { job, onApproved, onBack }: { job: Job; onApproved: (j: Job) => void; onBack: () => void } = $props();

  const plan = $derived(job.plan);
  let acked = $state<Record<string, boolean>>({});
  let rowLimit = $state(5);
  let allRows = $state(false);
  let dryRun = $state(false);
  let busy = $state(false);
  let error = $state('');

  const allAcked = $derived(plan.attention.every((a) => acked[a.id]));
  const rowsToWrite = $derived(allRows ? plan.row_count - plan.blocked_rows : Math.min(rowLimit, plan.row_count - plan.blocked_rows));
  const perRow = $derived(plan.impact.filter((v) => v.scope === 'row'));
  const host = $derived(job.helix_url.replace(/^https?:\/\//, ''));
  const canApprove = $derived(allAcked && plan.blocking_count === 0 && rowsToWrite > 0 && !busy);

  const kindLabel: Record<string, string> = { rule: 'Mapping', template: 'Template value', generated: 'Generated', override: 'Your override' };

  async function approve() {
    busy = true; error = '';
    try {
      const j = await api.approve(job.id, {
        plan_hash: plan.plan_hash,
        acknowledged: Object.keys(acked).filter((k) => acked[k]),
        row_limit: allRows ? 0 : rowLimit,
        dry_run: dryRun,
      });
      onApproved(j);
    } catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }
</script>

<section class="grid">
  <div class="stack">
    <div class="card pad stack">
      <div class="row"><h2>Needs your attention</h2><span class="spacer"></span>
        <span class="badge {allAcked ? 'ok' : 'warn'}">{Object.values(acked).filter(Boolean).length} / {plan.attention.length} acknowledged</span></div>
      <p class="muted small">These mappings and values are not straight from the sheet, or were changed by you. Acknowledge each one, or go back and override it.</p>
      <div class="row">
        <button class="btn sm" onclick={() => (acked = Object.fromEntries(plan.attention.map((a) => [a.id, true])))}>Acknowledge all</button>
        <button class="btn sm ghost" onclick={onBack}>← Back to review to override</button>
      </div>
      <ul class="attn">
        {#each plan.attention as a (a.id)}
          <li class:done={acked[a.id]}>
            <label class="row top">
              <input type="checkbox" bind:checked={acked[a.id]} />
              <span class="stack tight">
                <span><span class="badge {a.kind === 'override' ? 'override' : a.kind === 'rule' ? 'warn' : a.kind}">{kindLabel[a.kind] ?? a.kind}</span> <span class="mono small">{a.title}</span></span>
                <span class="small muted">{a.detail}</span>
              </span>
            </label>
          </li>
        {:else}
          <li class="muted">Nothing needs attention.</li>
        {/each}
      </ul>
    </div>
  </div>

  <aside class="stack">
    <div class="card pad stack">
      <h2>Approve</h2>
      <dl class="facts">
        <dt>Workbook</dt><dd class="small">{job.file_name}</dd>
        <dt>Helix</dt><dd class="mono small">{host}</dd>
        <dt>Model bundle</dt><dd class="mono small">{plan.helix_bundle}</dd>
        <dt>Plan</dt><dd class="mono small" title={plan.plan_hash}>{plan.plan_hash.slice(0, 16)}…</dd>
        <dt>Ready rows</dt><dd>{(plan.row_count - plan.blocked_rows).toLocaleString()} of {plan.row_count.toLocaleString()}</dd>
      </dl>

      <fieldset class="stack tight">
        <legend class="small muted">Rows to process</legend>
        <label class="row"><input type="radio" name="scope" checked={!allRows} onchange={() => (allRows = false)} /> First
          <input type="number" min="1" max={plan.row_count} bind:value={rowLimit} disabled={allRows} style="width: 90px" /> rows</label>
        <label class="row"><input type="radio" name="scope" checked={allRows} onchange={() => (allRows = true)} /> All {(plan.row_count - plan.blocked_rows).toLocaleString()} rows</label>
      </fieldset>

      <label class="row mode" class:dry={dryRun}>
        <input type="checkbox" bind:checked={dryRun} />
        <span class="stack tight"><strong>Dry run only</strong><span class="xs muted">Optional: look up existing records and report create / update / unchanged without writing anything.</span></span>
      </label>

      <div class="small">
        Up to <strong>{(rowsToWrite * perRow.length + 2).toLocaleString()}</strong> records across {plan.impact.length} tables:
        2 shared records (issuer, product) + {perRow.length} per row. Existing records are found by business key or the ledger;
        only sheet-sourced fields are updated (D12).
      </div>

      {#if !dryRun}
        <div class="warnbox small">
          <strong>Writes to {host}</strong> through the Helix records API: creates new records, updates sheet-sourced
          fields of existing ones, and undoes a row's writes if any of them fails. <code>migrator cleanup</code> removes what was written.
        </div>
      {/if}

      {#if plan.blocking_count}<div class="err">✕ {plan.blocking_count} blocking issues — fix them in review first.</div>{/if}
      {#if error}<div class="err" role="alert">✕ {error}</div>{/if}

      <button class="btn primary big" disabled={!canApprove} onclick={approve}>
        {busy ? 'Starting…' : dryRun ? `Approve dry run (${rowsToWrite} rows)` : `Approve & write ${rowsToWrite} rows to Helix`}
      </button>
      {#if !allAcked}<span class="xs muted">Acknowledge every attention item to enable approval.</span>{/if}
    </div>
  </aside>
</section>

<style>
  .grid { display: grid; grid-template-columns: minmax(0, 1fr) 380px; gap: var(--space-5); align-items: start; }
  .pad { padding: var(--space-5); }
  .spacer { flex: 1; }
  .tight { gap: 2px; }
  .top { align-items: flex-start; }
  .attn { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--space-2); }
  .attn li { padding: var(--space-3); border: 1px solid var(--border); border-left: 3px solid var(--warning); border-radius: var(--radius-sm); background: var(--surface); }
  .attn li.done { border-left-color: var(--success); background: var(--surface-2); }
  .attn label { color: var(--text); cursor: pointer; }
  .attn input { margin-top: 3px; }
  .facts { display: grid; grid-template-columns: 110px 1fr; gap: 4px var(--space-3); margin: 0; }
  .facts dt { color: var(--text-muted); font-size: var(--fs-sm); }
  .facts dd { margin: 0; word-break: break-all; }
  fieldset { border: 1px solid var(--border); border-radius: var(--radius-sm); padding: var(--space-3); margin: 0; }
  fieldset label { color: var(--text); }
  .mode { align-items: flex-start; padding: var(--space-3); border-radius: var(--radius-sm); border: 1px solid var(--border); color: var(--text); cursor: pointer; }
  .mode.dry { background: var(--info-soft); border-color: var(--info); }
  .mode input { margin-top: 3px; }
  .warnbox { padding: var(--space-3); background: var(--warning-soft); color: var(--warning); border-radius: var(--radius-sm); }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-3); border-radius: var(--radius-sm); }
  .big { justify-content: center; padding: 10px 16px; }
  @media (max-width: 960px) { .grid { grid-template-columns: 1fr; } }
</style>
