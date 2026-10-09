<script lang="ts">
  // Asks for the reason of a revert or restore before it is saved.
  let { title, detail = '', action, onConfirm, onClose }: {
    title: string; detail?: string; action: string;
    onConfirm: (reason: string) => Promise<void>;
    onClose: () => void;
  } = $props();

  let dialog: HTMLDialogElement;
  let reason = $state('');
  let busy = $state(false);
  let error = $state('');

  $effect(() => { dialog.showModal(); });

  async function confirm(e: SubmitEvent) {
    e.preventDefault();
    if (reason.trim().length < 3) { error = 'Say why you are making this change.'; return; }
    busy = true; error = '';
    try { await onConfirm(reason.trim()); dialog.close(); }
    catch (err) { error = (err as Error).message; }
    finally { busy = false; }
  }
</script>

<dialog bind:this={dialog} onclose={onClose} aria-labelledby="reason-title">
  <form class="stack" onsubmit={confirm}>
    <h2 id="reason-title">{title}</h2>
    {#if detail}<p class="small muted">{detail}</p>{/if}
    <label for="reason-input">Reason <span class="req">*</span></label>
    <!-- svelte-ignore a11y_autofocus -->
    <input id="reason-input" type="text" bind:value={reason} autofocus required />
    {#if error}<div class="err" role="alert">✕ {error}</div>{/if}
    <div class="row">
      <span class="spacer"></span>
      <button type="button" class="btn sm" onclick={() => dialog.close()} disabled={busy}>Cancel</button>
      <button type="submit" class="btn sm primary" disabled={busy}>{busy ? 'Saving…' : action}</button>
    </div>
  </form>
</dialog>

<style>
  dialog { border: 1px solid var(--border); border-radius: var(--radius); background: var(--surface); color: var(--text); padding: var(--space-5); width: min(480px, calc(100vw - 32px)); box-shadow: var(--shadow); }
  dialog::backdrop { background: rgb(0 0 0 / 0.35); }
  h2 { margin: 0; font-size: var(--fs-lg, 1.1rem); }
  p { margin: 0; }
  .req { color: var(--danger); }
  .err { background: var(--danger-soft); color: var(--danger); padding: var(--space-2) var(--space-3); border-radius: var(--radius-sm); }
</style>
