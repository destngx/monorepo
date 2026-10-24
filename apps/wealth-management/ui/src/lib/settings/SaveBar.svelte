<script lang="ts">
  import { Button, Icon } from '@libs/ui-kit';
  import type { EngineSettingsForm } from './engine.svelte';

  // Shown while there is something to save or apply; changes can span tabs.
  let { form }: { form: EngineSettingsForm } = $props();

  const restartRequired = $derived(form.view?.restartRequired ?? false);
</script>

{#if form.view && (form.changeCount > 0 || restartRequired || form.saveError)}
  <!-- Sticks above the phone area bar; the right margin keeps Save clear of the assistant button. -->
  <div
    class="panel sticky bottom-tabbar z-10 mt-6 mr-16 flex flex-wrap items-center justify-end gap-3 p-3"
    role="region"
    aria-label="Pending settings"
  >
    {#if form.saveError}
      <p class="mr-auto flex items-start gap-2 text-sm text-rose-600 dark:text-rose-400" role="alert">
        <Icon name="alert" size={16} class="mt-0.5 shrink-0" />{form.saveError}
      </p>
    {:else if form.changeCount > 0}
      <p class="mr-auto text-sm">
        {form.changeCount} unsaved change{form.changeCount === 1 ? '' : 's'}
      </p>
    {:else}
      <p class="mr-auto flex items-center gap-2 text-sm" role="status">
        <Icon name="check" size={16} class="text-emerald-600 dark:text-emerald-400" />Saved. Restart the engine to
        apply.
      </p>
    {/if}
    {#if form.changeCount > 0}
      <Button disabled={form.saving} onclick={form.discard}>Discard</Button>
      <Button variant="primary" busy={form.saving} busyText="Saving…" onclick={form.save}>Save settings</Button>
    {:else if restartRequired}
      <Button variant="primary" icon="refresh" busy={form.restarting} busyText="Restarting…" onclick={form.restart}
        >Restart engine</Button
      >
    {/if}
  </div>
{/if}
