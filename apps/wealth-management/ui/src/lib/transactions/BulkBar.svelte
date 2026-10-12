<script lang="ts">
  import type { Category } from '$lib/api/types';

  // Bulk recategorise (WM-TX-9).
  interface Props {
    count: number;
    categories: Category[];
    busy: boolean;
    category: string;
    onapply: () => void;
    onclear: () => void;
  }

  let { count, categories, busy, category = $bindable(), onapply, onclear }: Props = $props();
</script>

<div
  class="mb-4 flex flex-wrap items-center gap-2 rounded-xl border border-indigo-500/30 bg-indigo-500/5 px-3 py-2 text-sm"
>
  <span class="font-medium">{count} selected</span>
  <label class="sr-only" for="bulk-category">New category</label>
  <select id="bulk-category" class="field w-auto min-w-48" bind:value={category}>
    <option value="" disabled>Move to category</option>
    {#each categories as category (category.name)}
      <option value={category.name}>{category.name}</option>
    {/each}
  </select>
  <button type="button" class="btn-primary" disabled={busy || !category} onclick={onapply}
    >{busy ? 'Updating…' : 'Recategorise'}</button
  >
  <button type="button" class="btn-ghost" disabled={busy} onclick={onclear}>Clear</button>
</div>
