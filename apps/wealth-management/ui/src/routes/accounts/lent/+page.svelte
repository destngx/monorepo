<script lang="ts">
  import { base } from '$app/paths';
  import type { MoneyLentPerson } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Masked from '$lib/components/Masked.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import SkeletonStat from '$lib/components/skeleton/SkeletonStat.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate } from '$lib/format';
  import { ageTone, balanceStatus, formatAge, repayStatus, splitPeople } from '$lib/lending/lending';

  // The header renders at once; the ledger streams in (one request).
  let { data } = $props();

  // One person's history open at a time; keyed by the person's position in the full list.
  let expanded = $state<string | null>(null);
  const toggle = (key: string) => (expanded = expanded === key ? null : key);
</script>

<PageHeader title="Money lent" subtitle="From transactions in lending categories; names hide with the privacy mask" />

<Deferred data={data.lent} label="money lent">
  {#snippet children(lent)}
    {@const groups = splitPeople(lent.people)}
    <div>
      {#if lent.people.length === 0}
        <Card>
          <EmptyState
            title="No money lent yet"
            detail="Tag the category you record lending in as Lending on the Budget page. Its payments are money lent, its deposits money repaid, and the payee is the person."
          />
          <p class="text-center text-sm">
            <a class="btn-ghost" href="{base}/budget"><Icon name="budget" size={16} /> Category natures</a>
          </p>
        </Card>
      {:else}
        <!-- WM-LEND-2 headline -->
        <Card>
          <p class="muted text-xs font-medium tracking-wide uppercase">Owed to you</p>
          <div class="mt-1 text-3xl font-semibold tracking-tight"><Money value={lent.owedToYou} /></div>
          <p class="muted mt-2 text-sm">
            {#if lent.peopleOwing === 0}
              Nobody owes you money right now.
            {:else}
              {lent.peopleOwing === 1 ? '1 person owes' : `${lent.peopleOwing} people owe`} you; the oldest open balance is
              {formatAge(lent.oldestDaysOutstanding)} old.
            {/if}
            {#if lent.peopleYouOwe > 0}
              You owe <Money value={lent.youOwe} /> to {lent.peopleYouOwe === 1
                ? '1 person'
                : `${lent.peopleYouOwe} people`}.
            {/if}
          </p>
        </Card>

        <div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
          <Stat label="Owed to you" value={lent.owedToYou} hint="Counted as an asset in net worth" />
          <Stat label="People owing you"><span class="tabular">{lent.peopleOwing}</span></Stat>
          <Stat label="Oldest open balance"><span>{formatAge(lent.oldestDaysOutstanding)}</span></Stat>
          <Stat label="You owe" value={lent.youOwe} hint="Counted as a liability" />
        </div>

        <!-- WM-LEND-3 per person -->
        <Card title="Open balances" subtitle="People who owe you first, then people you owe" class="mt-4">
          {#if groups.open.length}
            {@render ledger(groups.open, 'open')}
          {:else}
            <EmptyState title="Everything is settled" />
          {/if}
        </Card>

        {#if groups.settled.length}
          <details class="glass mt-4 p-5">
            <summary
              class="cursor-pointer rounded-md text-sm font-semibold tracking-wide text-slate-700 transition-colors hover:text-indigo-600 dark:text-slate-200 dark:hover:text-indigo-300"
            >
              Settled ({groups.settled.length})
            </summary>
            <div class="mt-4">{@render ledger(groups.settled, 'settled')}</div>
          </details>
        {/if}
      {/if}
    </div>
  {/snippet}
  {#snippet skeleton()}
    <div>
      <SkeletonCard lines={2} label="what is owed to you" />
      <div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
        {#each ['owed to you', 'people owing you', 'oldest open balance', 'you owe'] as label (label)}
          <SkeletonStat {label} />
        {/each}
      </div>
      <SkeletonCard title="Open balances" rows={5} lines={0} class="mt-4" />
    </div>
  {/snippet}
</Deferred>

{#snippet ledger(people: MoneyLentPerson[], group: string)}
  <div class="-mx-5 overflow-x-auto">
    <table class="w-full min-w-[60rem] text-sm">
      <thead>
        <tr class="muted text-left text-xs uppercase">
          <th class="px-5 py-2">Person</th>
          <th class="px-5 py-2 text-right">Outstanding</th>
          <th class="px-5 py-2 text-right">Lent</th>
          <th class="px-5 py-2 text-right">Repaid</th>
          <th class="px-5 py-2">Last activity</th>
          <th class="px-5 py-2">Outstanding for</th>
          <th class="px-5 py-2">Repay by</th>
          <th class="px-5 py-2"><span class="sr-only">History</span></th>
        </tr>
      </thead>
      <tbody class="divide-y divide-slate-900/5 dark:divide-white/5">
        {#each people as person, index (`${group}-${index}`)}
          {@const key = `${group}-${index}`}
          <tr>
            <td class="px-5 py-2.5 font-medium">
              <Masked text={person.name} />
              <span class="ml-1.5"
                ><Badge status={balanceStatus(person.outstanding).tone}>{balanceStatus(person.outstanding).label}</Badge
                ></span
              >
            </td>
            <td class="px-5 py-2.5 text-right font-semibold"><Money value={person.outstanding} tone /></td>
            <td class="px-5 py-2.5 text-right"><Money value={person.lent} /></td>
            <td class="px-5 py-2.5 text-right"><Money value={person.repaid} /></td>
            <td class="px-5 py-2.5">{formatDate(person.lastActivity)}</td>
            <td class="px-5 py-2.5">
              {#if person.daysOutstanding !== null}
                <Badge status={ageTone(person.daysOutstanding)}>{formatAge(person.daysOutstanding)}</Badge>
                <span class="muted text-xs">since {formatDate(person.openSince)}</span>
              {:else}
                <span class="muted">-</span>
              {/if}
            </td>
            <td class="px-5 py-2.5">
              {#if person.repayBy !== null}
                {@const repay = repayStatus(person.daysUntilRepay)}
                {formatDate(person.repayBy)}
                {#if repay}<span class="ml-1.5"><Badge status={repay.tone}>{repay.label}</Badge></span>{/if}
              {:else}
                <span class="muted" title="Write the agreed date in a memo, e.g. repay 30/11/2026">-</span>
              {/if}
            </td>
            <td class="px-5 py-2.5 text-right">
              <button
                type="button"
                class="btn-ghost px-2"
                aria-expanded={expanded === key}
                aria-label="{expanded === key ? 'Hide' : 'Show'} transactions"
                title="{expanded === key ? 'Hide' : 'Show'} transactions"
                onclick={() => toggle(key)}
                ><span class="inline-block transition-transform {expanded === key ? 'rotate-180' : ''}"
                  ><Icon name="chevronDown" size={16} /></span
                >
                {person.transactions.length}</button
              >
            </td>
          </tr>
          {#if expanded === key}
            <tr>
              <td colspan="8" class="bg-slate-900/[0.02] px-5 py-3 dark:bg-white/[0.02]">
                <table class="w-full text-xs">
                  <thead>
                    <tr class="muted text-left uppercase">
                      <th class="py-1 pr-4">Date</th>
                      <th class="py-1 pr-4">Account</th>
                      <th class="py-1 pr-4">Memo</th>
                      <th class="py-1 pr-4 text-right">Lent</th>
                      <th class="py-1 text-right">Repaid</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#each person.transactions as transaction (transaction.id)}
                      <tr>
                        <td class="py-1 pr-4">{formatDate(transaction.date)}</td>
                        <td class="py-1 pr-4">{transaction.accountName}</td>
                        <td class="py-1 pr-4"
                          >{#if transaction.memo}<Masked text={transaction.memo} />{/if}</td
                        >
                        <td class="py-1 pr-4 text-right">
                          {#if transaction.payment}<Money value={transaction.payment} />{/if}
                        </td>
                        <td class="py-1 text-right">
                          {#if transaction.deposit}<Money value={transaction.deposit} />{/if}
                        </td>
                      </tr>
                    {/each}
                  </tbody>
                </table>
              </td>
            </tr>
          {/if}
        {/each}
      </tbody>
    </table>
  </div>
{/snippet}
