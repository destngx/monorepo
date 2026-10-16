<script lang="ts">
  import { base } from '$app/paths';
  import type { BillsSummary } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import { safeToSpendTone } from './bills';

  // Home (ROADMAP R5): how much can be spent until the next income. The math
  // and the calendar stay on the bills page.
  let { summary }: { summary: BillsSummary } = $props();
  const safe = $derived(summary.safeToSpend);
  const tone = $derived(safeToSpendTone(safe));
</script>

<Card title="Safe to spend" subtitle="Until {safe.untilIncome ? 'the next income' : 'the end of the next 30 days'}">
  {#snippet actions()}
    <a href="{base}/bills" class="text-xs text-indigo-600 hover:underline dark:text-indigo-400">Bills</a>
  {/snippet}
  <p class="text-3xl font-semibold tracking-tight" data-testid="safe-amount"><Money value={safe.safeToSpend} /></p>
  <p class="mt-1 flex flex-wrap items-center gap-2 text-sm">
    <Badge status={tone}>{safe.shortBy > 0 ? 'Short' : safe.safeToSpend > 0 ? 'Covered' : 'Nothing left'}</Badge>
    {#if safe.shortBy > 0}
      <span>Short by <Money value={safe.shortBy} /> by {formatDate(safe.until)}</span>
    {:else}
      <span>About <Money value={safe.perDay} /> a day for {safe.days} {safe.days === 1 ? 'day' : 'days'}</span>
    {/if}
  </p>
  <p class="muted mt-3 text-xs">
    After <Money value={safe.billsDue + safe.cardPayments} /> of bills and card payments due by {formatDate(safe.until)}
    and <Money value={safe.goalSetAside} /> set aside for goals.
  </p>
</Card>
