<script lang="ts">
  import { base } from '$app/paths';
  import type { BillsSummary } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import { safeToSpendBadge } from './bills';

  // Home (ROADMAP R5): how much ready money is free to spend until the next
  // wage (D33), after what is committed and the essentials (D36). The math,
  // the accounts and the calendar stay on the bills page.
  let { summary }: { summary: BillsSummary } = $props();
  const safe = $derived(summary.safeToSpend);
  const badge = $derived(safeToSpendBadge(safe));
  const tiers = $derived([
    { label: 'Ready', value: safe.readyMoney },
    { label: 'Committed', value: safe.committed },
    { label: 'Essentials', value: safe.essentials },
    { label: 'Free', value: safe.safeToSpend },
  ]);
</script>

<Card title="Safe to spend" subtitle="Until {safe.untilIncome ? 'the next income' : 'the end of the next 30 days'}">
  {#snippet actions()}
    <a href="{base}/bills" class="text-xs text-indigo-600 hover:underline dark:text-indigo-400">Bills</a>
  {/snippet}
  <p class="muted text-xs font-semibold tracking-wide uppercase">Free to spend</p>
  <p class="text-3xl font-semibold tracking-tight" data-testid="safe-amount"><Money value={safe.safeToSpend} /></p>
  <p class="mt-1 flex flex-wrap items-center gap-2 text-sm">
    <Badge status={badge.tone}>{badge.label}</Badge>
    {#if safe.shortBy > 0}
      <span>Short by <Money value={safe.shortBy} /> by {formatDate(safe.until)}</span>
    {:else if safe.tightBy > 0}
      <span>Essentials short by <Money value={safe.tightBy} /> by {formatDate(safe.until)}</span>
    {:else}
      <span>About <Money value={safe.perDay} /> a day for {safe.days} {safe.days === 1 ? 'day' : 'days'}</span>
    {/if}
  </p>
  {#if safe.natureTagged}
    <ol class="mt-3 flex flex-wrap items-center gap-x-1 gap-y-1 text-xs" aria-label="Where ready money goes">
      {#each tiers as tier, index (tier.label)}
        {#if index > 0}<li aria-hidden="true" class="muted">→</li>{/if}
        <li data-testid="safe-tier" class={tier.label === 'Free' ? 'font-semibold' : ''}>
          <span class="muted">{tier.label}</span>
          <Money value={tier.value} />
        </li>
      {/each}
    </ol>
  {:else}
    <p class="muted mt-3 text-xs" data-testid="safe-untagged">
      Tag categories fixed or essential on the <a href="{base}/budget" class="link">budget page</a> to keep rent and groceries
      apart from money free to spend.
    </p>
  {/if}
  <p class="muted mt-3 text-xs" data-testid="safe-math">
    Committed: bills, card payments{safe.natureTagged ? ' and fixed costs' : ''} due {safe.untilIncome
      ? 'before'
      : 'by'}
    {formatDate(safe.until)}{#if safe.untilIncome}; what is due on payday is paid by that wage{/if}.
    {#if safe.natureTagged}Essentials: your usual groceries, transport and other needs until then.{/if}
    {#if safe.reserved > 0}<Money value={safe.reserved} /> is kept as minimum balances.{/if}
    {#if safe.excludedAccounts.length}Left out: {safe.excludedAccounts.length}
      {safe.excludedAccounts.length === 1 ? 'account' : 'accounts'} (crypto or not ready to use).{/if}
  </p>
</Card>
