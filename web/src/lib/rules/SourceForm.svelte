<script lang="ts">
  import type { FieldSource, TemplateFieldView } from '../api';
  import { colLabel, typeText } from './labels';

  let { field, headers, onSave, onCancel }: {
    field: TemplateFieldView;
    headers: Record<string, string>;
    onSave: (source: FieldSource, reason: string) => Promise<void>;
    onCancel: () => void;
  } = $props();

  type Kind = 'const' | 'col' | 'format';
  const uid = Math.random().toString(36).slice(2, 8);
  // svelte-ignore state_referenced_locally
  const s = field.source; // the form edits a copy; it is remounted per field
  // svelte-ignore state_referenced_locally
  let kind = $state<Kind>(s.col ? 'col' : s.format ? 'format' : 'const');
  // svelte-ignore state_referenced_locally
  let value = $state(s.const ?? '');
  // svelte-ignore state_referenced_locally
  let col = $state(s.col ?? '');
  // svelte-ignore state_referenced_locally
  let mapRows = $state(Object.entries(s.map ?? {}).map(([from, to]) => ({ from, to })));
  // svelte-ignore state_referenced_locally
  let format = $state(s.format ?? '');
  // svelte-ignore state_referenced_locally
  let attention = $state(s.attention ?? '');
  let reason = $state('');
  let busy = $state(false);
  let error = $state('');

  const allowed = $derived(field.type?.kind === 'enum' ? field.type.enum ?? [] : field.type?.kind === 'boolean' ? ['true', 'false'] : []);

  function source(): FieldSource {
    const out: FieldSource = {};
    if (kind === 'const') out.const = value.trim();
    if (kind === 'col') {
      out.col = col.trim().toUpperCase();
      const m: Record<string, string> = {};
      for (const r of mapRows) if (r.from.trim()) m[r.from.trim()] = r.to.trim();
      if (Object.keys(m).length) out.map = m;
    }
    if (kind === 'format') out.format = format.trim();
    if (attention.trim()) out.attention = attention.trim();
    return out;
  }

  async function save() {
    error = '';
    if (reason.trim().length < 3) { error = 'Say why you are making this change.'; return; }
    busy = true;
    try { await onSave(source(), reason.trim()); }
    catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }
</script>

<form class="form stack" onsubmit={(e) => { e.preventDefault(); save(); }}>
  <div class="grid">
    <span class="lbl">Value comes from</span>
    <div class="row wrap" role="radiogroup" aria-label="Value comes from">
      <label class="row chk"><input type="radio" bind:group={kind} value="const" /> A fixed value</label>
      <label class="row chk"><input type="radio" bind:group={kind} value="col" /> A sheet column</label>
      <label class="row chk"><input type="radio" bind:group={kind} value="format" /> A pattern</label>
    </div>

    {#if kind === 'const'}
      <label for="v-{uid}">Fixed value</label>
      <div class="stack tight">
        {#if allowed.length}
          <select id="v-{uid}" bind:value={value} style="width: 260px">
            <option value="">Choose…</option>
            {#each allowed as a}<option value={a}>{a}</option>{/each}
          </select>
        {:else}
          <input id="v-{uid}" type="text" bind:value={value} />
        {/if}
        {#if field.type}<span class="xs muted">{typeText(field.type)}</span>{/if}
      </div>
    {:else if kind === 'col'}
      <label for="c-{uid}">Sheet column</label>
      <div class="row">
        <input id="c-{uid}" type="text" bind:value={col} maxlength="3" style="width: 70px" />
        <span class="small muted">{col ? colLabel(col.toUpperCase(), headers) : ''}</span>
      </div>
      <span class="lbl">Translate values</span>
      <div class="stack tight">
        {#each mapRows as r, i}
          <div class="row">
            <input type="text" aria-label="Sheet value" bind:value={r.from} placeholder="value in the sheet" />
            <span aria-hidden="true">→</span>
            <input type="text" aria-label="Value written" bind:value={r.to} placeholder="value written" class:invalid={allowed.length > 0 && r.to !== '' && !allowed.includes(r.to)} />
            <button type="button" class="btn sm ghost" aria-label="Remove translation" onclick={() => (mapRows = mapRows.filter((_, j) => j !== i))}>✕</button>
          </div>
        {/each}
        {#if allowed.length}<span class="xs muted">Allowed values: {allowed.join(', ')}</span>{/if}
        <div><button type="button" class="btn sm" onclick={() => (mapRows = [...mapRows, { from: '', to: '' }])}>+ Add translation</button></div>
      </div>
    {:else}
      <label for="f-{uid}">Pattern</label>
      <div class="stack tight">
        <input id="f-{uid}" type="text" bind:value={format} placeholder={'e.g. PH-{PN} or {C}, FL'} />
        <span class="xs faint">{'{PN}'} is the policy number; {'{C}'} is the value of column C.</span>
      </div>
    {/if}

    <label for="a-{uid}">Flag at approval</label>
    <textarea id="a-{uid}" rows="2" bind:value={attention} placeholder="Optional: a warning the approver must acknowledge"></textarea>

    <label for="r-{uid}">Reason for this change <span class="req">*</span></label>
    <input id="r-{uid}" type="text" bind:value={reason} placeholder="e.g. Issuer name confirmed by finance" required />
  </div>
  {#if error}<div class="err" role="alert">✕ {error}</div>{/if}
  <div class="row">
    <span class="spacer"></span>
    <button type="button" class="btn sm" onclick={onCancel} disabled={busy}>Cancel</button>
    <button type="submit" class="btn sm primary" disabled={busy}>{busy ? 'Saving…' : 'Save value'}</button>
  </div>
</form>

<style>
  .form { padding: var(--space-4); background: var(--surface-2); border-left: 3px solid var(--accent); }
  .grid { display: grid; grid-template-columns: 170px 1fr; gap: var(--space-3); align-items: start; }
  .grid > label, .lbl { color: var(--text-muted); font-size: var(--fs-sm); padding-top: 6px; }
  .tight { gap: var(--space-2); }
  .chk { gap: 6px; color: var(--text); font-size: var(--fs-sm); }
  textarea { resize: vertical; width: 100%; }
  input.invalid { border-color: var(--danger); }
  .req { color: var(--danger); }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-2) var(--space-3); border-radius: var(--radius-sm); }
  @media (max-width: 720px) { .grid { grid-template-columns: 1fr; } .grid > label, .lbl { padding-top: 0; } }
</style>
