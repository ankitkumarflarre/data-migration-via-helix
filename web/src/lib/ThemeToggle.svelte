<script lang="ts">
  import { onMount } from 'svelte';

  type Mode = 'system' | 'light' | 'dark';
  let mode = $state<Mode>('system');

  function apply(m: Mode) {
    const root = document.documentElement;
    if (m === 'system') root.removeAttribute('data-theme');
    else root.setAttribute('data-theme', m);
    try { localStorage.setItem('theme', m); } catch { /* storage unavailable */ }
  }

  onMount(() => {
    try {
      const saved = localStorage.getItem('theme') as Mode | null;
      if (saved === 'light' || saved === 'dark' || saved === 'system') mode = saved;
    } catch { /* storage unavailable */ }
    apply(mode);
  });

  function cycle() {
    mode = mode === 'system' ? 'light' : mode === 'light' ? 'dark' : 'system';
    apply(mode);
  }
  const label = $derived(mode === 'system' ? '◐ System' : mode === 'light' ? '☀ Light' : '☾ Dark');
</script>

<button class="btn sm" onclick={cycle} title="Theme: {mode} (click to change)" aria-label="Theme: {mode}">{label}</button>
