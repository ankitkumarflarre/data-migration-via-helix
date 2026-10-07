<script lang="ts">
  import { onMount } from 'svelte';
  import { api, type Health, type Job } from './lib/api';
  import Upload from './lib/Upload.svelte';
  import Review from './lib/Review.svelte';
  import Approve from './lib/Approve.svelte';
  import Run from './lib/Run.svelte';
  import ThemeToggle from './lib/ThemeToggle.svelte';
  import Browse from './lib/Browse.svelte';

  type View = 'migrate' | 'browse';
  let view = $state<View>(location.hash.startsWith('#browse') ? 'browse' : 'migrate');

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
      } catch { history.replaceState(null, '', location.pathname); }
    }
    try { health = await api.health(); } catch { health = null; }
  });

  // Keep the job in the URL so a refresh returns to it.
  $effect(() => {
    const want = view === 'browse' ? '#browse' : job ? `#job=${job.id}` : '';
    if (location.hash !== want) history.replaceState(null, '', location.pathname + want);
  });

  const reachable = (i: number) => i === 0 || (job !== null && (i <= 2 || (job.approvals?.length ?? 0) > 0));
  const stepIndex = $derived(steps.findIndex((s) => s.id === step));

  function onUploaded(j: Job) { job = j; step = 'review'; }
  function onApproved(j: Job) { job = j; step = 'run'; }
  function restart() { job = null; step = 'upload'; }
</script>

<header class="topbar">
  <div class="brand">
    <span class="logo" aria-hidden="true">⇄</span>
    <div>
      <div class="title">Rater → Helix Migrator</div>
      <div class="xs muted">Policy Data · confirmed mapping rules · iteration 1 demo</div>
    </div>
  </div>
  <nav class="views" aria-label="Sections">
    <button class:on={view === 'migrate'} aria-current={view === 'migrate' ? 'page' : undefined} onclick={() => (view = 'migrate')}>Migrate</button>
    <button class:on={view === 'browse'} aria-current={view === 'browse' ? 'page' : undefined} onclick={() => (view = 'browse')}>Browse data</button>
  </nav>
  <div class="spacer"></div>
  {#if health}
    <span class="badge {health.helix_reachable ? 'ok' : 'bad'}" title={health.helix_error ?? health.helix_url}>
      {health.helix_reachable ? '● Helix connected' : '✕ Helix unreachable'}
    </span>
    <span class="xs muted mono host">{health.helix_url.replace('https://', '')}</span>
  {/if}
  <ThemeToggle />
</header>

{#if view === 'browse'}
<main><Browse /></main>
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

<style>
  .topbar {
    display: flex; align-items: center; gap: var(--space-3);
    padding: var(--space-3) var(--space-5); background: var(--surface);
    border-bottom: 1px solid var(--border); position: sticky; top: 0; z-index: 10;
  }
  .brand { display: flex; align-items: center; gap: var(--space-3); }
  .logo {
    width: 34px; height: 34px; border-radius: 9px; display: grid; place-items: center;
    background: var(--accent); color: var(--accent-text); font-size: 18px; font-weight: 700;
  }
  .title { font-weight: 650; font-size: var(--fs-lg); }
  .views { display: flex; gap: 2px; margin-left: var(--space-5); }
  .views button {
    border: none; background: none; color: var(--text-muted); font: 500 var(--fs) var(--font);
    padding: 7px 12px; border-radius: var(--radius-sm); cursor: pointer; white-space: nowrap;
  }
  .views button:hover { background: var(--surface-2); color: var(--text); }
  .views button.on { background: var(--accent-soft); color: var(--text); }
  .spacer { flex: 1; }
  .host { max-width: 240px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
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
  main { padding: var(--space-5); max-width: 1440px; margin: 0 auto; }
  @media (max-width: 720px) {
    .topbar, .stepper { padding: var(--space-3) var(--space-4); }
    main { padding: var(--space-4); }
    .sep, .host { display: none; }
    .views { margin-left: 0; }
  }
</style>
