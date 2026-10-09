<script lang="ts">
  import type { EffectiveRule, RuleOverride, Target } from './api';
  import FieldPicker from './FieldPicker.svelte';

  let { rule, current, onApply, onClose }: {
    rule: EffectiveRule;
    current: RuleOverride | undefined;
    onApply: (o: RuleOverride | null) => Promise<void>;
    onClose: () => void;
  } = $props();

  const key = (t: Target) => `${t.variant}|${t.field}`;
  const parse = (k: string): Target => { const [variant, field] = k.split('|'); return { variant, field }; };

  // svelte-ignore state_referenced_locally
  let target = $state(key(current?.target ?? rule.target));
  // svelte-ignore state_referenced_locally
  let exclude = $state(current?.exclude ?? false);
  // svelte-ignore state_referenced_locally
  let mapRows = $state<{ from: string; to: string }[]>(
    Object.entries(current?.map ?? rule.transform.map ?? {}).map(([from, to]) => ({ from, to })),
  );
  let picking = $state(false);
  let extra = $state<Target[]>([]);
  let busy = $state(false);
  let error = $state('');

  const options = $derived.by(() => {
    const seen = new Set<string>();
    const out: { k: string; label: string }[] = [];
    const add = (t: Target, tag: string) => {
      const k = key(t);
      if (seen.has(k)) return;
      seen.add(k);
      out.push({ k, label: `${t.variant}.${t.field}${tag}` });
    };
    add(rule.target, '  (report default)');
    if (current?.target) add(current.target, '  (current)');
    for (const t of extra) add(t, '  (picked)');
    for (const t of rule.alternatives) add(t, '');
    return out;
  });

  function mapObject(): Record<string, string> {
    const m: Record<string, string> = {};
    for (const r of mapRows) if (r.from.trim() !== '') m[r.from.trim()] = r.to;
    return m;
  }

  async function apply() {
    busy = true; error = '';
    const o: RuleOverride = {};
    if (exclude) o.exclude = true;
    if (target !== key(rule.target)) o.target = parse(target);
    const m = mapObject();
    const orig = rule.transform.map ?? {};
    if (JSON.stringify(m) !== JSON.stringify(orig)) o.map = m;
    try { await onApply(Object.keys(o).length ? o : null); onClose(); }
    catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }

  async function reset() {
    busy = true; error = '';
    try { await onApply(null); onClose(); } catch (e) { error = (e as Error).message; } finally { busy = false; }
  }
</script>

<div class="editor stack">
  <div class="row wrap">
    <strong>{rule.id}</strong>
    <span class="muted small">Column {rule.excel_column} · “{rule.header}”{#if rule.when} · only when column {rule.when.column} is {rule.when.in.join(' or ')}{/if}</span>
    <span class="spacer"></span>
    <label class="row chk"><input type="checkbox" bind:checked={exclude} /> Exclude this column</label>
  </div>

  {#if !exclude}
    <div class="grid">
      <label for="tgt-{rule.id}">Target table.field</label>
      <div class="row wrap">
        <select id="tgt-{rule.id}" bind:value={target} style="flex:1; min-width: 280px">
          {#each options as o}<option value={o.k}>{o.label}</option>{/each}
        </select>
        <button class="btn sm" onclick={() => (picking = !picking)}>Other field…</button>
      </div>
      {#if picking}
        <span></span>
        <FieldPicker onCancel={() => (picking = false)} onPick={(t) => { extra = [...extra, t]; target = key(t); picking = false; }} />
      {/if}

      <span class="lbl">Value map <span class="xs faint">(source → value; unmatched values pass through)</span></span>
      <div class="stack maps">
        {#each mapRows as r, i}
          <div class="row">
            <input type="text" aria-label="Source value" bind:value={r.from} placeholder="source value" />
            <span aria-hidden="true">→</span>
            <input type="text" aria-label="Mapped value" bind:value={r.to} placeholder="value written" />
            <button class="btn sm ghost" aria-label="Remove mapping" onclick={() => (mapRows = mapRows.filter((_, j) => j !== i))}>✕</button>
          </div>
        {/each}
        <div><button class="btn sm" onclick={() => (mapRows = [...mapRows, { from: '', to: '' }])}>+ Add mapping</button></div>
      </div>
    </div>
  {/if}

  {#if rule.report_locations?.length}
    <details class="xs muted">
      <summary>{rule.report_locations.length} locations named in the mapping report</summary>
      <div class="locs mono">{rule.report_locations.join('\n')}</div>
    </details>
  {/if}

  {#if error}<div class="err" role="alert">✕ {error}</div>{/if}
  <div class="row">
    <button class="btn sm ghost" onclick={reset} disabled={busy || !current}>Reset to default</button>
    <span class="spacer"></span>
    <button class="btn sm" onclick={onClose} disabled={busy}>Cancel</button>
    <button class="btn sm primary" onclick={apply} disabled={busy}>{busy ? 'Re-planning…' : 'Apply override'}</button>
  </div>
</div>

<style>
  .editor { padding: var(--space-4); background: var(--surface-2); border-left: 3px solid var(--accent); }
  .grid { display: grid; grid-template-columns: 170px 1fr; gap: var(--space-3); align-items: start; }
  .lbl { color: var(--text-muted); font-size: var(--fs-sm); padding-top: 6px; }
  .maps input { width: 200px; }
  .chk { gap: 6px; color: var(--text); }
  .spacer { flex: 1; }
  .locs { white-space: pre; max-height: 160px; overflow: auto; padding: var(--space-2); background: var(--surface); border-radius: var(--radius-sm); margin-top: 4px; }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-2) var(--space-3); border-radius: var(--radius-sm); }
  @media (max-width: 720px) { .grid { grid-template-columns: 1fr; } .maps input { width: 120px; } }
</style>
