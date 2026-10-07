<script lang="ts">
  import { fmtValue, type BrowseItem } from './api';

  let { item, known = new Set<string>(), onJump }: {
    item: BrowseItem;
    known?: Set<string>;
    onJump?: (id: string) => void;
  } = $props();

  let showEmpty = $state(false);
  const entries = $derived(
    Object.entries(item.fields)
      .filter(([, v]) => showEmpty || (v !== null && v !== undefined && v !== ''))
      .sort(([a], [b]) => a.localeCompare(b)),
  );
  const emptyCount = $derived(Object.values(item.fields).filter((v) => v === null || v === undefined || v === '').length);
</script>

<div class="rv stack">
  <div class="meta row wrap xs muted">
    <span>id <code title={item.id}>{item.id}</code></span>
    <span>version {item.version}</span>
    {#if item.updated_at}<span>updated {item.updated_at.replace('T', ' ').slice(0, 19)}</span>{/if}
    <span class="spacer"></span>
    {#if emptyCount}
      <label class="row chk"><input type="checkbox" bind:checked={showEmpty} /> show {emptyCount} empty</label>
    {/if}
  </div>
  <dl class="grid">
    {#each entries as [k, v]}
      {@const s = fmtValue(v)}
      <dt class="mono">{k}{#if item.generated?.includes(k)} <span class="badge generated" title="Generated placeholder (D6)">gen</span>{/if}</dt>
      <dd class="mono">
        {#if typeof v === 'string' && known.has(v) && onJump}
          <button class="jump" onclick={() => onJump(v)} title="Show the referenced record">→ {s.slice(0, 8)}…</button>
        {:else if s === ''}<span class="faint">—</span>
        {:else}{s}{/if}
      </dd>
    {:else}
      <dt class="faint">No values</dt><dd></dd>
    {/each}
  </dl>
</div>

<style>
  .rv { gap: var(--space-2); }
  .spacer { flex: 1; }
  .chk { gap: 4px; }
  .grid {
    display: grid; grid-template-columns: minmax(160px, max-content) 1fr; gap: 2px var(--space-4);
    margin: 0; font-size: var(--fs-sm);
  }
  dt { color: var(--text-muted); }
  dd { margin: 0; word-break: break-word; }
  .jump {
    border: none; background: var(--info-soft); color: var(--info); border-radius: 4px; padding: 0 6px;
    font: inherit; cursor: pointer;
  }
  .jump:hover { text-decoration: underline; }
  @media (max-width: 640px) { .grid { grid-template-columns: 1fr; } dd { margin-bottom: 6px; } }
</style>
