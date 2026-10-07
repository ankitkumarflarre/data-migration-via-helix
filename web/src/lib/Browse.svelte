<script lang="ts">
  import { onMount, tick } from 'svelte';
  import { browse, fmtValue, type BrowseItem, type EntityField, type EntityInfo, type PolicyResult, type TableResult } from './api';
  import RecordView from './RecordView.svelte';

  type Mode = 'policy' | 'table';
  let mode = $state<Mode>('policy');

  // ---- by policy number ----
  let policyNumber = $state('');
  let pBusy = $state(false);
  let pError = $state('');
  let pResult = $state<PolicyResult | null>(null);
  let collapsed = $state<Record<string, boolean>>({});
  let highlight = $state('');

  const known = $derived(new Set(pResult?.tables.flatMap((t) => t.records.map((r) => r.id)) ?? []));

  async function fetchPolicy() {
    if (!policyNumber.trim()) return;
    pBusy = true; pError = ''; pResult = null; collapsed = {};
    try { pResult = await browse.policy(policyNumber.trim()); }
    catch (e) { pError = (e as Error).message; }
    finally { pBusy = false; }
  }

  async function jump(id: string) {
    const t = pResult?.tables.find((t) => t.records.some((r) => r.id === id));
    if (t) collapsed[t.variant] = false;
    highlight = id;
    await tick();
    document.getElementById(`rec-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'center' });
    setTimeout(() => { if (highlight === id) highlight = ''; }, 2000);
  }

  const linkClass: Record<string, string> = { key: 'sheet', ledger: 'generated', references: 'reference', referenced_by: 'neutral' };
  const linkIcon: Record<string, string> = { key: '🔑', ledger: '📒', references: '→', referenced_by: '←' };

  // ---- by table ----
  let entities = $state<EntityInfo[]>([]);
  let entQuery = $state('');
  let entity = $state('');
  let entOpen = $state(false);
  let fields = $state<EntityField[]>([]);
  let field = $state('');
  let value = $state('');
  let tBusy = $state(false);
  let tError = $state('');
  let tResult = $state<TableResult | null>(null);
  let rows = $state<BrowseItem[]>([]);
  let selected = $state<BrowseItem | null>(null);

  onMount(async () => {
    try { entities = await browse.entities(); } catch (e) { tError = (e as Error).message; }
  });

  const entMatches = $derived(
    entQuery.trim() === '' ? entities.slice(0, 50) :
      entities.filter((e) => e.entity.includes(entQuery.trim().toLowerCase()) || e.title.toLowerCase().includes(entQuery.trim().toLowerCase())).slice(0, 50),
  );
  const fieldInfo = $derived(fields.find((f) => f.key === field));

  async function chooseEntity(e: string) {
    entity = e; entQuery = e; entOpen = false; field = ''; value = ''; fields = []; tResult = null; rows = []; selected = null;
    try { fields = await browse.entityFields(e); } catch (err) { tError = (err as Error).message; }
  }

  async function fetchTable(more = false) {
    if (!entity) return;
    tBusy = true; tError = '';
    try {
      const res = await browse.table(entity, field, value, more ? tResult?.next_cursor ?? '' : '');
      rows = more ? [...rows, ...res.records] : res.records;
      const cols = more && tResult ? Array.from(new Set([...tResult.columns, ...res.columns])) : res.columns;
      tResult = { ...res, columns: cols };
      if (!more) selected = null;
    } catch (e) { tError = (e as Error).message; }
    finally { tBusy = false; }
  }

  const shown = $derived((tResult?.columns ?? []).slice(0, 8));
</script>

<section class="stack gap">
  <div class="row wrap">
    <h1>Browse Helix data</h1>
    <span class="spacer"></span>
    <div class="seg" role="tablist" aria-label="Browse mode">
      <button role="tab" aria-selected={mode === 'policy'} class:on={mode === 'policy'} onclick={() => (mode = 'policy')}>By policy number</button>
      <button role="tab" aria-selected={mode === 'table'} class:on={mode === 'table'} onclick={() => (mode = 'table')}>By table</button>
    </div>
  </div>

  {#if mode === 'policy'}
    <form class="card pad row wrap" onsubmit={(e) => { e.preventDefault(); fetchPolicy(); }}>
      <label for="pn">Policy number</label>
      <input id="pn" type="text" bind:value={policyNumber} placeholder="e.g. TEST0001" autocomplete="off" style="flex: 1; min-width: 200px" />
      <button class="btn primary" type="submit" disabled={pBusy || !policyNumber.trim()}>{pBusy ? 'Fetching…' : 'Fetch all linked records'}</button>
      <p class="xs muted full">Finds the policy, the records the migrator wrote for it (ledger), what it references (party, product, issuer…) and every record that references it, following links up to 4 levels.</p>
    </form>

    {#if pError}<div class="err" role="alert">✕ {pError}</div>{/if}
    {#if pBusy}<div class="card pad muted">Following references in Helix…</div>{/if}

    {#if pResult}
      <div class="summary row wrap">
        <strong>{pResult.records}</strong> records in <strong>{pResult.tables.length}</strong> tables
        <span class="xs muted">· {pResult.queries} Helix queries · {(pResult.elapsed_ms / 1000).toFixed(1)} s</span>
        <span class="spacer"></span>
        {#if pResult.tables.length}
          <button class="btn sm ghost" onclick={() => (collapsed = Object.fromEntries(pResult!.tables.map((t) => [t.variant, true])))}>Collapse all</button>
          <button class="btn sm ghost" onclick={() => (collapsed = {})}>Expand all</button>
        {/if}
      </div>
      {#each pResult.notes as n}<div class="note">ⓘ {n}</div>{/each}

      {#each pResult.tables as t (t.variant)}
        <article class="card">
          <button class="thead" aria-expanded={!collapsed[t.variant]} onclick={() => (collapsed[t.variant] = !collapsed[t.variant])}>
            <span class="chev" aria-hidden="true">{collapsed[t.variant] ? '▸' : '▾'}</span>
            <span class="mono strong">{t.variant}</span>
            <span class="badge neutral">{t.entity}</span>
            <span class="spacer"></span>
            <span class="xs muted">{t.records.length} record{t.records.length === 1 ? '' : 's'}</span>
          </button>
          {#if !collapsed[t.variant]}
            {#each t.records as it (it.id)}
              <div class="rec" id="rec-{it.id}" class:hl={highlight === it.id}>
                <div class="row wrap links">
                  {#each it.links ?? [] as l}
                    <span class="badge {linkClass[l.kind]}" title={l.from ?? ''}>{linkIcon[l.kind]} {l.text}</span>
                  {/each}
                </div>
                <RecordView item={it} {known} onJump={jump} />
              </div>
            {/each}
          {/if}
        </article>
      {/each}
    {/if}
  {:else}
    <form class="card pad stack" onsubmit={(e) => { e.preventDefault(); fetchTable(); }}>
      <div class="filters">
        <div class="stack tight combo">
          <label for="ent">Table (entity)</label>
          <input id="ent" type="search" bind:value={entQuery} placeholder={entities.length ? `Search ${entities.length} tables…` : 'Loading tables…'}
            autocomplete="off" onfocus={() => (entOpen = true)} oninput={() => { entOpen = true; entity = ''; }}
            onblur={() => setTimeout(() => (entOpen = false), 150)} role="combobox" aria-expanded={entOpen} aria-controls="ent-list" />
          {#if entOpen && entMatches.length}
            <div class="list" id="ent-list" role="listbox">
              {#each entMatches as e}
                <button type="button" role="option" aria-selected={e.entity === entity} class="opt" onmousedown={() => chooseEntity(e.entity)}>
                  <span class="mono">{e.entity}</span> <span class="xs faint">{e.module} · {e.leaves.length} variant{e.leaves.length === 1 ? '' : 's'}</span>
                </button>
              {/each}
            </div>
          {/if}
        </div>
        <div class="stack tight">
          <label for="col">Column <span class="faint">(optional)</span></label>
          <select id="col" bind:value={field} disabled={!entity}>
            <option value="">— any —</option>
            {#if entity}<option value="{entity}_id">{entity}_id · record id</option>{/if}
            {#each fields as f}<option value={f.key}>{f.key} · {f.type.type}</option>{/each}
          </select>
        </div>
        <div class="stack tight">
          <label for="val">Value <span class="faint">(optional)</span></label>
          {#if fieldInfo?.type.enum?.length}
            <select id="val" bind:value={value}><option value="">— any —</option>{#each fieldInfo.type.enum as e}<option value={e}>{e}</option>{/each}</select>
          {:else}
            <input id="val" type="text" bind:value={value} disabled={!field} placeholder={field ? 'equals…' : 'choose a column first'} />
          {/if}
        </div>
        <button class="btn primary end" type="submit" disabled={!entity || tBusy}>{tBusy ? 'Fetching…' : 'Fetch records'}</button>
      </div>
    </form>

    {#if tError}<div class="err" role="alert">✕ {tError}</div>{/if}

    {#if tResult}
      <div class="summary row wrap">
        <strong>{rows.length}</strong> record{rows.length === 1 ? '' : 's'} from <span class="mono">{tResult.entity}</span>
        {#if tResult.where}<span class="xs muted">where <code>{tResult.where}</code></span>{/if}
        {#if tResult.columns.length > shown.length}<span class="xs muted">· showing {shown.length} of {tResult.columns.length} columns; click a row for all</span>{/if}
      </div>
      {#if rows.length === 0}
        <div class="card pad muted">No records match.</div>
      {:else}
        <div class="split">
          <div class="table-wrap">
            <table>
              <thead><tr><th>id</th><th>variant</th><th>v</th>{#each shown as c}<th>{c}</th>{/each}</tr></thead>
              <tbody>
                {#each rows as it (it.id)}
                  <tr class="click" class:sel={selected?.id === it.id} onclick={() => (selected = it)}>
                    <td class="mono xs" title={it.id}>{it.id.slice(0, 8)}…</td>
                    <td class="mono xs">{it.variant}</td>
                    <td class="xs">{it.version}</td>
                    {#each shown as c}<td class="xs cell">{fmtValue(it.fields[c])}</td>{/each}
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
          {#if selected}
            <aside class="card pad detail">
              <div class="row"><strong class="mono small">{selected.variant}</strong><span class="spacer"></span>
                <button class="btn sm ghost" onclick={() => (selected = null)} aria-label="Close details">✕</button></div>
              <RecordView item={selected} />
            </aside>
          {/if}
        </div>
        {#if tResult.next_cursor}
          <div><button class="btn" disabled={tBusy} onclick={() => fetchTable(true)}>{tBusy ? 'Loading…' : 'Load more'}</button></div>
        {/if}
      {/if}
    {/if}
  {/if}
</section>

<style>
  .gap { gap: var(--space-4); }
  .pad { padding: var(--space-4) var(--space-5); }
  .spacer { flex: 1; }
  .tight { gap: 4px; }
  .full { flex-basis: 100%; }
  .strong { font-weight: 600; }
  .seg { display: inline-flex; border: 1px solid var(--border-strong); border-radius: var(--radius-sm); overflow: hidden; }
  .seg button { border: none; background: var(--surface); color: var(--text-muted); padding: 7px 14px; font: 500 var(--fs) var(--font); cursor: pointer; }
  .seg button + button { border-left: 1px solid var(--border-strong); }
  .seg button.on { background: var(--accent); color: var(--accent-text); }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-3); border-radius: var(--radius-sm); }
  .note { background: var(--info-soft); color: var(--info); padding: var(--space-2) var(--space-3); border-radius: var(--radius-sm); font-size: var(--fs-sm); }
  .summary { gap: var(--space-2); }
  .thead {
    display: flex; align-items: center; gap: var(--space-3); width: 100%; padding: var(--space-3) var(--space-4);
    border: none; background: var(--surface); color: var(--text); cursor: pointer; font: inherit; text-align: left; flex-wrap: wrap;
    border-radius: var(--radius) var(--radius) 0 0;
  }
  .thead:hover { background: var(--surface-2); }
  .chev { color: var(--text-faint); width: 12px; }
  .rec { padding: var(--space-3) var(--space-4); border-top: 1px solid var(--border); transition: background 0.4s; }
  .rec.hl { background: var(--accent-soft); }
  .links { gap: 6px; margin-bottom: var(--space-2); }
  .filters { display: grid; grid-template-columns: minmax(220px, 2fr) minmax(180px, 1.5fr) minmax(160px, 1.5fr) auto; gap: var(--space-3); align-items: end; }
  .combo { position: relative; }
  .list {
    position: absolute; top: 100%; left: 0; right: 0; z-index: 5; max-height: 280px; overflow: auto; margin-top: 4px;
    background: var(--surface); border: 1px solid var(--border-strong); border-radius: var(--radius-sm); box-shadow: var(--shadow-lg);
  }
  .opt { display: block; width: 100%; text-align: left; border: none; background: none; color: var(--text); padding: 6px 10px; cursor: pointer; font-size: var(--fs-sm); }
  .opt:hover { background: var(--accent-soft); }
  .end { height: 34px; }
  .split { display: grid; grid-template-columns: minmax(0, 1fr); gap: var(--space-4); }
  .split:has(.detail) { grid-template-columns: minmax(0, 1.4fr) minmax(300px, 1fr); }
  .detail { position: sticky; top: 140px; max-height: 70vh; overflow: auto; }
  tr.click { cursor: pointer; }
  tr.sel td { background: var(--accent-soft) !important; }
  .cell { max-width: 220px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  @media (max-width: 900px) {
    .filters { grid-template-columns: 1fr; }
    .split:has(.detail) { grid-template-columns: 1fr; }
    .detail { position: static; }
  }
</style>
