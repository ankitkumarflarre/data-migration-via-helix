<script lang="ts">
  import { onMount } from 'svelte';
  import { api, rulesApi, type Change, type ExcludedColumn, type FieldSource, type RuleInput, type RuleSetInfo, type RuleSetView, type RuleView, type TemplateFieldView } from '../api';
  import RuleForm from './RuleForm.svelte';
  import SourceForm from './SourceForm.svelte';
  import ReasonDialog from './ReasonDialog.svelte';
  import { colLabel, humanize, isDuckCreekCopy, q, recordName, sourceText, tableName, targetText, transformText, typeText, when, whenText } from './labels';

  type Tab = 'columns' | 'values' | 'excluded' | 'history';
  type Filter = '' | 'attention' | 'changed' | 'conditional' | 'off' | 'outside' | 'copy';
  type Group = 'column' | 'table';

  let sets = $state<RuleSetInfo[]>([]);
  let setName = $state('');
  let view = $state<RuleSetView | null>(null);
  let error = $state('');
  let loading = $state(true);
  let tab = $state<Tab>('columns');
  let search = $state('');
  let filter = $state<Filter>('');
  let group = $state<Group>('column');
  let editing = $state<string | null>(null); // rule id | "new" | "tmpl:record id|field"
  let prefill = $state<{ column: string; header: string } | null>(null);
  let open = $state<Record<string, boolean>>({});
  let flash = $state('');
  let dialog = $state<{ title: string; detail?: string; action: string; run: (reason: string) => Promise<void> } | null>(null);

  onMount(async () => {
    try {
      sets = await api.ruleSets();
      setName = sets[0]?.name ?? '';
      if (setName) view = await rulesApi.get(setName);
    } catch (e) { error = (e as Error).message; }
    finally { loading = false; }
  });

  const headers = $derived(view?.headers ?? {});
  const colNum = (c: string) => [...c].reduce((n, ch) => n * 26 + ch.charCodeAt(0) - 64, 0);
  const ruleTitle = (id: string) => { const r = view?.rules.find((x) => x.id === id); return r ? `${r.header} (column ${r.excel_column})` : id; };

  function matches(r: RuleView): boolean {
    const s = search.trim().toLowerCase();
    if (s && ![r.id, r.excel_column, r.header, r.target.variant, r.target.instance ?? '', r.target.field, recordName(r.target.variant, r.target.instance), humanize(r.target.field), r.note ?? '', r.attention ?? '']
      .some((x) => x.toLowerCase().includes(s))) return false;
    switch (filter) {
      case 'attention': return !!r.attention || !!r.target_error;
      case 'changed': return r.status !== 'reviewed';
      case 'conditional': return !!r.when;
      case 'off': return !!r.disabled;
      case 'outside': return !!r.target_basis;
      case 'copy': return isDuckCreekCopy(r.target.variant);
    }
    return true;
  }

  const shown = $derived((view?.rules ?? []).filter(matches).sort((a, b) => colNum(a.excel_column) - colNum(b.excel_column) || a.id.localeCompare(b.id)));
  const groups = $derived.by(() => {
    if (group === 'column') return [{ title: '', rules: shown }];
    const by = new Map<string, RuleView[]>();
    for (const r of shown) by.set(r.target.variant, [...(by.get(r.target.variant) ?? []), r]);
    return [...by.entries()].sort(([a], [b]) => tableName(a).localeCompare(tableName(b))).map(([v, rules]) => ({ title: v, rules }));
  });
  const counts = $derived({
    rules: view?.rules.length ?? 0,
    changed: view?.rules.filter((r) => r.status !== 'reviewed').length ?? 0,
    off: view?.rules.filter((r) => r.disabled).length ?? 0,
    attention: view?.rules.filter((r) => r.attention || r.target_error).length ?? 0,
    values: view?.templates.reduce((n, t) => n + t.fields.length, 0) ?? 0,
  });

  function done(v: RuleSetView, msg: string) {
    view = v; editing = null; prefill = null; flash = msg;
    setTimeout(() => { if (flash === msg) flash = ''; }, 4000);
  }

  async function saveRule(r: RuleView, input: RuleInput, reason: string) {
    done(await rulesApi.put(setName, r.id, input, reason), `Saved ${r.id}. New uploads use it now.`);
  }
  async function addRule(input: RuleInput, reason: string) {
    const v = await rulesApi.add(setName, input, reason);
    done(v, `Added a rule for column ${input.excel_column}.`);
  }
  async function saveSource(id: string, variant: string, instance: string | undefined, f: TemplateFieldView, src: FieldSource, reason: string) {
    done(await rulesApi.putTemplate(setName, id, f.field, src, reason), `Saved ${recordName(variant, instance)} › ${humanize(f.field)}.`);
  }

  function backToReviewed(r: RuleView) {
    dialog = {
      title: r.status === 'added' ? `Remove rule ${r.id}?` : `Return ${r.id} to the reviewed rule?`,
      detail: r.status === 'added' ? `${r.header} (column ${r.excel_column}) will no longer be written.` : 'Your edits stay in the history and can be restored.',
      action: r.status === 'added' ? 'Remove rule' : 'Return to reviewed',
      run: async (reason) => done(await rulesApi.revert(setName, 'rule', r.id, 0, reason), r.status === 'added' ? `Removed ${r.id}.` : `${r.id} is back to the reviewed rule.`),
    };
  }
  function templateBack(id: string, variant: string, f: TemplateFieldView) {
    dialog = {
      title: `Return ${humanize(f.field)} to the reviewed value?`, action: 'Return to reviewed',
      run: async (reason) => done(await rulesApi.revert(setName, 'template', `${id}|${f.field}`, 0, reason), `${humanize(f.field)} is back to the reviewed value.`),
    };
  }
  function restore(c: Change) {
    dialog = {
      title: `Restore version #${c.seq}?`, detail: `${changeTitle(c)} goes back to how it was after this change.`, action: 'Restore',
      run: async (reason) => done(await rulesApi.revert(setName, c.kind, c.key, c.seq, reason), `Restored version #${c.seq}.`),
    };
  }

  function changeTitle(c: Change): string {
    if (c.kind === 'rule') return `Rule ${c.key} · ${c.rule ? `${c.rule.header} (column ${c.rule.excel_column})` : ruleTitle(c.key)}`;
    const [id, f] = [c.key.slice(0, c.key.lastIndexOf('|')), c.key.slice(c.key.lastIndexOf('|') + 1)];
    const [v, inst] = id.split('#');
    return `${recordName(v, inst)} › ${humanize(f)}`;
  }
  function changeText(c: Change): string {
    if (c.kind === 'template') return c.source ? sourceText(c.source, headers) : 'Back to the reviewed value';
    if (!c.rule) return view?.rules.some((r) => r.id === c.key) ? 'Back to the reviewed rule' : 'Rule removed';
    const r = c.rule;
    return [r.disabled ? `Switched off: ${r.disabled}` : `→ ${targetText(r.target)}`, whenText(r.when, headers), transformText(r.transform).join(' ')].filter(Boolean).join(' · ');
  }

  function addFor(e: ExcludedColumn) {
    prefill = { column: e.excel_column, header: e.header };
    editing = 'new'; tab = 'columns';
  }
</script>

<section class="stack gap">
  <header class="card head">
    <div class="stack tight">
      <h1>Mapping rules</h1>
      <p class="small muted">How each column of the rater's <strong>{view?.sheet ?? 'Policy Data'}</strong> sheet is written to Helix. Changes apply to new uploads straight away and are kept with a reason and history.</p>
      {#if view}
        <div class="row wrap xs muted">
          {#if sets.length > 1}
            <label for="ruleset">Rule set</label>
            <select id="ruleset" bind:value={setName} onchange={async () => { view = await rulesApi.get(setName); }}>
              {#each sets as s}<option value={s.name}>{s.name}</option>{/each}
            </select>
          {:else}<span class="mono">{view.name}</span>{/if}
          <span>Source report: {view.source_report}</span>
          {#if view.sha !== view.reviewed_sha}<span class="badge warn">✎ Edited since the reviewed version</span>{:else}<span class="badge ok">✓ Reviewed version</span>{/if}
        </div>
      {/if}
    </div>
    {#if view}
      <div class="stats">
        <div><b>{counts.rules}</b><span>column rules</span></div>
        <div><b>{counts.changed}</b><span>edited or added</span></div>
        <div><b>{counts.attention}</b><span>flagged</span></div>
        <div><b>{counts.off}</b><span>switched off</span></div>
      </div>
    {/if}
  </header>

  {#if loading}<p class="muted">Loading rules…</p>{/if}
  {#if error}<div class="err" role="alert">✕ {error}</div>{/if}
  {#if flash}<div class="ok-note" role="status">✓ {flash}</div>{/if}

  {#if view}
    <div class="tabs" role="tablist">
      <button role="tab" aria-selected={tab === 'columns'} class:on={tab === 'columns'} onclick={() => (tab = 'columns')}>Sheet columns ({counts.rules})</button>
      <button role="tab" aria-selected={tab === 'values'} class:on={tab === 'values'} onclick={() => (tab = 'values')}>Fixed & generated values ({counts.values})</button>
      <button role="tab" aria-selected={tab === 'excluded'} class:on={tab === 'excluded'} onclick={() => (tab = 'excluded')}>Excluded columns ({view.excluded.length})</button>
      <button role="tab" aria-selected={tab === 'history'} class:on={tab === 'history'} onclick={() => (tab = 'history')}>History ({view.changes.length})</button>
      <span class="spacer"></span>
      <a class="btn sm ghost" href={rulesApi.exportUrl(setName, 'pins')} download title="Pins file with every edit, to commit and regenerate with make rules">⤓ Export pins</a>
      <a class="btn sm ghost" href={rulesApi.exportUrl(setName, 'templates')} download title="Templates file with every edit">⤓ Export templates</a>
    </div>

    {#if tab === 'columns'}
      <div class="row wrap toolbar">
        <input type="search" placeholder="Search column, header or Helix field" bind:value={search} aria-label="Search rules" style="flex:1; min-width: 220px" />
        <label for="flt">Show</label>
        <select id="flt" bind:value={filter}>
          <option value="">All rules</option>
          <option value="attention">Flagged for approval</option>
          <option value="changed">Edited or added here</option>
          <option value="conditional">Only for some rows</option>
          <option value="off">Switched off</option>
          <option value="outside">Target not in the report</option>
          <option value="copy">Written to a Duck Creek copy</option>
        </select>
        <label for="grp">Group by</label>
        <select id="grp" bind:value={group}><option value="column">Sheet column</option><option value="table">Helix table</option></select>
        <button class="btn sm primary" onclick={() => { prefill = null; editing = editing === 'new' ? null : 'new'; }}>+ Add rule</button>
      </div>

      {#if editing === 'new'}
        <article class="card">
          <div class="cardhead"><strong>New rule</strong><span class="xs muted">for a column without one, e.g. from the under-review analysis</span></div>
          <RuleForm {prefill} {headers} onSave={addRule} onCancel={() => { editing = null; prefill = null; }} />
        </article>
      {/if}

      {#if !shown.length}<p class="muted">No rule matches.</p>{/if}

      {#each groups as g (g.title)}
        {#if g.title}
          <h2 class="ghead">{tableName(g.title)} <span class="mono xs faint">{g.title}</span>{#if isDuckCreekCopy(g.title)}<span class="badge neutral" title="Copies the Duck Creek XML layout; a Helix-native home is preferred (S16)">Duck Creek copy</span>{/if}</h2>
        {/if}
        {#each g.rules as r (r.id)}
          <article class="card rule" class:off={!!r.disabled}>
            <div class="cardhead">
              <span class="col mono" title="Sheet column">{r.excel_column}</span>
              <div class="stack tight grow">
                <div class="row wrap sentence">
                  <strong>{r.header}</strong>
                  <span aria-hidden="true" class="muted">→</span>
                  {#if r.disabled}<span class="muted">not written</span>{:else}<span>{recordName(r.target.variant, r.target.instance)} › <strong>{humanize(r.target.field)}</strong></span>{/if}
                  {#if r.target_type && !r.disabled}<span class="xs muted">({typeText(r.target_type)})</span>{/if}
                </div>
                <div class="row wrap badges">
                  {#if r.status === 'edited'}<span class="badge override">✎ Edited</span>{/if}
                  {#if r.status === 'added'}<span class="badge override">＋ Added here</span>{/if}
                  {#if r.disabled}<span class="badge neutral">⏸ Switched off</span>{/if}
                  {#if r.when}<span class="badge reference">◐ Some rows only</span>{/if}
                  {#if r.attention}<span class="badge warn">⚠ Flagged</span>{/if}
                  {#if r.target_basis}<span class="badge neutral" title={r.target_basis}>Not in the report</span>{/if}
                  {#if isDuckCreekCopy(r.target.variant) && !r.disabled}<span class="badge neutral" title="Copies the Duck Creek XML layout (S16)">Duck Creek copy</span>{/if}
                  {#if r.target_error}<span class="badge bad">✕ {r.target_error}</span>{/if}
                  <span class="xs faint mono">{r.id}</span>
                </div>
              </div>
              <div class="row actions">
                {#if r.status !== 'reviewed'}<button class="btn sm ghost" onclick={() => backToReviewed(r)}>{r.status === 'added' ? 'Remove' : 'Undo edits'}</button>{/if}
                <button class="btn sm" aria-expanded={editing === r.id} onclick={() => (editing = editing === r.id ? null : r.id)}>{editing === r.id ? 'Close' : 'Edit'}</button>
              </div>
            </div>

            <ul class="facts small">
              {#if r.disabled}<li><span class="k">Off</span>{r.disabled}</li>{/if}
              {#if r.when}<li><span class="k">Rows</span>{whenText(r.when, headers)}</li>{/if}
              {#if !r.disabled}<li><span class="k">Value</span>{transformText(r.transform).join(' ')}</li>{/if}
              {#if r.attention}<li class="warnc"><span class="k">⚠ Flag</span>{r.attention}</li>{/if}
              {#if r.target_basis}<li><span class="k">Why here</span>{r.target_basis}</li>{/if}
              {#if r.note}<li><span class="k">Note</span>{r.note}</li>{/if}
              {#if r.reviewed}<li class="muted"><span class="k">Reviewed</span>{r.reviewed.disabled ? `switched off` : targetText(r.reviewed.target)}{#if r.reviewed.when} · {whenText(r.reviewed.when, headers).toLowerCase()}{/if} · {transformText(r.reviewed.transform).join(' ')}</li>{/if}
            </ul>

            <details bind:open={open[r.id]} class="tech xs">
              <summary class="muted">Technical details</summary>
              <dl>
                <dt>Helix field</dt><dd class="mono">{r.target.variant}{r.target.instance ? `#${r.target.instance}` : ''}.{r.target.field}{r.target_required ? ' (required)' : ''}</dd>
                <dt>Mapping report</dt><dd>{r.report_status}{r.confidence ? ` · confidence ${r.confidence}` : ''}{r.report_no ? ` · row #${r.report_no}` : ''}</dd>
                {#if r.alternatives.length}<dt>Other fields the report names</dt><dd class="mono">{r.alternatives.map((a) => `${a.variant}.${a.field}`).join('\n')}</dd>{/if}
                {#if r.report_locations.length}<dt>Report locations</dt><dd class="mono">{r.report_locations.join('\n')}</dd>{/if}
                {#if r.changes}<dt>History</dt><dd>{r.changes} change{r.changes === 1 ? '' : 's'} · <button class="linkbtn" onclick={() => { tab = 'history'; search = r.id; }}>show</button></dd>{/if}
              </dl>
            </details>

            {#if editing === r.id}
              <RuleForm rule={r} {headers} onSave={(input, reason) => saveRule(r, input, reason)} onCancel={() => (editing = null)} />
            {/if}
          </article>
        {/each}
      {/each}

    {:else if tab === 'values'}
      <p class="small muted">Records and fields the migrator writes besides the sheet columns: fixed values, values built from a pattern, links between records and generated placeholders for required fields the sheet does not have.</p>
      {#each view.templates.filter((t) => t.fields.length) as t (t.id)}
        <article class="card">
          <div class="cardhead">
            <div class="stack tight grow">
              <strong>{recordName(t.variant, t.instance)}</strong>
              <span class="xs muted">{t.scope === 'job' ? 'Written once per upload and shared by every row' : 'One per sheet row'}{t.key ? ` · found by ${humanize(t.key).toLowerCase()}` : ''} · <span class="mono">{t.variant}</span></span>
            </div>
          </div>
          <div class="table-wrap flat"><table>
            <thead><tr><th>Field</th><th>Value</th><th></th></tr></thead>
            <tbody>
              {#each t.fields as f (f.field)}
                {@const k = `tmpl:${t.id}|${f.field}`}
                <tr>
                  <td><span>{humanize(f.field)}</span>{#if f.required}<span class="req" title="Required by Helix">*</span>{/if}<div class="xs faint mono">{f.field}{f.type ? ` · ${typeText(f.type)}` : ''}</div></td>
                  <td>
                    <div class="row wrap tight">
                      {sourceText(f.source, headers)}
                      {#if f.source.generate}<span class="badge generated">Placeholder</span>{/if}
                      {#if f.status !== 'reviewed'}<span class="badge override">✎ Edited</span>{/if}
                    </div>
                    {#if f.source.attention}<div class="xs warnc">⚠ {f.source.attention}</div>{/if}
                    {#if f.reviewed}<div class="xs muted">Reviewed: {sourceText(f.reviewed, headers)}</div>{/if}
                  </td>
                  <td class="nowrap">
                    {#if f.status !== 'reviewed'}<button class="btn sm ghost" onclick={() => templateBack(t.id, t.variant, f)}>Undo</button>{/if}
                    {#if f.editable}<button class="btn sm" onclick={() => (editing = editing === k ? null : k)}>{editing === k ? 'Close' : 'Edit'}</button>{:else}<span class="xs faint" title="Links between records are part of the model">fixed link</span>{/if}
                  </td>
                </tr>
                {#if editing === k}
                  <tr class="formrow"><td colspan="3"><SourceForm field={f} {headers} onSave={(src, reason) => saveSource(t.id, t.variant, t.instance, f, src, reason)} onCancel={() => (editing = null)} /></td></tr>
                {/if}
              {/each}
            </tbody>
          </table></div>
        </article>
      {/each}

    {:else if tab === 'excluded'}
      <p class="small muted">Columns the mapping report confirmed but that are deliberately not written.</p>
      {#each view.excluded as e (e.excel_column)}
        <article class="card cardhead">
          <span class="col mono">{e.excel_column}</span>
          <div class="stack tight grow"><strong>{e.header}</strong><span class="small muted">{e.reason}</span></div>
          <button class="btn sm" onclick={() => addFor(e)}>Add a rule for it…</button>
        </article>
      {:else}<p class="muted">No excluded columns.</p>{/each}

    {:else}
      <div class="row wrap toolbar">
        <input type="search" placeholder="Filter by rule id, column or field" bind:value={search} aria-label="Filter history" style="flex:1" />
      </div>
      {@const hist = view.changes.filter((c) => !search.trim() || `${c.key} ${changeTitle(c)} ${c.reason}`.toLowerCase().includes(search.trim().toLowerCase()))}
      {#each hist as c (c.seq)}
        <article class="card hist">
          <div class="row wrap">
            <span class="mono xs faint">#{c.seq}</span>
            <strong class="small">{changeTitle(c)}</strong>
            <span class="xs muted">{when(c.at)}</span>
            <span class="spacer"></span>
            <button class="btn sm ghost" onclick={() => restore(c)}>Restore this version</button>
          </div>
          <div class="small">{changeText(c)}</div>
          <div class="xs muted">Reason: {q(c.reason)}</div>
        </article>
      {:else}<p class="muted">{view.changes.length ? 'No change matches.' : 'No changes yet. Every edit made here is listed with its reason.'}</p>{/each}
    {/if}
  {/if}
</section>

{#if dialog}
  <ReasonDialog title={dialog.title} detail={dialog.detail} action={dialog.action} onConfirm={dialog.run} onClose={() => (dialog = null)} />
{/if}

<style>
  .gap { gap: var(--space-4); }
  .tight { gap: var(--space-1); }
  .grow { flex: 1; min-width: 0; }
  .head { display: flex; gap: var(--space-5); padding: var(--space-4) var(--space-5); align-items: center; flex-wrap: wrap; }
  .head > .stack { flex: 1; min-width: 260px; }
  h1 { margin: 0; font-size: var(--fs-xl); }
  .head p { margin: 0; }
  .stats { display: flex; gap: var(--space-5); }
  .stats div { display: flex; flex-direction: column; }
  .stats b { font-size: var(--fs-xl); font-variant-numeric: tabular-nums; }
  .stats span { font-size: var(--fs-xs); color: var(--text-muted); }
  .tabs { display: flex; gap: var(--space-1); border-bottom: 1px solid var(--border); align-items: center; flex-wrap: wrap; }
  .tabs button { border: none; background: none; padding: 8px 14px; font: 500 var(--fs) var(--font); color: var(--text-muted); cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -1px; }
  .tabs button.on { color: var(--text); border-bottom-color: var(--accent); }
  .toolbar { gap: var(--space-2); }
  .ghead { font-size: var(--fs); margin: var(--space-3) 0 0; display: flex; gap: var(--space-2); align-items: center; flex-wrap: wrap; }
  .rule.off { opacity: 0.75; }
  .cardhead { display: flex; gap: var(--space-3); align-items: flex-start; padding: var(--space-3) var(--space-4); }
  .col { min-width: 38px; height: 30px; padding: 0 6px; display: grid; place-items: center; background: var(--accent-soft); color: var(--accent); border-radius: var(--radius-sm); font-weight: 700; }
  .sentence { gap: var(--space-2); }
  .badges { gap: var(--space-1); }
  .actions { gap: var(--space-1); }
  .facts { list-style: none; margin: 0; padding: 0 var(--space-4) var(--space-2) calc(var(--space-4) + 38px + var(--space-3)); display: flex; flex-direction: column; gap: 3px; }
  .facts .k { display: inline-block; min-width: 76px; color: var(--text-muted); font-size: var(--fs-xs); text-transform: uppercase; letter-spacing: 0.03em; }
  .warnc { color: var(--warning); }
  .tech { padding: 0 var(--space-4) var(--space-3) calc(var(--space-4) + 38px + var(--space-3)); }
  .tech dl { display: grid; grid-template-columns: 180px 1fr; gap: 4px var(--space-3); margin: var(--space-2) 0 0; }
  .tech dt { color: var(--text-muted); }
  .tech dd { margin: 0; white-space: pre-wrap; word-break: break-word; }
  .linkbtn { border: none; background: none; color: var(--accent); cursor: pointer; padding: 0; font: inherit; }
  .flat { border: none; border-top: 1px solid var(--border); border-radius: 0; }
  .formrow td { padding: 0; }
  .formrow:hover td { background: none; }
  .req { color: var(--danger); margin-left: 2px; }
  .nowrap { white-space: nowrap; }
  .hist { padding: var(--space-3) var(--space-4); display: flex; flex-direction: column; gap: var(--space-1); }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-3); border-radius: var(--radius-sm); }
  .ok-note { background: var(--success-soft); color: var(--success); padding: var(--space-2) var(--space-3); border-radius: var(--radius-sm); }
  @media (max-width: 720px) {
    .cardhead { flex-wrap: wrap; }
    .facts, .tech { padding-left: var(--space-4); }
    .tech dl { grid-template-columns: 1fr; }
    .stats { gap: var(--space-4); }
  }
</style>
