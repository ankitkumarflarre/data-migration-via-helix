<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type Job, type RuleSetInfo } from './api';

  let { onUploaded }: { onUploaded: (j: Job) => void } = $props();

  let ruleSets = $state<RuleSetInfo[]>([]);
  let ruleSet = $state('');
  let file = $state<File | null>(null);
  let dragging = $state(false);
  let busy = $state(false);
  let error = $state('');
  let input: HTMLInputElement;

  onMount(async () => {
    try {
      ruleSets = await api.ruleSets();
      ruleSet = ruleSets[0]?.name ?? '';
    } catch (e) { error = (e as Error).message; }
  });

  function pick(f: File | undefined | null) {
    error = '';
    if (!f) return;
    if (!/\.(xlsx|xlsm)$/i.test(f.name)) { error = 'Choose an .xlsx or .xlsm workbook.'; return; }
    if (f.size > 25 * 1024 * 1024) { error = 'The workbook is larger than 25 MB.'; return; }
    file = f;
  }

  async function submit() {
    if (!file) return;
    busy = true; error = '';
    try { onUploaded(await api.upload(file, ruleSet)); }
    catch (e) { error = (e as Error).message; }
    finally { busy = false; }
  }

  const selected = $derived(ruleSets.find((r) => r.name === ruleSet));
  const kb = (n: number) => (n / 1024 / 1024).toFixed(2) + ' MB';
</script>

<section class="wrap-upload">
  <div class="intro">
    <h1>Upload a rater workbook</h1>
    <p class="muted">The <strong>Policy Data</strong> sheet is read, the confirmed mapping rules are applied, and you review every impacted Helix table and column before anything is written.</p>
  </div>

  <div class="card panel stack">
    <div
      class="drop"
      class:dragging
      role="button"
      tabindex="0"
      aria-label="Choose a workbook"
      onclick={() => input.click()}
      onkeydown={(e) => (e.key === 'Enter' || e.key === ' ') && input.click()}
      ondragover={(e) => { e.preventDefault(); dragging = true; }}
      ondragleave={() => (dragging = false)}
      ondrop={(e) => { e.preventDefault(); dragging = false; pick(e.dataTransfer?.files?.[0]); }}
    >
      <div class="icon" aria-hidden="true">⬆</div>
      {#if file}
        <div class="fname">{file.name}</div>
        <div class="xs muted">{kb(file.size)} · click to choose another</div>
      {:else}
        <div><strong>Drop a workbook here</strong> or click to browse</div>
        <div class="xs muted">.xlsx or .xlsm, up to 25 MB</div>
      {/if}
      <input bind:this={input} type="file" accept=".xlsx,.xlsm" hidden onchange={(e) => pick((e.currentTarget as HTMLInputElement).files?.[0])} />
    </div>

    <div class="row wrap">
      <label for="ruleset">Mapping rules</label>
      <select id="ruleset" bind:value={ruleSet}>
        {#each ruleSets as r}<option value={r.name}>{r.name}</option>{/each}
      </select>
      {#if selected}
        <span class="xs muted">{selected.rules} confirmed rules · sheet “{selected.sheet}” · from <em>{selected.source_report}</em></span>
      {/if}
    </div>

    {#if error}<div class="err" role="alert">✕ {error}</div>{/if}

    <div class="row">
      <span class="spacer"></span>
      <button class="btn primary" disabled={!file || busy || !ruleSet} onclick={submit}>
        {busy ? 'Reading and mapping…' : 'Read & map workbook'}
      </button>
    </div>
  </div>
</section>

<style>
  .wrap-upload { max-width: 760px; margin: var(--space-6) auto; display: flex; flex-direction: column; gap: var(--space-5); }
  .intro { display: flex; flex-direction: column; gap: var(--space-2); }
  .panel { padding: var(--space-5); }
  .drop {
    border: 2px dashed var(--border-strong); border-radius: var(--radius); padding: var(--space-6);
    text-align: center; cursor: pointer; display: flex; flex-direction: column; align-items: center; gap: var(--space-2);
    background: var(--surface-2); transition: border-color 0.15s, background 0.15s;
  }
  .drop:hover, .drop.dragging { border-color: var(--accent); background: var(--accent-soft); }
  .icon { font-size: 28px; color: var(--accent); }
  .fname { font-weight: 600; word-break: break-all; }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-3); border-radius: var(--radius-sm); }
  .spacer { flex: 1; }
</style>
