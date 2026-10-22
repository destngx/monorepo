<script lang="ts">
  import type { Snippet } from 'svelte';

  let { title, subtitle, actions }: { title: string; subtitle?: string; actions?: Snippet } = $props();
</script>

<div class="mb-8 flex flex-wrap items-end justify-between gap-4">
  <div class="min-w-0">
    <h1 class="text-4xl leading-tight sm:text-5xl">{title}</h1>
    {#if subtitle}<p class="muted mt-1 text-sm">{subtitle}</p>{/if}
  </div>
  {#if actions}<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>{/if}
</div>
