<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type Health, type Job } from './lib/api';
  import Upload from './lib/Upload.svelte';
  import Review from './lib/Review.svelte';
  import Approve from './lib/Approve.svelte';
  import Run from './lib/Run.svelte';
  import ThemeToggle from './lib/ThemeToggle.svelte';
  import Browse from './lib/Browse.svelte';
  import WorkspaceIcon from './lib/personalhome/WorkspaceIcon.svelte';
  import QuoteApplication from './lib/personalhome/QuoteApplication.svelte';
  import Rules from './lib/rules/Rules.svelte';
  import BillingWorkspace from './lib/billing/BillingWorkspace.svelte';

  type View = 'migrate' | 'browse' | 'rules' | 'personalhome' | 'billing';
  const viewOf = (h: string): View => h.startsWith('#reference') || h.startsWith('#personalhome') ? 'personalhome' : h.startsWith('#billing') ? 'billing' : h.startsWith('#browse') ? 'browse' : h.startsWith('#rules') ? 'rules' : 'migrate';
  let view = $state<View>(viewOf(location.hash));

  type Step = 'upload' | 'review' | 'approve' | 'run';
  const steps: { id: Step; label: string }[] = [
    { id: 'upload', label: 'Upload' },
    { id: 'review', label: 'Review & override' },
    { id: 'approve', label: 'Approve' },
    { id: 'run', label: 'Write to Helix' },
  ];

  let step = $state<Step>('upload');
  let job = $state<Job | null>(null);
  let health = $state<Health | null>(null);

  onMount(async () => {
    const m = location.hash.match(/job=([0-9a-f]+)/);
    if (m) {
      try {
        job = await api.job(m[1]);
        step = job.progress ? 'run' : 'review';
      } catch { history.replaceState(null, '', location.pathname + (location.hash.startsWith('#personalhome') ? location.hash.split('?')[0] : '')); }
    }
    try { health = await api.health(); } catch { health = null; }
  });

  onMount(() => {
    const syncView = () => { const next = viewOf(location.hash); if (next !== view && !window.dispatchEvent(new Event('quote-before-leave', { cancelable: true }))) return; view = next; };
    window.addEventListener('hashchange', syncView);
    return () => window.removeEventListener('hashchange', syncView);
  });

  // Keep the job in the URL so a refresh returns to it.
  $effect(() => {
    if (view === 'billing') { if (!location.hash.startsWith('#billing')) history.replaceState(null, '', location.pathname + '#billing'); return; }
    const want = view === 'personalhome' ? `${(location.hash.startsWith('#personalhome') ? location.hash : '#personalhome/newquote').split('?')[0]}${job ? `?job=${job.id}` : ''}` : view === 'browse' ? '#browse' : view === 'rules' ? '#rules' : job ? `#job=${job.id}` : '';
    if (location.hash !== want) history.replaceState(null, '', location.pathname + want);
  });

  const reachable = (i: number) => i === 0 || (job !== null && (i <= 2 || (job.approvals?.length ?? 0) > 0));
  const stepIndex = $derived(steps.findIndex((s) => s.id === step));

  function switchView(next: View) { if (next === view) return; if (!window.dispatchEvent(new Event('quote-before-leave', { cancelable: true }))) return; view = next; }

  function onUploaded(j: Job) { job = j; step = 'review'; }
  function onApproved(j: Job) { job = j; step = 'run'; }
  function restart() { job = null; step = 'upload'; }
</script>

<div class="workspace-shell">
<aside class="app-rail">
  <a class="rail-brand" href="#personalhome" title="Safe Point Insurance"><img src="/safepoint-logo.webp" alt="Safe Point" width="36" height="36" /></a>
  <nav aria-label="Sections">
    <div class="rail-label">Workspace</div>
    <button class:on={view === 'personalhome'} aria-current={view === 'personalhome' ? 'page' : undefined} onclick={() => switchView('personalhome')}><WorkspaceIcon name="grid"/>Dashboard</button>
    <button class:on={view === 'billing'} aria-current={view === 'billing' ? 'page' : undefined} onclick={() => switchView('billing')}><WorkspaceIcon name="card"/>Billing</button>
    <button class:on={view === 'browse'} aria-current={view === 'browse' ? 'page' : undefined} onclick={() => switchView('browse')}><WorkspaceIcon name="database"/>Browse data</button>
    <button class:on={view === 'migrate'} aria-current={view === 'migrate' ? 'page' : undefined} onclick={() => switchView('migrate')}><WorkspaceIcon name="transfer"/>Migrate</button>
    <button class:on={view === 'rules'} aria-current={view === 'rules' ? 'page' : undefined} onclick={() => switchView('rules')}><WorkspaceIcon name="rules"/>Rules</button>
  </nav>
  <div class="rail-bottom"><span class="avatar" title="Local workspace">MW</span></div>
</aside>
<div class="workspace-content">
<header class="topbar">
  <div class="workspace-label"><div class="segmented" aria-label="Workspace"><span class="seg on">Daily Experience</span></div><span class="workspace-name">Safe Point · {view === 'billing' ? 'Billing' : 'Personal Home'}</span></div>
  <div class="spacer"></div>
  {#if health}<span class="badge {health.helix_reachable || health.offline ? 'ok' : 'bad'}">{health.offline ? '● Local workspace' : health.helix_reachable ? '● Helix connected' : '✕ Helix unreachable'}</span>{/if}
  <ThemeToggle />
</header>

{#if view === 'personalhome'}
<main><QuoteApplication /></main>
{:else if view === 'billing'}
<main><BillingWorkspace /></main>
{:else if view === 'browse'}
<main><Browse /></main>
{:else if view === 'rules'}
<main><Rules /></main>
{:else}
<nav class="stepper" aria-label="Progress">
  {#each steps as s, i}
    <button
      class="step"
      class:active={s.id === step}
      class:done={i < stepIndex}
      disabled={!reachable(i)}
      aria-current={s.id === step ? 'step' : undefined}
      onclick={() => (step = s.id)}
    >
      <span class="num">{i < stepIndex ? '✓' : i + 1}</span>{s.label}
    </button>
    {#if i < steps.length - 1}<span class="sep" aria-hidden="true"></span>{/if}
  {/each}
  {#if job}
    <span class="spacer"></span>
    <span class="xs muted file" title={job.file_sha}>📄 {job.file_name}</span>
    <button class="btn sm ghost" onclick={restart}>New upload</button>
  {/if}
</nav>

<main>
  {#if step === 'upload'}
    <Upload onUploaded={onUploaded} />
  {:else if step === 'review' && job}
    <Review bind:job onNext={() => (step = 'approve')} />
  {:else if step === 'approve' && job}
    <Approve {job} onApproved={onApproved} onBack={() => (step = 'review')} />
  {:else if step === 'run' && job}
    <Run bind:job onBack={() => (step = 'review')} />
  {/if}
</main>
{/if}

</div>
</div>

<style>
  .workspace-shell{min-height:100vh;padding-left:72px}
  .app-rail{width:72px;position:fixed;inset:0 auto 0 0;z-index:20;background:var(--surface);border-right:1px solid var(--border);display:flex;flex-direction:column}
  .rail-brand{height:60px;display:flex;align-items:center;justify-content:center;text-decoration:none}
  .rail-brand img{width:36px;height:36px;object-fit:contain;border-radius:6px;display:block}
  .rail-label{display:none}
  .app-rail nav{display:grid;gap:2px;padding-top:4px}
  .app-rail button{display:flex;flex-direction:column;align-items:center;justify-content:center;gap:4px;padding:8px 4px;min-height:56px;border:0;border-left:3px solid transparent;background:none;color:var(--text-muted);font:500 10px var(--font);cursor:pointer}
  .app-rail button.on{background:var(--accent-soft);border-left-color:var(--accent);color:var(--accent);font-weight:600}
  .app-rail button:hover:not(.on){background:var(--surface-2)}
  .rail-bottom{margin-top:auto;padding:12px 4px;display:flex;justify-content:center}
  .avatar{display:grid;place-items:center;background:var(--surface-3);color:var(--text-muted);width:32px;height:32px;border-radius:50%;font-size:12px;font-weight:600;box-shadow:0 0 0 2px var(--accent)}
  .workspace-content{min-width:0}
  .topbar{min-height:48px;display:flex;align-items:center;gap:16px;padding:4px 20px;background:var(--surface);border-bottom:1px solid color-mix(in srgb, #000 10%, transparent);position:sticky;top:0;z-index:10}
  .workspace-label{display:flex;align-items:center;gap:14px}.workspace-name{color:var(--text-muted);font-size:12px}
  .segmented{display:inline-flex;padding:4px;background:var(--surface-3);border-radius:8px;gap:4px}
  .seg{padding:4px 8px;border-radius:6px;font-size:12px;font-weight:600;color:var(--text-muted)}
  .seg.on{background:var(--experience-bg);color:var(--experience-text);box-shadow:0 1px 3px #73c16766}
  .spacer{flex:1}
  .stepper {
    display: flex; align-items: center; gap: var(--space-2); flex-wrap: wrap;
    padding: var(--space-3) var(--space-5); border-bottom: 1px solid var(--border); background: var(--surface);
  }
  .step {
    display: inline-flex; align-items: center; gap: var(--space-2); border: none; background: none;
    color: var(--text-muted); font: 500 var(--fs) var(--font); padding: 6px 10px; border-radius: 999px; cursor: pointer;
  }
  .step:disabled { cursor: not-allowed; opacity: 0.5; }
  .step:hover:not(:disabled) { background: var(--surface-2); }
  .step.active { color: var(--text); background: var(--accent-soft); }
  .num {
    width: 22px; height: 22px; border-radius: 50%; display: grid; place-items: center;
    font-size: var(--fs-xs); font-weight: 700; background: var(--surface-3); color: var(--text-muted);
  }
  .step.active .num { background: var(--accent); color: var(--accent-text); }
  .step.done .num { background: var(--success-soft); color: var(--success); }
  .sep { width: 28px; height: 1px; background: var(--border-strong); }
  .file { max-width: 340px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  main { padding: var(--space-5); max-width: 1800px; margin: 0 auto; }
  @media (max-width: 720px) {
    .workspace-shell{padding-left:60px}.app-rail{width:60px}.workspace-name{display:none}.topbar{padding:0 14px;gap:8px}.topbar .badge{display:none}main{padding:16px}.sep{display:none}.stepper{padding:12px}
  }
</style>
