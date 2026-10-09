<script lang="ts">
  import { api, type FieldType, type RuleInput, type RuleView, type Target } from '../api';
  import FieldPicker from '../FieldPicker.svelte';
  import { colLabel, humanize, recordName, typeText } from './labels';

  let { rule = null, prefill = null, headers, onSave, onCancel }: {
    rule?: RuleView | null; // null: add a new rule
    prefill?: { column: string; header: string } | null;
    headers: Record<string, string>;
    onSave: (input: RuleInput, reason: string) => Promise<void>;
    onCancel: () => void;
  } = $props();

  const key = (t: Target) => `${t.variant}|${t.instance ?? ''}|${t.field}`;
  const parse = (k: string): Target => { const [variant, instance, field] = k.split('|'); return instance ? { variant, instance, field } : { variant, field }; };
  const uid = Math.random().toString(36).slice(2, 8);

  // svelte-ignore state_referenced_locally
  let column = $state(rule?.excel_column ?? prefill?.column ?? '');
  // svelte-ignore state_referenced_locally
  let header = $state(rule?.header ?? prefill?.header ?? '');
  // svelte-ignore state_referenced_locally
  let target = $state(rule ? key(rule.target) : '');
  // svelte-ignore state_referenced_locally
  let caseMode = $state(rule?.transform.case ?? '');
  // svelte-ignore state_referenced_locally
  let mapRows = $state(Object.entries(rule?.transform.map ?? {}).map(([from, to]) => ({ from, to })));
  // svelte-ignore state_referenced_locally
  let conditional = $state(!!rule?.when);
  // svelte-ignore state_referenced_locally
  let whenColumn = $state(rule?.when?.column ?? 'B');
  // svelte-ignore state_referenced_locally
  let whenValues = $state((rule?.when?.not_in ?? rule?.when?.in ?? []).join(', '));
  // svelte-ignore state_referenced_locally
  let whenNot = $state(!!rule?.when?.not_in?.length);
  // svelte-ignore state_referenced_locally
  let off = $state(!!rule?.disabled);
  // svelte-ignore state_referenced_locally
  let offReason = $state(rule?.disabled ?? '');
  // svelte-ignore state_referenced_locally
  let attention = $state(rule?.attention ?? '');
  // svelte-ignore state_referenced_locally
  let note = $state(rule?.note ?? '');
  let reason = $state('');
  // svelte-ignore state_referenced_locally
  let picking = $state(!rule);
  let extra = $state<Target[]>([]);
  let busy = $state(false);
  let error = $state('');
  let targetType = $state<FieldType | undefined>(undefined);

  const options = $derived.by(() => {
    const seen = new Set<string>();
    const out: { k: string; label: string }[] = [];
    const add = (t: Target, tag: string) => {
      const k = key(t);
      if (seen.has(k)) return;
      seen.add(k);
      out.push({ k, label: `${recordName(t.variant, t.instance)} › ${humanize(t.field)}${tag}` });
    };
    if (rule) add(rule.target, ' (current)');
    if (rule?.reviewed) add(rule.reviewed.target, ' (reviewed default)');
    for (const t of extra) add(t, ' (picked)');
    for (const t of rule?.alternatives ?? []) add(t, ' (named in the report)');
    return out;
  });

  // Look up the chosen field's type so value maps can be checked as you type.
  $effect(() => {
    const t = target;
    targetType = undefined;
    if (!t) return;
    if (rule && t === key(rule.target)) { targetType = rule.target_type; return; }
    const { variant, field } = parse(t);
    api.variant(variant).then((v) => { if (target === t) targetType = v.fields.find((f) => f.key === field)?.type; }).catch(() => {});
  });

  const allowed = $derived(targetType?.kind === 'enum' ? targetType.enum ?? [] : targetType?.kind === 'boolean' ? ['true', 'false'] : []);

  function input(): RuleInput {
    const map: Record<string, string> = {};
    for (const r of mapRows) if (r.from.trim() !== '') map[r.from.trim()] = r.to.trim();
    const values = whenValues.split(',').map((v) => v.trim()).filter(Boolean);
    return {
      excel_column: column.trim().toUpperCase(), header: header.trim(), target: parse(target),
      transform: { case: caseMode || undefined, map: Object.keys(map).length ? map : undefined },
      when: conditional ? (whenNot ? { column: whenColumn.trim().toUpperCase(), not_in: values } : { column: whenColumn.trim().toUpperCase(), in: values }) : null,
      attention: attention.trim(), note: note.trim(), disabled: off ? offReason.trim() || reason.trim() : '',
    };
  }

  async function save() {
    error = '';
    if (!target) { error = 'Choose the Helix field this column is written to.'; return; }
    if (reason.trim().length < 3) { error = 'Say why you are making this change.'; return; }
    busy = true;
    try { await onSave(input(), reason.trim()); }
    catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }
</script>

<form class="form stack" onsubmit={(e) => { e.preventDefault(); save(); }}>
  {#if !rule}
    <div class="grid">
      <label for="col-{uid}">Sheet column</label>
      <div class="row wrap">
        <input id="col-{uid}" type="text" bind:value={column} placeholder="T" maxlength="3" style="width:70px" required />
        <label for="hdr-{uid}" class="sr">Header</label>
        <input id="hdr-{uid}" type="text" bind:value={header} placeholder="Header as written in row 1, e.g. Roof Covering" style="flex:1" required />
      </div>
    </div>
  {/if}

  <div class="grid">
    <label for="tgt-{uid}">Written to</label>
    <div class="stack tight">
      {#if options.length}
        <div class="row wrap">
          <select id="tgt-{uid}" bind:value={target} style="flex:1; min-width: 260px">
            {#if !target}<option value="">Choose a Helix field…</option>{/if}
            {#each options as o}<option value={o.k}>{o.label}</option>{/each}
          </select>
          <button type="button" class="btn sm" onclick={() => (picking = !picking)}>Other field…</button>
        </div>
      {/if}
      {#if target}<span class="xs muted"><span class="mono">{parse(target).variant}{parse(target).instance ? `#${parse(target).instance}` : ''}.{parse(target).field}</span>{#if targetType} · {typeText(targetType)}{/if}</span>{/if}
      {#if picking}
        <FieldPicker onCancel={() => (picking = false)} onPick={(t) => { extra = [...extra, t]; target = key(t); picking = false; }} />
      {/if}
    </div>

    <label for="case-{uid}">Letter case</label>
    <select id="case-{uid}" bind:value={caseMode} style="width: 220px">
      <option value="">Keep as written</option>
      <option value="upper">Change to UPPER CASE</option>
      <option value="lower">Change to lower case</option>
    </select>

    <span class="lbl">Translate values <span class="xs faint">sheet value → value written; others pass through; leave the right side empty to write nothing for that value</span></span>
    <div class="stack tight">
      {#each mapRows as r, i}
        <div class="row">
          <input type="text" aria-label="Sheet value" bind:value={r.from} placeholder="value in the sheet" />
          <span aria-hidden="true">→</span>
          <input type="text" aria-label="Value written" bind:value={r.to} placeholder="value written" list={allowed.length ? `allowed-${uid}` : undefined}
            class:invalid={allowed.length > 0 && r.to !== '' && !allowed.includes(r.to)} />
          <button type="button" class="btn sm ghost" aria-label="Remove translation" onclick={() => (mapRows = mapRows.filter((_, j) => j !== i))}>✕</button>
        </div>
      {/each}
      {#if allowed.length}
        <datalist id="allowed-{uid}">{#each allowed as a}<option value={a}></option>{/each}</datalist>
        <span class="xs muted">Allowed values: {allowed.join(', ')}</span>
      {/if}
      <div><button type="button" class="btn sm" onclick={() => (mapRows = [...mapRows, { from: '', to: '' }])}>+ Add translation</button></div>
    </div>

    <span class="lbl">Rows</span>
    <div class="stack tight">
      <label class="row chk"><input type="checkbox" bind:checked={conditional} /> Only for some rows</label>
      {#if conditional}
        <div class="row wrap">
          <span class="small">when</span>
          <input type="text" aria-label="Condition column" bind:value={whenColumn} maxlength="3" style="width:60px" />
          <span class="small muted">{colLabel(whenColumn.toUpperCase(), headers)}</span>
          <select aria-label="Condition kind" bind:value={whenNot}><option value={false}>is</option><option value={true}>has a value other than</option></select>
          <input type="text" aria-label="Condition values" bind:value={whenValues} placeholder="HO6, HO4" style="flex:1; min-width: 140px" />
        </div>
        <span class="xs faint">Separate several values with commas. Matching ignores upper/lower case.</span>
      {/if}
    </div>

    <span class="lbl">Status</span>
    <div class="stack tight">
      <label class="row chk"><input type="checkbox" bind:checked={off} /> Switch this rule off (nothing is written for this column)</label>
      {#if off}<input type="text" aria-label="Why the rule is off" bind:value={offReason} placeholder="Why it is off (defaults to the reason below)" />{/if}
    </div>

    <label for="att-{uid}">Flag at approval</label>
    <textarea id="att-{uid}" rows="2" bind:value={attention} placeholder="Optional: a warning the approver must acknowledge"></textarea>

    <label for="note-{uid}">Note</label>
    <textarea id="note-{uid}" rows="2" bind:value={note} placeholder="Optional: background for the next reviewer"></textarea>

    <label for="why-{uid}">Reason for this change <span class="req">*</span></label>
    <input id="why-{uid}" type="text" bind:value={reason} placeholder="e.g. Underwriting confirmed Class B is impact rated" required />
  </div>

  {#if error}<div class="err" role="alert">✕ {error}</div>{/if}
  <div class="row">
    <span class="xs muted">Checked against the live Helix model when you save. New uploads use the change at once.</span>
    <span class="spacer"></span>
    <button type="button" class="btn sm" onclick={onCancel} disabled={busy}>Cancel</button>
    <button type="submit" class="btn sm primary" disabled={busy}>{busy ? 'Saving…' : rule ? 'Save rule' : 'Add rule'}</button>
  </div>
</form>

<style>
  .form { padding: var(--space-4); background: var(--surface-2); border-top: 1px solid var(--border); border-left: 3px solid var(--accent); }
  .grid { display: grid; grid-template-columns: 170px 1fr; gap: var(--space-3); align-items: start; }
  .grid > label, .lbl { color: var(--text-muted); font-size: var(--fs-sm); padding-top: 6px; }
  .tight { gap: var(--space-2); }
  .chk { gap: 6px; color: var(--text); font-size: var(--fs-sm); padding-top: 0; }
  textarea { resize: vertical; width: 100%; }
  input.invalid { border-color: var(--danger); }
  .req { color: var(--danger); }
  .sr { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0 0 0 0); }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-2) var(--space-3); border-radius: var(--radius-sm); }
  @media (max-width: 720px) { .grid { grid-template-columns: 1fr; } .grid > label, .lbl { padding-top: 0; } }
</style>
