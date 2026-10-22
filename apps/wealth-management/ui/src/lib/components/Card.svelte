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

<section class="panel p-5 {className}">
  {#if title || actions}
    <header class="mb-4 flex flex-wrap items-start justify-between gap-x-3 gap-y-2">
      <div class="min-w-0">
        {#if title}<h2 class="font-serif text-base font-semibold tracking-tight text-stone-900 dark:text-stone-50">
            {title}
          </h2>{/if}
        {#if subtitle}<p class="muted mt-0.5 text-xs">{subtitle}</p>{/if}
      </div>
      {#if actions}<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>{/if}
    </header>
  {/if}
  {@render children()}
</section>
