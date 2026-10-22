<script lang="ts">
  import type { Category } from '$lib/api/types';
  import { Button } from '@libs/ui-kit';

  // Bulk recategorise (WM-TX-9). Recategorise is busy while `onapply` runs;
  // the page reports per-row results, an unexpected error is toasted here.
  interface Props {
    count: number;
    categories: Category[];
    /** The categories are still streaming in. */
    loading?: boolean;
    category: string;
    onapply: () => Promise<unknown>;
    onclear: () => void;
  }

  let { count, categories, loading = false, category = $bindable(), onapply, onclear }: Props = $props();
  let applying = $state(false);

  async function apply() {
    applying = true;
    try {
      await onapply();
    } finally {
      applying = false;
    }
  }
</script>

<div
  class="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-accent-500/30 bg-accent-500/5 px-3 py-2 text-sm"
>
  <span class="font-medium">{count} selected</span>
  <label class="sr-only" for="bulk-category">New category</label>
  <select id="bulk-category" class="field w-auto min-w-48" bind:value={category} disabled={loading || applying}>
    <option value="" disabled>{loading ? 'Loading categories…' : 'Move to category'}</option>
    {#each categories as category (category.name)}
      <option value={category.name}>{category.name}</option>
    {/each}
  </select>
  <Button
    variant="primary"
    disabled={!category}
    busyText="Updating…"
    errorContext="Could not recategorise"
    onclick={apply}>Recategorise</Button
  >
  <Button disabled={applying} onclick={onclear}>Clear</Button>
</div>
