<script lang="ts">
  import type { GoalPool, PoolRole } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';

  let { pool, asOf }: { pool: GoalPool; asOf: string } = $props();

  const roleLabel: Record<PoolRole, string> = {
    asset: 'Counted',
    cardDebt: 'Card debt',
    crypto: 'Crypto, not counted',
    archived: 'Archived, not counted',
    empty: 'Empty',
  };
  const months = $derived(pool.bufferMonths === 1 ? '1 month' : `${pool.bufferMonths} months`);
  const listed = $derived(pool.accounts.filter((account) => account.role !== 'empty'));
</script>

<Card title="Savings pool" subtitle="Goals share this money, filled by priority; balances as of the last sync">
  <dl class="grid grid-cols-[1fr_auto] gap-x-4 gap-y-1.5 text-sm">
    <dt>Accounts</dt>
    <dd class="text-right"><Money value={pool.assets} /></dd>
    <dt>Card debt</dt>
    <dd class="text-right">− <Money value={pool.cardDebt} /></dd>
    <dt>
      Spending buffer
      <span class="muted text-xs">({months} of average spending, <Money value={pool.averageMonthlyExpense} />)</span>
    </dt>
    <dd class="text-right">− <Money value={pool.spendingBuffer} /></dd>
    <dt class="border-t border-slate-900/10 pt-1.5 font-semibold dark:border-white/10">Pool</dt>
    <dd class="border-t border-slate-900/10 pt-1.5 text-right font-semibold dark:border-white/10">
      <Money value={pool.total} />
    </dd>
  </dl>

  <p class="muted mt-4 text-xs font-medium tracking-wide uppercase">Not counted toward goals</p>
  <dl class="mt-1.5 grid grid-cols-[1fr_auto] gap-x-4 gap-y-1.5 text-sm">
    <dt>Crypto <span class="muted text-xs">(upside; counts once converted)</span></dt>
    <dd class="text-right"><Money value={pool.crypto} /></dd>
    <dt>Money lent <span class="muted text-xs">(owed to you)</span></dt>
    <dd class="text-right"><Money value={pool.moneyLent} /></dd>
    {#if pool.archived > 0}
      <dt>Archived accounts</dt>
      <dd class="text-right"><Money value={pool.archived} /></dd>
    {/if}
  </dl>

  {#if listed.length}
    <details class="mt-4">
      <summary
        class="cursor-pointer text-xs font-medium transition-colors hover:text-indigo-600 dark:hover:text-indigo-300"
        >Accounts ({listed.length})</summary
      >
      <ul class="mt-2 divide-y divide-slate-900/5 text-sm dark:divide-white/5">
        {#each listed as account (account.name)}
          <li class="flex items-center justify-between gap-3 py-1.5">
            <span class="min-w-0 truncate"
              >{account.name} <span class="muted text-xs">{roleLabel[account.role]}</span></span
            >
            <Money value={account.balance} />
          </li>
        {/each}
      </ul>
    </details>
  {/if}
  <p class="muted mt-3 text-xs">As of {formatDate(asOf)}.</p>
</Card>
