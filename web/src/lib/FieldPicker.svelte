<script lang="ts">
  import { api, type Leaf, type SchemaField, type Target } from './api';

  let { onPick, onCancel }: { onPick: (t: Target) => void; onCancel: () => void } = $props();

  let leaves = $state<Leaf[]>([]);
  let query = $state('');
  let variant = $state('');
  let fields = $state<SchemaField[]>([]);
  let field = $state('');
  let error = $state('');
  let loading = $state(true);

  $effect(() => {
    api.variants()
      .then((r) => { leaves = r.variants; })
      .catch((e) => (error = (e as Error).message))
      .finally(() => (loading = false));
  });

  const matches = $derived(
    query.trim().length < 2 ? [] :
      leaves.filter((l) => l.variant.toLowerCase().includes(query.trim().toLowerCase()) || l.title.toLowerCase().includes(query.trim().toLowerCase())).slice(0, 40),
  );

  async function choose(v: string) {
    variant = v; field = ''; fields = []; error = '';
    try { fields = (await api.variant(v)).fields; } catch (e) { error = (e as Error).message; }
  }
</script>

<div class="picker stack">
  <div class="row wrap">
    <input type="search" placeholder={loading ? 'Loading Helix variants…' : 'Search a Helix table (variant), e.g. dwelling'} bind:value={query} aria-label="Search variants" style="flex:1" />
    <button class="btn sm ghost" onclick={onCancel}>Cancel</button>
  </div>
  {#if error}<div class="xs bad">{error}</div>{/if}
  {#if matches.length}
    <div class="list" role="listbox" aria-label="Variants">
      {#each matches as l}
        <button class="opt" class:sel={l.variant === variant} role="option" aria-selected={l.variant === variant} onclick={() => choose(l.variant)}>
          <span class="mono">{l.variant}</span> <span class="xs faint">{l.module}</span>
        </button>
      {/each}
    </div>
  {:else if query.trim().length >= 2 && !loading}
    <div class="xs muted">No variant matches “{query}”.</div>
  {/if}
  {#if variant}
    <div class="row wrap">
      <label for="pick-field">Field of <span class="mono">{variant}</span></label>
      <select id="pick-field" bind:value={field}>
        <option value="">Choose a field…</option>
        {#each fields as f}<option value={f.key}>{f.key} · {f.type.type}{f.required ? ' · required' : ''}</option>{/each}
      </select>
      <button class="btn sm primary" disabled={!field} onclick={() => onPick({ variant, field })}>Use this field</button>
    </div>
  {/if}
</div>

<style>
  .picker { padding: var(--space-3); border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--surface-2); }
  .list { max-height: 200px; overflow: auto; display: flex; flex-direction: column; border: 1px solid var(--border); border-radius: var(--radius-sm); background: var(--surface); }
  .opt { text-align: left; border: none; background: none; padding: 5px 9px; cursor: pointer; color: var(--text); font-size: var(--fs-sm); }
  .opt:hover, .opt.sel { background: var(--accent-soft); }
  .bad { color: var(--danger); }
</style>
