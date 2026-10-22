<script lang="ts">
  import { base } from '$app/paths';
  import type { SafeToSpend } from '$lib/api/types';
  import Money from '$lib/components/Money.svelte';
  import { formatMonth } from '$lib/format';

  // What safe to spend sets aside by category (D36): the fixed costs still
  // to pay per month of the window, and the essentials until the horizon.
  let { safe }: { safe: SafeToSpend } = $props();
  const basisLabel = (basis: 'budget' | 'average') => (basis === 'budget' ? 'budget' : '6-month average');
</script>

<div class="mt-4 grid grid-cols-1 gap-4 text-sm sm:grid-cols-2" data-testid="set-aside">
  {#if safe.natureTagged}
    <section aria-labelledby="fixed-costs-title">
      <h3 id="fixed-costs-title" class="text-xs font-semibold tracking-wide uppercase">Fixed costs still to pay</h3>
      <ul class="mt-1 divide-y divide-stone-900/5 dark:divide-white/5">
        {#each safe.fixed as line (line.category + line.month)}
          <li class="flex justify-between gap-2 py-1.5">
            <span class="min-w-0">
              <span class="block truncate">{line.category}, {formatMonth(line.month)}</span>
              <span class="muted text-xs"
                >{basisLabel(line.basis)}
                <Money value={line.expected} />, spent <Money value={line.spent} />{#if line.inBills > 0},
                  <Money value={line.inBills} /> in recurring bills{/if}</span
              >
            </span>
            <Money value={line.amount} />
          </li>
        {:else}
          <li class="muted py-1.5">No category tagged fixed has a budget or spending history.</li>
        {/each}
      </ul>
    </section>
    <section aria-labelledby="essentials-title">
      <h3 id="essentials-title" class="text-xs font-semibold tracking-wide uppercase">Essentials until then</h3>
      <ul class="mt-1 divide-y divide-stone-900/5 dark:divide-white/5">
        {#each safe.essential as line (line.category)}
          <li class="flex justify-between gap-2 py-1.5">
            <span class="min-w-0">
              <span class="block truncate">{line.category}</span>
              <span class="muted text-xs"
                ><Money value={line.perDay} /> a day ({basisLabel(line.basis)}) for {line.days}
                {line.days === 1 ? 'day' : 'days'}</span
              >
            </span>
            <Money value={line.amount} />
          </li>
        {:else}
          <li class="muted py-1.5">No category tagged essential has a budget or spending history.</li>
        {/each}
      </ul>
    </section>
  {:else}
    <p class="muted sm:col-span-2">
      No category is tagged fixed or essential, so nothing is set aside for rent or groceries. Tag them on the
      <a href="{base}/budget" class="link">budget page</a>.
    </p>
  {/if}
</div>
