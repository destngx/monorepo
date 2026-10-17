<script lang="ts">
  import Icon from '$lib/components/Icon.svelte';
  import type { BulkFailure } from './entry';

  interface Props {
    failures: BulkFailure[];
    ondismiss: () => void;
  }

  let { failures, ondismiss }: Props = $props();
</script>

<div role="alert" class="mb-4 rounded-xl border border-orange-500/40 p-3 text-sm">
  <div class="flex items-start justify-between gap-2">
    <p class="font-medium text-orange-700 dark:text-orange-300">These were not recategorised and stay selected:</p>
    <button type="button" class="icon-btn" aria-label="Dismiss" title="Dismiss" onclick={ondismiss}
      ><Icon name="x" size={16} /></button
    >
  </div>
  <ul class="mt-1 space-y-0.5">
    {#each failures as failure (failure.id)}
      <li><span class="font-medium">{failure.payee}</span> ({failure.id}): {failure.message}</li>
    {/each}
  </ul>
</div>
