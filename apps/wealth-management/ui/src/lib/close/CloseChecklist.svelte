<script lang="ts">
  import { base } from '$app/paths';
  import type { MonthlyClose } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { CLOSE_STATE, closeFigures, closeHeadline, closeLink } from './close';

  interface Props {
    close: MonthlyClose;
    /**
     * Writes the month's snapshot; the page reloads the checklist afterwards.
     * A returned promise keeps the button busy and toasts its error.
     */
    onWriteSnapshot: () => unknown;
  }

  // ROADMAP R8: every item's state comes from the engine; each links to where the user acts.
  let { close, onWriteSnapshot }: Props = $props();
  const headline = $derived(closeHeadline(close));
</script>

<section class="panel mb-4 border-l-4 p-5 {STATUS[headline.tone].edge}" aria-labelledby="close-headline">
  <h2 id="close-headline" class="text-lg font-semibold" data-testid="close-headline">{headline.text}</h2>
  <p class="muted mt-1 text-sm">
    {close.monthEnded
      ? `Figures up to ${formatDate(close.asOf)}.`
      : `The month is still running: figures up to ${formatDate(close.asOf)}.`}
  </p>
</section>

<ol class="panel divide-y divide-stone-900/5 dark:divide-white/5" aria-label="Checklist">
  {#each close.items as item (item.key)}
    {@const state = CLOSE_STATE[item.state]}
    {@const link = closeLink(item.key, close.month)}
    <li
      class="flex flex-wrap items-start justify-between gap-3 p-4 transition-colors hover:bg-stone-900/[0.02] dark:hover:bg-white/[0.03]"
      data-item={item.key}
    >
      <div class="min-w-0 flex-1">
        <div class="flex flex-wrap items-center gap-2">
          <Badge status={state.tone}>{state.label}</Badge>
          <h3 class="font-medium">{item.title}</h3>
        </div>
        <p class="muted mt-1 text-sm"><MaskedText text={item.detail} /></p>
        {#if closeFigures(item).length}
          <dl class="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-sm">
            {#each closeFigures(item) as figure (figure.label)}
              <div class="flex gap-1.5">
                <dt class="muted">{figure.label}</dt>
                <dd><Money value={figure.value} /></dd>
              </div>
            {/each}
          </dl>
        {/if}
        {#if item.key === 'snapshot' && close.snapshot}
          <dl class="mt-1 flex flex-wrap gap-x-4 gap-y-1 text-sm" data-testid="close-snapshot">
            <div class="flex gap-1.5">
              <dt class="muted">Net worth</dt>
              <dd><Money value={close.snapshot.netWorth} /></dd>
            </div>
            {#if close.snapshot.healthScore !== null}
              <div class="flex gap-1.5">
                <dt class="muted">Health score</dt>
                <dd class="tabular">{Math.round(close.snapshot.healthScore)}</dd>
              </div>
            {/if}
            {#if close.snapshot.savingsRate !== null}
              <div class="flex gap-1.5">
                <dt class="muted">Savings rate</dt>
                <dd class="tabular">{close.snapshot.savingsRate.toFixed(1)}%</dd>
              </div>
            {/if}
          </dl>
        {/if}
      </div>
      <div class="flex items-center gap-2">
        {#if item.key === 'snapshot' && close.canWriteSnapshot}
          <Button
            variant="primary"
            size="sm"
            busyText="Writing…"
            errorContext="Could not write the snapshot"
            onclick={onWriteSnapshot}>{close.snapshot ? 'Update the snapshot' : 'Write the snapshot'}</Button
          >
        {/if}
        {#if link}
          <a class="link text-sm" href="{base}{link.href}">{link.label}</a>
        {/if}
      </div>
    </li>
  {/each}
</ol>
