<script lang="ts">
  import { onDestroy } from 'svelte';
  import { api, type Job, type RowResult } from './api';

  let { job = $bindable(), onBack }: { job: Job; onBack: () => void } = $props();

  let results = $state<RowResult[]>([]);
  let resultTotal = $state(0);
  let filter = $state('');
  let open = $state<number | null>(null);
  let timer: ReturnType<typeof setTimeout> | undefined;

  const p = $derived(job.progress);
  const running = $derived(p?.state === 'running' || p?.state === 'starting');
  const pct = $derived(p && p.total ? Math.round((p.done / p.total) * 100) : 0);
  const approval = $derived(job.approvals[job.approvals.length - 1]);
  const actions = ['created', 'updated', 'unchanged', 'reused'];

  async function refresh() {
    try {
      job = await api.job(job.id);
      const r = await api.results(job.id, filter, 0, 200);
      results = r.items; resultTotal = r.total;
    } catch { /* keep the last snapshot */ }
    if (job.progress?.state === 'running' || job.progress?.state === 'starting') timer = setTimeout(refresh, 800);
  }
  $effect(() => { const _f = filter; clearTimeout(timer); refresh(); });
  onDestroy(() => clearTimeout(timer));

  const variants = $derived(p ? Object.keys(p.counts).sort((a, b) => {
    const ia = job.plan.impact.findIndex((v) => v.variant === a);
    const ib = job.plan.impact.findIndex((v) => v.variant === b);
    return ia - ib;
  }) : []);
  const stateBadge: Record<string, string> = { running: 'neutral', starting: 'neutral', done: 'ok', aborted: 'bad' };
  const statusBadge: Record<string, string> = { written: 'ok', failed: 'bad', blocked: 'warn', skipped: 'neutral' };
</script>

{#if p}
  <section class="stack gap">
    <div class="card pad stack">
      <div class="row wrap">
        <h2>{p.dry_run ? 'Dry run' : 'Writing to Helix'}</h2>
        <span class="badge {stateBadge[p.state] ?? 'neutral'}">{p.state === 'done' ? '✓ done' : p.state === 'aborted' ? '✕ aborted' : '⟳ ' + p.state}</span>
        {#if p.dry_run}<span class="badge reference">nothing written</span>{/if}
        <span class="spacer"></span>
        {#if !running}
          <a class="btn" href="/api/jobs/{job.id}/results.csv" download>⬇ Results CSV</a>
          <button class="btn" onclick={onBack}>← Review / run again</button>
        {/if}
      </div>
      <div class="bar" role="progressbar" aria-valuenow={pct} aria-valuemin="0" aria-valuemax="100" aria-label="Rows processed">
        <div class="fill" class:bad={p.failed > 0} style="width: {pct}%"></div>
      </div>
      <div class="row wrap small">
        <span><strong>{p.done}</strong> / {p.total} rows</span>
        <span class="good">✓ {p.written} {p.dry_run ? 'would be written' : 'written'}</span>
        <span class:bad={p.failed > 0}>✕ {p.failed} failed</span>
        <span class="muted">{p.blocked} blocked (skipped)</span>
        {#if approval}<span class="xs faint mono">run {approval.run_id} · plan {approval.plan_hash.slice(0, 12)}</span>{/if}
      </div>
      {#if p.error}<div class="err" role="alert">✕ {p.error}</div>{/if}
    </div>

    <div class="cols">
      <div class="card">
        <div class="head"><h3>Records by table</h3></div>
        <div class="table-wrap flat"><table>
          <thead><tr><th>Table (variant)</th>{#each actions as a}<th class="num">{a}</th>{/each}</tr></thead>
          <tbody>
            {#each variants as v}
              <tr><td class="mono small">{v}</td>{#each actions as a}<td class="num">{p.counts[v]?.[a] ?? ''}</td>{/each}</tr>
            {:else}<tr><td colspan="5" class="muted">No records yet.</td></tr>{/each}
          </tbody>
        </table></div>
      </div>

      <div class="card">
        <div class="head row">
          <h3>Rows</h3><span class="spacer"></span>
          <select bind:value={filter} aria-label="Filter rows">
            <option value="">All</option><option value="written">Written</option><option value="failed">Failed</option><option value="blocked">Blocked</option>
          </select>
          <span class="xs muted">{resultTotal}{resultTotal > results.length ? `, first ${results.length}` : ''}</span>
        </div>
        <div class="table-wrap flat scroll"><table>
          <thead><tr><th>Row</th><th>Policy</th><th>Status</th><th>Details</th></tr></thead>
          <tbody>
            {#each results as r (r.row)}
              <tr>
                <td>{r.row}</td><td class="mono">{r.policy_number}</td>
                <td><span class="badge {statusBadge[r.status] ?? 'neutral'}">{r.status}</span></td>
                <td class="small">
                  {#if r.message}<div class="bad">{r.message}</div>{/if}
                  {#if r.actions?.length}
                    <button class="btn sm ghost" onclick={() => (open = open === r.row ? null : r.row)}>{open === r.row ? 'Hide' : `${r.actions.length} records`}</button>
                    {#if open === r.row}
                      <ul class="acts">{#each r.actions as a}<li><span class="badge neutral">{a.action}</span> <span class="mono xs">{a.variant}</span> <span class="mono xs faint">{a.record_id ?? ''}</span></li>{/each}</ul>
                    {/if}
                  {/if}
                </td>
              </tr>
            {:else}<tr><td colspan="4" class="muted">{running ? 'Waiting for the first rows…' : 'No rows.'}</td></tr>{/each}
          </tbody>
        </table></div>
      </div>
    </div>
  </section>
{:else}
  <div class="card pad">This job has not been approved yet. <button class="btn sm" onclick={onBack}>Back</button></div>
{/if}

<style>
  .gap { gap: var(--space-4); }
  .pad { padding: var(--space-5); }
  .spacer { flex: 1; }
  .bar { height: 10px; background: var(--surface-3); border-radius: 999px; overflow: hidden; }
  .fill { height: 100%; background: var(--accent); transition: width 0.3s; }
  .fill.bad { background: linear-gradient(90deg, var(--accent), var(--danger)); }
  .good { color: var(--success); }
  .bad { color: var(--danger); }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-3); border-radius: var(--radius-sm); }
  .cols { display: grid; grid-template-columns: minmax(0, 1fr) minmax(0, 1fr); gap: var(--space-4); align-items: start; }
  .head { padding: var(--space-3) var(--space-4); }
  .flat { border: none; border-top: 1px solid var(--border); border-radius: 0; }
  .scroll { max-height: 560px; overflow: auto; }
  .num { text-align: right; font-variant-numeric: tabular-nums; }
  .acts { list-style: none; padding: 0; margin: 4px 0 0; display: flex; flex-direction: column; gap: 2px; }
  a.btn { text-decoration: none; }
  @media (max-width: 1100px) { .cols { grid-template-columns: 1fr; } }
</style>
