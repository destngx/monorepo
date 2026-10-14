<script lang="ts">
  import type { EngineSettingsForm } from './engine.svelte';

  // Shown while there is something to save or apply; changes can span tabs.
  let { form }: { form: EngineSettingsForm } = $props();

  const restartRequired = $derived(form.view?.restartRequired ?? false);
</script>

{#if form.view && (form.changeCount > 0 || restartRequired || form.saveError)}
  <div
    class="glass sticky bottom-4 z-10 mt-6 flex flex-wrap items-center justify-end gap-3 p-3"
    role="region"
    aria-label="Pending settings"
  >
    {#if form.saveError}
      <p class="mr-auto text-sm text-rose-600 dark:text-rose-400" role="alert">{form.saveError}</p>
    {:else if form.changeCount > 0}
      <p class="mr-auto text-sm">
        {form.changeCount} unsaved change{form.changeCount === 1 ? '' : 's'}
      </p>
    {:else}
      <p class="mr-auto text-sm">Saved. Restart the engine to apply.</p>
    {/if}
    {#if form.changeCount > 0}
      <button type="button" class="btn-ghost" disabled={form.saving} onclick={form.discard}>Discard</button>
      <button type="button" class="btn-primary" disabled={form.saving} onclick={form.save}
        >{form.saving ? 'Saving...' : 'Save settings'}</button
      >
    {:else if restartRequired}
      <button type="button" class="btn-primary" disabled={form.restarting} onclick={form.restart}
        >{form.restarting ? 'Restarting...' : 'Restart engine'}</button
      >
    {/if}
  </div>
{/if}
