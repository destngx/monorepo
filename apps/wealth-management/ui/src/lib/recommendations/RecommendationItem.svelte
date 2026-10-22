<script lang="ts">
  import MaskedText from '$lib/components/MaskedText.svelte';
  import { base } from '$app/paths';
  import type { Recommendation, RecommendationStatusInput } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Button } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import { SNOOZE_DAYS, sourceLabel, STATUS_LABEL, STATUS_TONE } from './recommendations';

  // One recommendation with its actions (ROADMAP R7). The status is a Badge
  // `status` (icon and label), never colour alone.
  interface Props {
    item: Recommendation;
    /** Another item is changing: actions wait (each change re-reads the list). */
    busy?: boolean;
    /** Shows the status badge; a list grouped by status can leave it out. */
    showStatus?: boolean;
    /** Changes the status; a returned promise keeps the pressed button busy and toasts its error. */
    onstatus: (item: Recommendation, input: RecommendationStatusInput) => unknown;
  }

  let { item, busy = false, showStatus = false, onstatus }: Props = $props();
  const pending = $derived(item.status === 'open' || item.status === 'snoozed');

  // The pressed action spins; the others wait for it.
  let running = $state<string | null>(null);
  async function set(key: string, input: RecommendationStatusInput) {
    running = key;
    try {
      await onstatus(item, input);
    } finally {
      running = null;
    }
  }
  const waiting = (key: string) => (busy || running !== null) && running !== key;
  const ERROR = 'Could not update the recommendation';
</script>

<li
  class="rounded-xl border border-stone-900/10 p-3 transition-colors hover:border-stone-900/20 dark:border-white/10 dark:hover:border-white/20"
>
  <div class="flex flex-wrap items-start justify-between gap-2">
    <div class="min-w-0">
      <p class="text-sm font-semibold"><MaskedText text={item.title} /></p>
      <p class="mt-0.5 text-sm text-stone-700 dark:text-stone-300"><MaskedText text={item.action} /></p>
    </div>
    <div class="flex shrink-0 flex-wrap items-center gap-2 text-sm">
      {#if showStatus}<Badge status={STATUS_TONE[item.status]}>{STATUS_LABEL[item.status]}</Badge>{/if}
      {#if item.impactVnd !== null}
        <span class="whitespace-nowrap"><Money value={item.impactVnd} /> <span class="muted text-xs">a year</span></span
        >
      {:else}
        <span class="muted text-xs">No estimate</span>
      {/if}
    </div>
  </div>

  {#if item.evidence || item.evidencePath}
    <p class="muted mt-2 text-xs">
      {#if item.evidencePath}<a class="link" href="{base}{item.evidencePath}">Evidence</a>{item.evidence
          ? ': '
          : ''}{/if}<MaskedText text={item.evidence} />
    </p>
  {/if}

  <div class="muted mt-1 flex flex-wrap gap-x-3 text-xs">
    <span>{sourceLabel(item.source)}, {formatDate(item.created)}</span>
    {#if item.status === 'snoozed' && item.snoozedUntil}<span>Snoozed until {formatDate(item.snoozedUntil)}</span>{/if}
    {#if (item.status === 'done' || item.status === 'dismissed') && item.statusChanged}
      <span>{STATUS_LABEL[item.status]} {formatDate(item.statusChanged)}</span>
    {/if}
  </div>
  {#if item.outcome}
    <p class="mt-1 text-xs"><span class="font-medium">Outcome:</span> <MaskedText text={item.outcome} /></p>
  {/if}

  <div class="mt-2 flex flex-wrap gap-1">
    {#if pending}
      <Button
        size="sm"
        icon="check"
        busy={running === 'done'}
        disabled={waiting('done')}
        success="Marked done."
        errorContext={ERROR}
        onclick={() => set('done', { status: 'done' })}>Done</Button
      >
      <Button
        size="sm"
        icon="x"
        busy={running === 'dismissed'}
        disabled={waiting('dismissed')}
        success="Dismissed."
        errorContext={ERROR}
        onclick={() => set('dismissed', { status: 'dismissed' })}>Dismiss</Button
      >
      {#each SNOOZE_DAYS as days (days)}
        <Button
          size="sm"
          icon="clock"
          busy={running === `snooze-${days}`}
          disabled={waiting(`snooze-${days}`)}
          success="Snoozed for {days} days."
          errorContext={ERROR}
          onclick={() => set(`snooze-${days}`, { status: 'snoozed', snoozeDays: days })}>Snooze {days} days</Button
        >
      {/each}
    {/if}
    {#if item.status !== 'open'}
      <Button
        size="sm"
        icon="refresh"
        busy={running === 'open'}
        disabled={waiting('open')}
        success="Reopened."
        errorContext={ERROR}
        onclick={() => set('open', { status: 'open' })}>Reopen</Button
      >
    {/if}
  </div>
</li>
