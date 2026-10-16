<script lang="ts">
  import { base } from '$app/paths';
  import type { Recommendation, RecommendationStatusInput } from '$lib/api/types';
  import { maskText } from '$lib/ai/moneyMask';
  import Badge from '$lib/components/Badge.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';
  import { SNOOZE_DAYS, sourceLabel, STATUS_LABEL, STATUS_TONE } from './recommendations';

  // One recommendation with its actions (ROADMAP R7). The status is a Badge
  // `status` (icon and label), never colour alone.
  interface Props {
    item: Recommendation;
    busy?: boolean;
    /** Shows the status badge; a list grouped by status can leave it out. */
    showStatus?: boolean;
    onstatus: (item: Recommendation, input: RecommendationStatusInput) => void;
  }

  let { item, busy = false, showStatus = false, onstatus }: Props = $props();
  const pending = $derived(item.status === 'open' || item.status === 'snoozed');
</script>

<li class="rounded-xl border border-slate-900/10 p-3 dark:border-white/10">
  <div class="flex flex-wrap items-start justify-between gap-2">
    <div class="min-w-0">
      <p class="text-sm font-semibold">{maskText(item.title, mask.revealed)}</p>
      <p class="mt-0.5 text-sm text-slate-700 dark:text-slate-300">{maskText(item.action, mask.revealed)}</p>
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
      {#if item.evidencePath}<a
          class="text-indigo-600 hover:underline dark:text-indigo-400"
          href="{base}{item.evidencePath}">Evidence</a
        >{item.evidence ? ': ' : ''}{/if}{maskText(item.evidence, mask.revealed)}
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
    <p class="mt-1 text-xs"><span class="font-medium">Outcome:</span> {maskText(item.outcome, mask.revealed)}</p>
  {/if}

  <div class="mt-2 flex flex-wrap gap-1">
    {#if pending}
      <button
        type="button"
        class="btn-ghost px-2 py-1 text-xs"
        disabled={busy}
        onclick={() => onstatus(item, { status: 'done' })}><Icon name="check" size={14} /> Done</button
      >
      <button
        type="button"
        class="btn-ghost px-2 py-1 text-xs"
        disabled={busy}
        onclick={() => onstatus(item, { status: 'dismissed' })}><Icon name="x" size={14} /> Dismiss</button
      >
      {#each SNOOZE_DAYS as days (days)}
        <button
          type="button"
          class="btn-ghost px-2 py-1 text-xs"
          disabled={busy}
          onclick={() => onstatus(item, { status: 'snoozed', snoozeDays: days })}
          ><Icon name="clock" size={14} /> Snooze {days} days</button
        >
      {/each}
    {/if}
    {#if item.status !== 'open'}
      <button
        type="button"
        class="btn-ghost px-2 py-1 text-xs"
        disabled={busy}
        onclick={() => onstatus(item, { status: 'open' })}><Icon name="refresh" size={14} /> Reopen</button
      >
    {/if}
  </div>
</li>
