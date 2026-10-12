<script lang="ts">
  import InsightCard from '$lib/ai/InsightCard.svelte';
  import { api } from '$lib/api/endpoints';
  import { base } from '$app/paths';
  import { portfolioTotals, UTILIZATION_BAR } from '$lib/cards/cards';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Money from '$lib/components/Money.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import Stat from '$lib/components/Stat.svelte';
  import { formatDate, formatPercent } from '$lib/format';

  let { data } = $props();
  const totals = $derived(portfolioTotals(data.banks));
  const utilization = $derived(totals.limit ? (totals.usage / totals.limit) * 100 : 0);
</script>

<PageHeader title="Credit Cards" subtitle="Cashback, utilisation and statement cycles" />

<div class="grid grid-cols-2 gap-4 lg:grid-cols-5">
  <Stat label="Total limit" value={totals.limit} />
  <Stat label="Used" value={totals.usage} hint=" {formatPercent(utilization, 0)} utilisation" />
  <Stat label="Est. cashback (cycle)" value={totals.estimatedCashback} />
  <Stat label="Refunds YTD" value={totals.refundYtd} />
  <Stat label="Net earn YTD"><Money value={totals.netEarnYtd} tone signed /></Stat>
</div>

{#if data.banks.length === 0}
  <Card class="mt-4"
    ><EmptyState title="No credit cards configured" detail="Cards come from the engine's card configuration." /></Card
  >
{/if}

{#each data.banks as bank (bank.name)}
  <h2 class="mt-8 mb-3 text-lg font-semibold">{bank.name}</h2>
  {#each bank.accounts as group (group.name)}
    <Card
      title={group.name}
      subtitle={group.shared ? `Shared limit across ${group.cards.length} cards` : undefined}
      class="mb-4"
    >
      {#snippet actions()}
        <span class="muted text-xs"
          >Limit <Money value={group.limit} /> · Remaining <Money value={group.remainingLimit} /></span
        >
      {/snippet}
      <div class="grid gap-3 md:grid-cols-2">
        {#each group.cards as card (card.name)}
          <a
            href="{base}/accounts/credit-cards/{encodeURIComponent(card.name)}"
            class="rounded-xl border border-slate-900/10 p-4 transition hover:border-indigo-400 hover:bg-indigo-500/5 dark:border-white/10"
          >
            <div class="flex items-start justify-between gap-3">
              <div>
                <p class="font-medium">{card.name}</p>
                <p class="muted text-xs">Due {formatDate(card.cycle.nextDueDate)} · {card.cycle.daysUntilDue} days</p>
              </div>
              {#if card.cycle.dueUrgent}<Badge tone="red">Due soon</Badge>{/if}
            </div>
            <div class="mt-3 flex items-center gap-2">
              <ProgressBar
                percent={card.utilization}
                color={UTILIZATION_BAR[card.utilizationBand]}
                label="Utilisation"
              />
              <span class="muted text-xs whitespace-nowrap">{formatPercent(card.utilization, 0)}</span>
            </div>
            <dl class="mt-3 grid grid-cols-3 gap-2 text-xs">
              <div>
                <dt class="muted">This cycle</dt>
                <dd><Money value={card.spentThisCycle} /></dd>
              </div>
              <div>
                <dt class="muted">Est. cashback</dt>
                <dd><Money value={card.estimatedCashback} /></dd>
              </div>
              <div>
                <dt class="muted">Net earn YTD</dt>
                <dd><Money value={card.netEarnYtd} tone /></dd>
              </div>
            </dl>
          </a>
        {/each}
      </div>
    </Card>
  {/each}
{/each}

<!-- WM-AI-5 -->
<InsightCard
  title="AI credit card summary"
  class="mt-4"
  load={(force, signal) => api.review('credit-summary', {}, { force, signal })}
/>
