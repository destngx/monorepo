<script lang="ts">
  import type { Snippet } from 'svelte';

  let { title, subtitle, actions }: { title: string; subtitle?: string; actions?: Snippet } = $props();
</script>

<svelte:head><title>{title} · Wealth</title></svelte:head>

<div class="mb-6 flex flex-wrap items-end justify-between gap-3">
  <div class="min-w-0">
    <h1 class="font-serif text-2xl font-semibold tracking-tight sm:text-3xl">{title}</h1>
    {#if subtitle}<p class="muted mt-1 text-sm">{subtitle}</p>{/if}
  </div>
  {#if actions}<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>{/if}
</div>
