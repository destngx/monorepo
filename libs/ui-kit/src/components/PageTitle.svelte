<script lang="ts">
  import type { Snippet } from 'svelte';

  // The top of a page: the title, an optional subtitle and the page's actions.
  // With `site`, it also sets the document title to "Title · Site".
  interface Props {
    title: string;
    subtitle?: string;
    site?: string;
    actions?: Snippet;
  }

  let { title, subtitle, site, actions }: Props = $props();
</script>

<svelte:head>
  {#if site}<title>{title} · {site}</title>{/if}
</svelte:head>

<div class="mb-8 flex flex-wrap items-end justify-between gap-4">
  <div class="min-w-0">
    <h1 class="text-4xl leading-tight sm:text-5xl">{title}</h1>
    {#if subtitle}<p class="muted mt-1 text-sm">{subtitle}</p>{/if}
  </div>
  {#if actions}<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>{/if}
</div>
