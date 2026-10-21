<script lang="ts">
  import type { Snippet } from 'svelte';

  let { title, subtitle, actions }: { title: string; subtitle?: string; actions?: Snippet } = $props();
</script>

<div class="mb-5 flex flex-wrap items-end justify-between gap-3">
  <div class="min-w-0">
    <h1 class="text-2xl font-semibold tracking-tight">{title}</h1>
    {#if subtitle}<p class="muted mt-1 text-sm">{subtitle}</p>{/if}
  </div>
  {#if actions}<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>{/if}
</div>
