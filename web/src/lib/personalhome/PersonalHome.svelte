<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type Job, type PlanRow } from '../api';
  import MappingRegister from './MappingRegister.svelte';
  import { catalog, flow, activeFlow, enabled, inventoryFor, defaultScenario, mappingFor, matchesField, targetType, popups, exportManifest, type Scenario } from './model';
  let { job = null }: { job?: Job | null } = $props();
  let scenario = $state<Scenario>({ ...defaultScenario });
  let selected = $state('newquote');
  let tab = $state('flow');
  let query = $state('');
  let fieldFilter = $state('all');
  let rows = $state<PlanRow[]>([]);
  let rowNumber = $state(0);
  let rowOffset = $state(0);
  let totalRows = $state(0);
  let loading = $state(false);
  let error = $state('');
  let request = 0;
  const active = $derived(activeFlow(scenario));
  const page = $derived(flow.find(p => p.id === selected) ?? flow[0]);
  const inventory = $derived(inventoryFor(page));
  const index = $derived(active.findIndex(p => p.id === selected));
  const effective = $derived(job?.plan.rules ?? []);
  const visibleFields = $derived((inventory?.fields ?? []).filter(f => matchesField(page.id, f, query, effective) && (fieldFilter === 'all' || (fieldFilter === 'required' && f.required.includes('Required')) || (fieldFilter === 'mapped' && mappingFor(page.id, f, effective).target) || (fieldFilter === 'unmapped' && !mappingFor(page.id, f, effective).target))));
  const groups = $derived([...new Set(visibleFields.map(f => f.group))]);
  const row = $derived(rows.find(r => r.row === rowNumber));
  const searchPages = $derived(query.trim() ? flow.filter(p => [p.title, p.vm].some(v => v.toLowerCase().includes(query.toLowerCase())) || inventoryFor(p)?.fields.some(f => matchesField(p.id, f, query, effective))) : []);

  function readHash() {
    const value = location.hash.split('/')[1]?.split('?')[0];
    if (value === 'mappings') tab = 'mappings';
    else if (flow.some(p => p.id === value)) { selected = value; tab = 'flow'; }
  }
  onMount(() => { readHash(); window.addEventListener('hashchange', readHash); return () => window.removeEventListener('hashchange', readHash); });
  function navigate(id: string) { selected = id; tab = 'flow'; location.hash = `reference/${id}${job ? `?job=${job.id}` : ''}`; }
  function mappings() { tab = 'mappings'; location.hash = `reference/mappings${job ? `?job=${job.id}` : ''}`; }
  function changeScenario(key: keyof Scenario, value: boolean) {
    scenario = { ...scenario, [key]: value };
    if (page.kind === 'conditional' && !enabled(page, scenario)) navigate('claimshistory');
  }
  const toggles: { key: keyof Scenario; label: string }[] = [
    {key:'clueSuccess',label:'CLUE succeeded'}, {key:'clueClaims',label:'CLUE has claims'},
    {key:'scoreAvailable',label:'Insurance score available'}, {key:'billing',label:'DCT Billing enabled'},
    {key:'inForce',label:'Policy in force'}, {key:'agreementPending',label:'Subscriber agreement pending'},
  ];
  $effect(() => {
    const id = job?.id; const hash = job?.plan.plan_hash; const offset = rowOffset;
    void hash;
    const version = ++request;
    rows = []; totalRows = 0; error = ''; loading = !!id;
    if (!id) return;
    api.rows(id, offset, 20).then(result => {
      if (version !== request) return;
      rows = result.items ?? []; totalRows = result.total; rowNumber = rows[0]?.row ?? 0;
    }).catch(e => { if (version === request) error = e instanceof Error ? e.message : String(e); }).finally(() => { if (version === request) loading = false; });
  });
  function download() {
    const blob = new Blob([JSON.stringify(exportManifest(scenario), null, 2)], {type:'application/json'});
    const url = URL.createObjectURL(blob); const a = document.createElement('a');
    a.href = url; a.download = 'personalhome-implementation-manifest.json'; a.click();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
</script>

<div class="workspace-header">
  <div><div class="eyebrow">MANATEE · FLORIDA · PERSONAL HOME</div><h1>PersonalHome page flow</h1><p class="muted">Explore the policy journey, field rules and entity mapping evidence.</p></div>
  <button class="btn" onclick={download}>Export implementation manifest</button>
</div>
<div class="overview">
  <div><b>{active.length}</b><span>pages in this scenario</span></div><div><b>170</b><span>documented fields</span></div><div><b>20 / 81</b><span>confirmed spreadsheet columns</span></div><div><b>2</b><span>suppressed pages</span></div>
</div>
<div class="notice">Flow and migration review workspace. Scenario switches preview navigation; they do not order reports, rate, collect payments or bind a policy.</div>
<div class="workspace">
  <aside>
    <label class="search-label">Search all pages and fields<input type="search" placeholder="Name, field or Helix target" bind:value={query} /></label>
    {#if query.trim()}
      <div class="eyebrow">SEARCH RESULTS · {searchPages.length}</div>
      {#each searchPages as result}<button class="page-link" class:chosen={selected === result.id} onclick={() => navigate(result.id)}>{result.title}</button>{/each}
      {#if !searchPages.length}<p class="muted">No matching pages or fields.</p>{/if}
      <button class="btn sm ghost" onclick={() => query = ''}>Clear search</button>
    {/if}
    <div class="eyebrow">POLICY JOURNEY</div>
    {#each active as node, i}
      <button class="page-link" class:chosen={tab === 'flow' && selected === node.id} aria-current={tab === 'flow' && selected === node.id ? 'step' : undefined} onclick={() => navigate(node.id)}><span class="step-num">{i+1}</span><span>{node.title}{#if node.kind === 'conditional'}<small>Conditional</small>{/if}</span></button>
    {/each}
    <details><summary>Inactive conditional pages</summary>{#each flow.filter(p => p.kind === 'conditional' && !enabled(p, scenario)) as node}<button class="page-link" onclick={() => navigate(node.id)}>{node.title}<small>{node.gate}</small></button>{/each}</details>
    <details><summary>Auxiliary &amp; suppressed</summary>{#each flow.filter(p => ['auxiliary','suppressed'].includes(p.kind)) as node}<button class="page-link" class:chosen={selected === node.id} onclick={() => navigate(node.id)}>{node.title}<small>{node.kind}</small></button>{/each}</details>
    <button class="page-link register" class:chosen={tab === 'mappings'} onclick={mappings}>Entity mapping register →</button>
  </aside>
  <div class="content">
    <details class="card scenario"><summary>Flow scenario <span class="muted">· preview conditional branches</span></summary><div class="toggles">{#each toggles as toggle}<label><input type="checkbox" checked={scenario[toggle.key]} onchange={e => changeScenario(toggle.key, e.currentTarget.checked)} />{toggle.label}</label>{/each}</div><p class="muted">Underwriting preview uses FL Select. Other product variants need their own field definitions.</p></details>
    {#if tab === 'mappings'}<MappingRegister />{:else}
      <section class="card page-heading">
        <div class="eyebrow">{page.kind} · {page.vm}</div><h2>{page.title}</h2><p>{page.summary}</p>
        {#if page.gate}<p class="condition">Condition: {page.gate}</p>{/if}
        {#if page.kind === 'conditional' && !enabled(page, scenario)}<p class="warning">This page is skipped in the selected scenario. Its documentation remains available here.</p>{/if}
        {#if page.kind === 'suppressed'}<p class="warning">Suppressed by PersonalHome. Excluded from Previous / Next navigation.</p>{/if}
        {#if page.gap}<p class="notice">Integration gap: {page.gap}</p>{/if}
        {#if page.id === 'summary' || page.id === 'commit'}<div class="readiness"><b>Bind readiness</b><p>{scenario.agreementPending ? 'Subscriber agreement is pending; Bind is blocked.' : 'Scenario marks the agreement acknowledged. Production must verify the acknowledgement and signature.'}</p><p>Rating, underwriting, billing and transaction services must confirm eligibility before binding.</p><button class="btn" disabled>Bind unavailable — integration required</button></div>{/if}
        {#if page.id === 'dwellinginfo'}<button class="btn sm" onclick={() => navigate('locationdetail')}>Open Location Detail →</button>{/if}
        {#if inventory}<details><summary>Source metadata</summary><dl><dt>Inventory ViewModel</dt><dd>{inventory.viewModel}</dd><dt>Model collection</dt><dd>{inventory.topic}</dd><dt>Manuscript</dt><dd>{inventory.manuscript}</dd><dt>Inheritance</dt><dd>{inventory.inheritance}</dd><dt>Source navigation</dt><dd>{inventory.navPrev || 'Entry'} → {inventory.navNext}</dd></dl><p class="muted">Source navigation is preserved for review; the journey uses narrative order and skips suppressed pages.</p></details>{/if}
      </section>
      {#if job}<section class="card preview"><h3>Current migration preview</h3><p class="muted">{job.file_name} · Effective mapping overrides apply. Values below are planned migration values, not entered quote data.</p>
        {#if loading}<p role="status">Loading planned rows…</p>{:else if error}<p role="alert" class="warning">{error}</p>{:else}<label>Policy row <select bind:value={rowNumber}>{#each rows as r}<option value={r.row}>Row {r.row} · {r.policy_number}{r.blocked ? ' · BLOCKED' : ''}</option>{/each}</select></label><div class="pager"><button class="btn sm" disabled={rowOffset === 0} onclick={() => rowOffset = Math.max(0,rowOffset-20)}>Previous rows</button><span>{totalRows ? rowOffset+1 : 0}–{Math.min(rowOffset+20,totalRows)} of {totalRows}</span><button class="btn sm" disabled={rowOffset+20 >= totalRows} onclick={() => rowOffset += 20}>Next rows</button></div>{/if}
      </section>{/if}
      <section class="card field-section">
        <div class="field-toolbar"><h3>Field inventory <span class="muted">{visibleFields.length} / {inventory?.fields.length ?? 0}</span></h3><label>Filter <select bind:value={fieldFilter}><option value="all">All fields</option><option value="required">Required / conditional</option><option value="mapped">Has correspondence</option><option value="unmapped">Unmapped</option></select></label></div>
        {#if !inventory}<p class="muted">The document describes this page but provides no detailed field inventory. No fields or mappings have been invented.</p>{:else if !visibleFields.length}<p>No fields match. Clear the search or change the filter.</p>{/if}
        {#each groups as group (page.id + group)}<h4>{group}</h4>{#each visibleFields.filter(f => f.group === group) as field (page.id + ':' + field.id)}
          {@const mapping = mappingFor(page.id, field, effective)}
          {@const values = mapping.target && !mapping.excluded ? row?.records.filter(r => r.variant === mapping.target?.variant).map(r => r.fields[mapping.target!.field]).filter(v => v !== undefined) ?? [] : []}
          <details class="field"><summary><span><b>{field.label}</b><small>{field.tech}</small></span><span class="field-flags"><span>{field.required}</span><span class:has-mapping={!!mapping.target}>{mapping.status}</span></span></summary>
            <dl><dt>Data path</dt><dd><code>{field.path}</code></dd><dt>Type / control</dt><dd>{field.type} · {field.control}</dd><dt>Read / write</dt><dd>{field.rw}</dd><dt>Visibility</dt><dd>{field.visibility} · {field.visCondition}</dd><dt>Enabled</dt><dd>{field.enabled} · {field.enabledCond}</dd><dt>Required</dt><dd>{field.required} · {field.reqCond}</dd><dt>Default</dt><dd>{field.defaultVal}</dd><dt>Validation</dt><dd>{field.validation}</dd><dt>Calculation</dt><dd>{field.calc}</dd><dt>Lookup</dt><dd>{field.lookup}</dd><dt>Origin</dt><dd>{field.pbLob} · {field.source}</dd></dl>
            <div class="mapping"><b>{mapping.status}</b><p>{mapping.note}</p>{#if mapping.target}<code>{mapping.target.variant}.{mapping.target.field}</code><p>Target type (offline schema): {targetType(mapping.target)}</p>{/if}
              {#if mapping.overridden}<p class="warning">Target or transform overridden in the current migration job.</p>{/if}
              {#if mapping.rule}<p>Rule {mapping.rule.id} · Spreadsheet column {mapping.rule.excel_column} · {mapping.rule.header}</p><p>Spreadsheet transform: <code>{JSON.stringify(mapping.transform)}</code></p>{#if 'attention' in mapping.rule}<p class="warning">{mapping.rule.attention}</p>{/if}{/if}
              {#if job && mapping.target}<p><b>Planned value:</b> {mapping.excluded ? 'Excluded from this job' : values.length ? values.map(v => JSON.stringify(v)).join(' · ') : 'Not present in this planned row'}</p>{/if}
            </div>
          </details>
        {/each}{/each}
      </section>
      {#if inventory?.dependencies.length}<section class="card dependencies"><h3>Documented dependencies</h3>{#each inventory.dependencies as dep}<div><b>{dep.type}</b><p><code>{dep.source}</code> → <code>{dep.target}</code></p><p>{dep.condition} · {dep.result}</p></div>{/each}<p class="muted">These are source rules for review, not executed calculations.</p></section>{/if}
      {#if popups.some(p => p.parent === page.id)}<section class="card popup-list"><h3>On-demand dialogs</h3>{#each popups.filter(p => p.parent === page.id) as popup}<div><h4>{popup.name}</h4><p>{popup.detail}</p></div>{/each}</section>{/if}
      <nav class="pager" aria-label="PersonalHome page navigation">
        {#if index >= 0}<button class="btn" disabled={index === 0} onclick={() => navigate(active[index-1].id)}>← Previous</button><span class="muted">{index+1} of {active.length}</span><button class="btn primary" disabled={index === active.length-1} onclick={() => navigate(active[index+1].id)}>Next →</button>{:else}<button class="btn" onclick={() => navigate(page.id === 'locationdetail' ? 'dwellinginfo' : 'newquote')}>Return to policy journey</button>{/if}
      </nav>
    {/if}
    <p class="source muted">Source: {catalog.source} · 9 inventories · SHA-256 <code>{catalog.sha256.slice(0,16)}…</code></p>
  </div>
</div>

<style>
  .workspace-header { display:flex; justify-content:space-between; align-items:center; gap:20px; margin-bottom:24px; }
  h1 { margin:6px 0; font-size:28px; } h2 { font-size:24px; margin:10px 0; } h3 { margin:0 0 12px; } h4 { margin:22px 0 10px; }
  p { line-height:1.6; } .eyebrow { font-size:11px; font-weight:700; letter-spacing:1px; color:var(--text-muted); }
  .overview { display:grid; grid-template-columns:repeat(4,1fr); gap:12px; margin-bottom:20px; }
  .overview>div { display:grid; gap:6px; padding:18px; background:var(--surface); border:1px solid var(--border); border-radius:10px; }
  .overview b { font-size:26px; } .overview span { font-size:12px; color:var(--text-muted); }
  .notice { padding:12px 16px; background:var(--info-soft); color:var(--info); border-radius:8px; margin-bottom:20px; line-height:1.6; }
  .workspace { display:grid; grid-template-columns:260px minmax(0,1fr); gap:24px; align-items:start; }
  aside { position:sticky; top:88px; max-height:calc(100vh - 110px); overflow-y:auto; background:var(--surface); border:1px solid var(--border); border-radius:12px; padding:16px; }
  aside .eyebrow { margin:24px 0 10px; } .search-label { display:grid; gap:8px; font-size:12px; font-weight:600; } input[type=search] { width:100%; min-width:0; }
  .page-link { display:flex; align-items:center; flex-wrap:wrap; gap:10px; width:100%; background:none; border:none; border-radius:7px; text-align:left; padding:10px 8px; color:var(--text-muted); cursor:pointer; }
  .page-link:hover { background:var(--surface-2); } .chosen { background:var(--accent-soft); color:var(--accent); font-weight:600; }
  .page-link small { display:block; font-size:10px; font-weight:400; } .step-num { display:grid; place-items:center; background:var(--surface-3); border-radius:50%; width:24px; height:24px; flex-shrink:0; font-size:11px; }
  .chosen .step-num { background:var(--accent); color:var(--accent-text); } aside details { margin-top:18px; font-size:12px; } .register { margin-top:20px; border-top:1px solid var(--border); }
  .card { padding:22px; margin-bottom:18px; border:1px solid var(--border); border-radius:12px; background:var(--surface); }
  summary { cursor:pointer; } .toggles { display:grid; grid-template-columns:1fr 1fr; gap:12px; margin:18px 0; } .toggles label { display:flex; gap:8px; align-items:center; }
  .condition, .warning { color:var(--warning); } .readiness { border-top:1px solid var(--border); padding:16px 0; }
  dl { display:grid; grid-template-columns:150px minmax(0,1fr); gap:10px 16px; font-size:13px; } dt { color:var(--text-muted); } dd { margin:0; overflow-wrap:anywhere; }
  .field-toolbar { display:flex; align-items:center; justify-content:space-between; gap:12px; flex-wrap:wrap; } .field-toolbar h3 { margin:0; }
  .field { border-top:1px solid var(--border); padding:13px 0; } .field summary { display:flex; justify-content:space-between; gap:16px; } .field summary::before { content:'▸'; color:var(--text-muted); } .field[open] summary::before { content:'▾'; }
  .field summary>span:first-of-type { flex:1; min-width:0; } .field small { display:block; font:11px var(--mono); color:var(--text-muted); overflow-wrap:anywhere; margin-top:5px; }
  .field-flags { text-align:right; font-size:10px; flex-shrink:0; } .field-flags span { display:block; color:var(--text-muted); margin-bottom:4px; } .field-flags .has-mapping { color:var(--info); }
  .mapping { background:var(--surface-2); padding:14px; border-radius:8px; font-size:12px; } code { overflow-wrap:anywhere; }
  .dependencies>div, .popup-list>div { border-top:1px solid var(--border); padding:12px 0; } .popup-list h4 { margin-top:6px; }
  .pager { display:flex; justify-content:space-between; gap:12px; align-items:center; margin:18px 0; } .source { font-size:11px; overflow-wrap:anywhere; }
  .preview select { max-width:100%; }
  @media(max-width:1000px) { .workspace { grid-template-columns:220px minmax(0,1fr); gap:14px; } .field-flags { max-width:110px; } }
  @media(max-width:720px) { .workspace { display:block; } aside { position:static; max-height:320px; margin-bottom:16px; } .overview { grid-template-columns:1fr 1fr; } .workspace-header { align-items:start; flex-direction:column; } .toggles { grid-template-columns:1fr; } dl { grid-template-columns:1fr; gap:5px; } dd { margin-bottom:10px; } .card { padding:16px; } }
</style>
