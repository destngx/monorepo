<script lang="ts">
  import type { Snippet } from 'svelte';
  import Money from './Money.svelte';
  import { formatSignedPercent } from '$lib/format';

  interface Props {
    label: string;
    value?: number | null;
    /** Percent change; colours up as good unless invert is set (e.g. expenses). */
    change?: number | null;
    invert?: boolean;
    /** Muted note under the value; a snippet when it holds an amount (Money or Masked). */
    hint?: string | Snippet;
    /** Custom value rendering instead of money. */
    children?: Snippet;
  }

  let { label, value, change, invert = false, hint, children }: Props = $props();

  const good = $derived(change === null || change === undefined ? null : invert ? change < 0 : change > 0);
</script>

<div class="glass p-4">
  <p class="muted text-xs font-medium tracking-wide uppercase">{label}</p>
  <div class="mt-1.5 text-xl font-semibold">
    {#if children}{@render children()}{:else}<Money {value} />{/if}
  </div>
  {#if change !== undefined || hint}
    <p class="mt-1 text-xs">
      {#if change !== undefined && change !== null}
        <span class={good ? 'text-emerald-600 dark:text-emerald-400' : 'text-orange-600 dark:text-orange-400'}
          >{formatSignedPercent(change)}</span
        >
      {/if}
      {#if typeof hint === 'function'}
        <span class="muted">{@render hint()}</span>
      {:else if hint}
        <span class="muted">{hint}</span>
      {/if}
    </p>
  {/if}
</div>
