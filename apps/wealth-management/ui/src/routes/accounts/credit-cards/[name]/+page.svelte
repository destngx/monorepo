<script lang="ts">
  import { base } from '$app/paths';
  import RecommendationsPanel from '$lib/recommendations/RecommendationsPanel.svelte';
  import { track } from '$lib/api/deferred.svelte';
  import { api } from '$lib/api/endpoints';
  import CardDetail from '$lib/cards/CardDetail.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SkeletonCard from '$lib/components/skeleton/SkeletonCard.svelte';
  import SkeletonStat from '$lib/components/skeleton/SkeletonStat.svelte';
  import { currentMonth, formatDate, formatMonth } from '$lib/format';

  // The name is in the URL: the header, back link and AI summary render at
  // once; the card's report streams in (the skeleton returns for another card).
  let { data } = $props();
  const detail = track(() => data.detail, { scope: () => data.name });
  const card = $derived(detail.value?.card);

  const subtitle = $derived(
    card
      ? `${card.bank} · ${card.program} · ${card.accountName}${card.last4 ? ` · *${card.last4}` : ''}${
          card.closedFrom ? ` · closed ${formatDate(card.closedFrom)}` : card.expiry ? ` · expires ${card.expiry}` : ''
        }`
      : undefined,
  );

  // AI summary for the current or a past period (WM-CARD-5); past months join once the history loads.
  const summaryMonths = $derived(
    [...new Set([currentMonth(), ...(card?.history ?? []).map((m) => m.month)])].sort().reverse().slice(0, 12),
  );
  let summaryMonth = $state(currentMonth());
</script>

<a href="{base}/accounts/credit-cards" class="link muted mb-3 inline-flex items-center gap-1 text-sm"
  ><Icon name="chevronLeft" size={14} /> Credit cards</a
>
<PageHeader title={data.name} {subtitle} />

<Deferred data={data.detail} scope={data.name} label="the card">
  {#snippet children(found)}
    {#if found}
      <CardDetail bank={found.bank} card={found.card} />
    {:else}
      <Card>
        <EmptyState
          title="No credit card named {data.name}"
          detail="It may have been renamed in the Cards tab. Pick a card from the credit cards page."
        />
      </Card>
    {/if}
  {/snippet}
  {#snippet skeleton()}
    <div>
      <div class="grid gap-4 lg:grid-cols-3">
        <SkeletonCard title="Limit" lines={3} label="the card" />
        <SkeletonCard title="This period" lines={4} label="the card" />
        <SkeletonCard title="Pay by" lines={3} label="the card" />
      </div>
      <div class="mt-4 grid grid-cols-2 gap-4 lg:grid-cols-4">
        {#each { length: 4 }, index (index)}<SkeletonStat label="the card" />{/each}
      </div>
    </div>
  {/snippet}
</Deferred>

<!-- WM-CARD-5 -->
<div class="mt-4">
  <label class="label" for="summary-month">AI summary period</label>
  <select id="summary-month" class="field mb-3 w-auto" bind:value={summaryMonth}>
    {#each summaryMonths as month (month)}
      <option value={month}>{formatMonth(month)}</option>
    {/each}
  </select>
  <RecommendationsPanel
    title="Recommendations"
    subtitle="From the period summary of {data.name} · {formatMonth(summaryMonth)}"
    page="/accounts/credit-cards"
    review={(force, signal) =>
      api.review('credit-summary', { card: data.name, month: summaryMonth }, { force, signal })}
  />
</div>
