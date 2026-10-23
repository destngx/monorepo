<script lang="ts">
  import type { Snippet } from 'svelte';
  import FoldToggle from './FoldToggle.svelte';

  interface Props {
    title?: string;
    subtitle?: string;
    actions?: Snippet;
    children: Snippet;
    class?: string;
    /** Adds a chevron before the title that folds the body; the actions hide while folded. */
    foldable?: boolean;
    /** Whether a foldable card is open; bindable. */
    open?: boolean;
  }

  let {
    title,
    subtitle,
    actions,
    children,
    class: className = '',
    foldable = false,
    open = $bindable(true),
  }: Props = $props();

  const id = $props.id();
  const folded = $derived(foldable && !open);
</script>

<section class="panel p-5 {className}">
  {#if title || actions || foldable}
    <header class="{folded ? '' : 'mb-4'} flex flex-wrap items-start justify-between gap-x-3 gap-y-2">
      <div class="flex min-w-0 items-start gap-1">
        {#if foldable}
          <FoldToggle {open} label={title ?? 'section'} controls="{id}-body" ontoggle={() => (open = !open)} />
        {/if}
        <div class="min-w-0">
          {#if title}<h2 class="heading">{title}</h2>{/if}
          {#if subtitle}<p class="muted mt-0.5 text-xs">{subtitle}</p>{/if}
        </div>
      </div>
      {#if actions && !folded}<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>{/if}
    </header>
  {/if}
  {#if foldable}
    <div id="{id}-body" hidden={folded}>{@render children()}</div>
  {:else}
    {@render children()}
  {/if}
</section>
