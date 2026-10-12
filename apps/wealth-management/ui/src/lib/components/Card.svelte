<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title?: string;
    subtitle?: string;
    actions?: Snippet;
    children: Snippet;
    class?: string;
  }

  let { title, subtitle, actions, children, class: className = '' }: Props = $props();
</script>

<section class="glass p-5 {className}">
  {#if title || actions}
    <header class="mb-4 flex items-start justify-between gap-3">
      <div>
        {#if title}<h2 class="text-sm font-semibold tracking-wide text-slate-700 dark:text-slate-200">{title}</h2>{/if}
        {#if subtitle}<p class="muted mt-0.5 text-xs">{subtitle}</p>{/if}
      </div>
      {#if actions}<div class="flex items-center gap-2">{@render actions()}</div>{/if}
    </header>
  {/if}
  {@render children()}
</section>
