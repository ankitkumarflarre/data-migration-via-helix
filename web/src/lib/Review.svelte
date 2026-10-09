<script lang="ts">
  import { api, fmtValue, ruleInput, rulesApi, type EffectiveRule, type FieldImpact, type Issue, type Job, type Overrides, type PlanRow, type RuleOverride, type VariantImpact } from './api';
  import RuleEditor from './RuleEditor.svelte';

  let { job = $bindable(), onNext }: { job: Job; onNext: () => void } = $props();

  type Tab = 'impact' | 'issues' | 'rows';
  let tab = $state<Tab>('impact');
  let editing = $state<string | null>(null); // "rule:HO-05" | "tmpl:variant|field"
  let tmplValue = $state('');
  let busy = $state(false);
  let error = $state('');
  let collapsed = $state<Record<string, boolean>>({});

  const plan = $derived(job.plan);
  const ruleById = $derived(Object.fromEntries(plan.rules.map((r) => [r.id, r])));
  const excluded = $derived(plan.rules.filter((r) => r.excluded));
  const sheetFields = $derived(plan.impact.reduce((n, v) => n + v.fields.filter((f) => f.source === 'sheet').length, 0));
  const totalRecords = $derived(plan.impact.reduce((n, v) => n + v.records, 0));
  const overrideCount = $derived(Object.keys(job.overrides.rules ?? {}).length + Object.keys(job.overrides.templates ?? {}).length);

  async function save(ov: Overrides) {
    busy = true; error = '';
    try { job = await api.setOverrides(job.id, ov); }
    catch (e) { error = (e as Error).message; throw e; }
    finally { busy = false; }
  }

  async function setRule(id: string, o: RuleOverride | null) {
    const rules = { ...(job.overrides.rules ?? {}) };
    if (o) rules[id] = o; else delete rules[id];
    await save({ ...job.overrides, rules });
  }

  // Make an override the rule's default for every new upload (Rules tab), then
  // re-plan this file with the updated rules instead of the override.
  async function saveDefault(rule: EffectiveRule, o: RuleOverride, reason: string) {
    const input = ruleInput(rule);
    if (o.target) input.target = o.target;
    if (o.map) input.transform = { ...input.transform, map: o.map };
    if (o.exclude) input.disabled = reason;
    await rulesApi.put(job.rule_set, rule.id, input, reason);
    const rules = { ...(job.overrides.rules ?? {}) };
    delete rules[rule.id];
    await save({ ...job.overrides, rules });
    await refreshRules();
  }

  async function refreshRules() {
    busy = true; error = '';
    try { job = await api.refreshRules(job.id); }
    catch (e) { error = (e as Error).message; throw e; }
    finally { busy = false; }
  }

  async function setTemplate(k: string, v: string | null) {
    const templates = { ...(job.overrides.templates ?? {}) };
    if (v !== null) templates[k] = v; else delete templates[k];
    await save({ ...job.overrides, templates });
    editing = null;
  }

  const srcLabel: Record<string, string> = {
    sheet: 'Sheet', template: 'Template', generated: 'Generated', reference: 'Reference', override: 'Overridden',
  };

  function from(f: FieldImpact): string {
    if (f.rule_id) return `${f.column} · ${f.header ?? ''}`;
    return f.template ?? '';
  }

  // ---- issues & rows tabs ----
  let issues = $state<Issue[]>([]);
  let issueTotal = $state(0);
  let issueSeverity = $state('');
  let rows = $state<PlanRow[]>([]);
  let rowTotal = $state(0);
  let rowOffset = $state(0);
  let openRow = $state<number | null>(null);

  $effect(() => {
    if (tab !== 'issues') return;
    const _hash = plan.plan_hash;
    api.issues(job.id, issueSeverity, 0, 200).then((p) => { issues = p.items; issueTotal = p.total; }).catch((e) => (error = e.message));
  });
  $effect(() => {
    if (tab !== 'rows') return;
    const _hash = plan.plan_hash;
    api.rows(job.id, rowOffset, 15).then((p) => { rows = p.items; rowTotal = p.total; }).catch((e) => (error = e.message));
  });

  function scopeLabel(v: VariantImpact) { return v.scope === 'job' ? 'once per upload' : 'per policy row'; }
</script>

<section class="stack gap">
  <div class="summary card">
    <div class="stat"><div class="n">{plan.row_count.toLocaleString()}</div><div class="xs muted">policy rows</div></div>
    <div class="stat"><div class="n">{plan.impact.length}</div><div class="xs muted">tables impacted</div></div>
    <div class="stat"><div class="n">{sheetFields}</div><div class="xs muted">columns from sheet</div></div>
    <div class="stat"><div class="n">{totalRecords.toLocaleString()}</div><div class="xs muted">records planned</div></div>
    <div class="stat"><div class="n {plan.blocking_count ? 'bad' : 'good'}">{plan.blocking_count}</div><div class="xs muted">blocking issues</div></div>
    <div class="stat"><div class="n {plan.attention.length ? 'warnc' : ''}">{plan.attention.length}</div><div class="xs muted">need attention</div></div>
    <div class="stat"><div class="n">{overrideCount}</div><div class="xs muted">overrides</div></div>
    <span class="spacer"></span>
    <div class="stack right">
      <button class="btn primary" disabled={plan.blocking_count > 0 || busy} onclick={onNext}>Continue to approval →</button>
      <span class="xs faint mono" title={plan.plan_hash}>plan {plan.plan_hash.slice(0, 12)} · Helix {plan.helix_bundle}</span>
    </div>
  </div>

  {#if error}<div class="err" role="alert">✕ {error}</div>{/if}
  {#if job.rules_outdated}
    <div class="note warn-note" role="status">
      ✎ The mapping rules were changed in the Rules tab after this file was planned. This plan still uses the earlier rules.
      <button class="btn sm" disabled={busy} onclick={() => refreshRules().catch(() => {})}>Re-plan with the latest rules</button>
      <a class="btn sm ghost" href="#rules">See the rules</a>
    </div>
  {/if}
  {#if plan.blocking_count > 0}
    <div class="note bad-note">✕ {plan.blocking_count} blocking issues on {plan.blocked_rows} rows. Fix them with an override (value map, other target, or exclude) before approving. <button class="btn sm" onclick={() => { tab = 'issues'; issueSeverity = 'blocking'; }}>Show issues</button></div>
  {/if}

  <div class="tabs" role="tablist">
    <button role="tab" aria-selected={tab === 'impact'} class:on={tab === 'impact'} onclick={() => (tab = 'impact')}>Impacted tables & columns</button>
    <button role="tab" aria-selected={tab === 'issues'} class:on={tab === 'issues'} onclick={() => (tab = 'issues')}>Issues ({plan.blocking_count + plan.warning_count})</button>
    <button role="tab" aria-selected={tab === 'rows'} class:on={tab === 'rows'} onclick={() => (tab = 'rows')}>Row preview</button>
    {#if busy}<span class="xs muted busy">Re-planning…</span>{/if}
  </div>

  {#if tab === 'impact'}
    <div class="legend xs muted row wrap">
      <span class="badge sheet">Sheet</span> mapped by a confirmed rule
      <span class="badge template">Template</span> fixed or derived value
      <span class="badge generated">Generated</span> required placeholder (D6)
      <span class="badge reference">Reference</span> link to another record
      <span class="badge override">Overridden</span> changed by you
    </div>

    {#each plan.impact as v (v.variant)}
      <article class="card variant">
        <button class="vhead" aria-expanded={!collapsed[v.variant]} onclick={() => (collapsed[v.variant] = !collapsed[v.variant])}>
          <span class="chev" aria-hidden="true">{collapsed[v.variant] ? '▸' : '▾'}</span>
          <span class="vname mono">{v.variant}</span>
          <span class="badge neutral">{v.entity}</span>
          <span class="badge {v.scope === 'job' ? 'reference' : 'neutral'}">{scopeLabel(v)}</span>
          {#if v.key}<span class="xs muted">find-or-create by <code>{v.key}</code></span>{:else}<span class="xs muted">no business key · linked via ledger</span>{/if}
          <span class="spacer"></span>
          <span class="xs muted">{v.records.toLocaleString()} records · {v.fields.length} fields</span>
        </button>
        {#if !collapsed[v.variant]}
          <div class="table-wrap flat">
            <table>
              <thead><tr><th>Field</th><th>Source</th><th>From</th><th>Type</th><th>Examples (before → after)</th><th>Values</th><th></th></tr></thead>
              <tbody>
                {#each v.fields as f (f.field)}
                  {@const rid = f.rule_id?.split(',')[0]}
                  {@const rowKey = rid ? `rule:${rid}` : `tmpl:${v.variant}|${f.field}`}
                  <tr class:attn={!!f.attention}>
                    <td><span class="mono">{f.field}</span>{#if f.required}<span class="req" title="Required by Helix">*</span>{/if}
                      {#if f.attention}<div class="xs warnc">⚠ {f.attention}</div>{/if}</td>
                    <td><span class="badge {f.overridden ? 'override' : f.source}">{f.overridden && f.source === 'sheet' ? 'Sheet · overridden' : srcLabel[f.source]}</span></td>
                    <td class="small">{from(f)}{#if f.rule_id}<div class="xs faint">{f.rule_id}</div>{/if}{#if f.condition}<div class="xs muted">{f.condition}</div>{/if}</td>
                    <td class="xs mono muted">{f.type.type}</td>
                    <td class="small">
                      {#each f.samples ?? [] as s}
                        <div class="sample">{#if s.before !== ''}<span class="mono">{s.before}</span> → {/if}<span class="mono" class:bad={s.after.startsWith('✗')}>{s.after}</span></div>
                      {:else}<span class="faint">—</span>{/each}
                    </td>
                    <td class="small nowrap">{f.values.toLocaleString()}{#if f.errors}<div class="bad xs">{f.errors} errors</div>{/if}</td>
                    <td class="nowrap">
                      {#if f.source !== 'reference'}
                        <button class="btn sm" onclick={() => { editing = editing === rowKey ? null : rowKey; tmplValue = job.overrides.templates?.[`${v.variant}|${f.field}`] ?? ''; }}>
                          {editing === rowKey ? 'Close' : 'Override'}
                        </button>
                      {/if}
                    </td>
                  </tr>
                  {#if editing === rowKey}
                    <tr class="edit-row"><td colspan="7">
                      {#if rid && ruleById[rid]}
                        <RuleEditor rule={ruleById[rid]} current={job.overrides.rules?.[rid]} onApply={(o) => setRule(rid, o)} onSaveDefault={(o, reason) => saveDefault(ruleById[rid], o, reason)} onClose={() => (editing = null)} />
                      {:else}
                        <div class="tedit row wrap">
                          <label for="tv-{f.field}">Value for <span class="mono">{v.variant}.{f.field}</span></label>
                          {#if f.type.enum?.length}
                            <select id="tv-{f.field}" bind:value={tmplValue}>
                              <option value="">Choose…</option>
                              {#each f.type.enum as e}<option value={e}>{e}</option>{/each}
                            </select>
                          {:else}
                            <input id="tv-{f.field}" type="text" bind:value={tmplValue} placeholder={f.type.kind === 'date' ? 'YYYY-MM-DD' : f.type.kind === 'boolean' ? 'true / false' : 'value'} />
                          {/if}
                          <span class="xs faint">Applies to every {v.scope === 'job' ? 'upload record' : 'row'}.</span>
                          <span class="spacer"></span>
                          <button class="btn sm ghost" disabled={busy || job.overrides.templates?.[`${v.variant}|${f.field}`] === undefined} onclick={() => setTemplate(`${v.variant}|${f.field}`, null)}>Reset</button>
                          <button class="btn sm primary" disabled={busy || tmplValue === ''} onclick={() => setTemplate(`${v.variant}|${f.field}`, tmplValue)}>Apply</button>
                        </div>
                      {/if}
                    </td></tr>
                  {/if}
                {/each}
              </tbody>
            </table>
          </div>
        {/if}
      </article>
    {/each}

    {#if excluded.length}
      <article class="card variant">
        <div class="vhead static"><strong>Excluded columns</strong><span class="xs muted">not written</span></div>
        <div class="table-wrap flat"><table><tbody>
          {#each excluded as r}
            <tr><td class="mono">{r.id}</td><td>Column {r.excel_column} · {r.header}</td><td class="mono small">{r.target.variant}.{r.target.field}</td>
              <td>{#if r.disabled && !job.overrides.rules?.[r.id]?.exclude}
                <span class="xs muted">Switched off in the Rules tab: {r.disabled}</span> <a class="btn sm ghost" href="#rules">Rules</a>
              {:else}<button class="btn sm" disabled={busy} onclick={() => setRule(r.id, null)}>Restore</button>{/if}</td></tr>
          {/each}
        </tbody></table></div>
      </article>
    {/if}
  {:else if tab === 'issues'}
    <div class="row">
      <label for="sev">Severity</label>
      <select id="sev" bind:value={issueSeverity}><option value="">All</option><option value="blocking">Blocking</option><option value="warning">Warning</option></select>
      <span class="xs muted">{issueTotal} issues{issueTotal > issues.length ? `, first ${issues.length} shown` : ''}</span>
    </div>
    {#if issues.length === 0}
      <div class="card empty">✓ No issues. Every confirmed column converted cleanly.</div>
    {:else}
      <div class="table-wrap"><table>
        <thead><tr><th>Row</th><th>Column</th><th>Rule</th><th>Target</th><th>Severity</th><th>Message</th></tr></thead>
        <tbody>{#each issues as i}<tr>
          <td>{i.row || 'job'}</td><td>{i.column ?? ''}</td><td class="mono">{i.rule_id ?? ''}</td><td class="mono small">{i.target ?? ''}</td>
          <td><span class="badge {i.severity === 'blocking' ? 'bad' : 'warn'}">{i.severity === 'blocking' ? '✕ blocking' : '⚠ warning'}</span></td><td>{i.message}</td>
        </tr>{/each}</tbody>
      </table></div>
    {/if}
  {:else}
    <div class="row">
      <button class="btn sm" disabled={rowOffset === 0} onclick={() => (rowOffset = Math.max(0, rowOffset - 15))}>← Prev</button>
      <span class="xs muted">Rows {rowOffset + 1}–{Math.min(rowOffset + 15, rowTotal)} of {rowTotal}</span>
      <button class="btn sm" disabled={rowOffset + 15 >= rowTotal} onclick={() => (rowOffset += 15)}>Next →</button>
    </div>
    <div class="table-wrap"><table>
      <thead><tr><th>Sheet row</th><th>Policy number</th><th>Records</th><th>Status</th><th></th></tr></thead>
      <tbody>
        {#each rows as r}
          <tr>
            <td>{r.row}</td><td class="mono">{r.policy_number}</td><td>{r.records.length}</td>
            <td>{#if r.blocked}<span class="badge bad">✕ blocked</span>{:else}<span class="badge ok">✓ ready</span>{/if}</td>
            <td><button class="btn sm" onclick={() => (openRow = openRow === r.row ? null : r.row)}>{openRow === r.row ? 'Hide' : 'Show records'}</button></td>
          </tr>
          {#if openRow === r.row}
            <tr class="edit-row"><td colspan="5">
              <div class="recs">
                {#each r.records as rec}
                  <div class="rec card">
                    <div class="row wrap"><strong class="mono small">{rec.variant}</strong>{#if rec.key_value}<span class="xs muted">key {rec.key_field} = <code>{rec.key_value}</code></span>{/if}</div>
                    <table class="kv"><tbody>
                      {#each Object.entries(rec.fields).sort() as [k, val]}
                        <tr><td class="mono xs">{k}</td><td class="mono xs">{fmtValue(val)}
                          {#if rec.sheet_fields.includes(k)}<span class="badge sheet">sheet</span>{:else if rec.generated?.includes(k)}<span class="badge generated">generated</span>{/if}</td></tr>
                      {/each}
                      {#each Object.entries(rec.refs ?? {}) as [k, to]}
                        <tr><td class="mono xs">{k}</td><td class="xs"><span class="badge reference">→ {to}</span></td></tr>
                      {/each}
                    </tbody></table>
                  </div>
                {/each}
              </div>
            </td></tr>
          {/if}
        {/each}
      </tbody>
    </table></div>
  {/if}
</section>

<style>
  .gap { gap: var(--space-4); }
  .summary { display: flex; align-items: center; gap: var(--space-5); padding: var(--space-4) var(--space-5); flex-wrap: wrap; }
  .stat .n { font-size: var(--fs-xl); font-weight: 700; font-variant-numeric: tabular-nums; }
  .good { color: var(--success); }
  .bad { color: var(--danger); }
  .warnc { color: var(--warning); }
  .right { align-items: flex-end; gap: 4px; }
  .spacer { flex: 1; }
  .err, .bad-note { background: var(--danger-soft); color: var(--danger); padding: var(--space-3); border-radius: var(--radius-sm); }
  .bad-note { display: flex; gap: var(--space-3); align-items: center; flex-wrap: wrap; }
  .warn-note { display: flex; gap: var(--space-3); align-items: center; flex-wrap: wrap; background: var(--warning-soft); color: var(--warning); padding: var(--space-3); border-radius: var(--radius-sm); }
  .tabs { display: flex; gap: var(--space-1); border-bottom: 1px solid var(--border); align-items: center; }
  .tabs button { border: none; background: none; padding: 8px 14px; font: 500 var(--fs) var(--font); color: var(--text-muted); cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -1px; }
  .tabs button.on { color: var(--text); border-bottom-color: var(--accent); }
  .busy { margin-left: auto; }
  .legend { gap: var(--space-2); }
  .variant { overflow: hidden; }
  .vhead { display: flex; align-items: center; gap: var(--space-3); width: 100%; padding: var(--space-3) var(--space-4); border: none; background: var(--surface); color: var(--text); cursor: pointer; text-align: left; flex-wrap: wrap; font: inherit; }
  .vhead.static { cursor: default; }
  .vhead:hover:not(.static) { background: var(--surface-2); }
  .vname { font-weight: 600; }
  .chev { color: var(--text-faint); width: 12px; }
  .flat { border: none; border-top: 1px solid var(--border); border-radius: 0; }
  .req { color: var(--danger); margin-left: 2px; }
  .sample { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 360px; }
  .nowrap { white-space: nowrap; }
  tr.attn td:first-child { box-shadow: inset 3px 0 0 var(--warning); }
  .edit-row td { padding: 0; background: var(--surface-2); }
  .edit-row:hover td { background: var(--surface-2); }
  .tedit { padding: var(--space-3) var(--space-4); border-left: 3px solid var(--accent); }
  .empty { padding: var(--space-5); color: var(--success); }
  .recs { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: var(--space-3); padding: var(--space-3); }
  .rec { padding: var(--space-3); display: flex; flex-direction: column; gap: var(--space-2); }
  .kv td { padding: 2px 6px; border: none; }
  .kv tr:hover td { background: none; }
</style>
