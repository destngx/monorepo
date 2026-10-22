<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import type { BalanceProjection, Reconciliation } from './review';

  // Above the draft list: each account's sheet balance now and once the
  // selected rows are written, so ticking a row shows its effect at once,
  // and the balance a screenshot showed when there is one. A card account
  // with a limit leads with the credit left, its balance below.
  interface Props {
    projections: BalanceProjection[];
    reconciliation: Reconciliation[];
    /** Button verb: "Commit" or "Approve". */
    verb: string;
  }

  let { projections, reconciliation, verb }: Props = $props();

  const checkByAccount = $derived(new Map(reconciliation.map((line) => [line.account, line])));
  // Screenshot balances for accounts with no open row still get checked.
  const checksOnly = $derived(
    reconciliation.filter((line) => !projections.some((projection) => projection.account === line.account)),
  );
</script>

{#snippet check(line: Reconciliation)}
  <p class="muted mt-1 flex flex-wrap items-center gap-1 text-xs" title="as of {line.asOf} · {line.materialId}">
    Screenshot <Money value={line.balanceAfter} currency={line.currency} />
    {#if line.difference === 0}
      <Badge status="good">Matches</Badge>
    {:else}
      <span class="text-amber-700 dark:text-amber-300"
        >Off by <Money value={line.difference} currency={line.currency} signed /></span
      >
    {/if}
  </p>
{/snippet}

<Card title="Balances after {verb.toLowerCase()}" subtitle="Sheet balance now and with the selected rows." class="mb-4">
  {#if projections.length || checksOnly.length}
    <ul class="grid grid-cols-1 gap-2 text-sm sm:grid-cols-[repeat(auto-fill,minmax(15rem,1fr))]">
      {#each projections as line (line.account)}
        {@const screenshot = checkByAccount.get(line.account)}
        <li class="rounded-lg bg-stone-900/5 px-3 py-2 dark:bg-white/5">
          <div class="flex items-baseline justify-between gap-3">
            <span class="font-medium">{line.account}</span>
            {#if line.credit}
              <span class="text-right"
                ><span class="font-medium"><Money value={line.credit.after} currency={line.currency} /></span>
                <span class="muted text-xs">credit left</span></span
              >
            {:else}
              <span class="font-medium"><Money value={line.after} currency={line.currency} /></span>
            {/if}
          </div>
          <div class="muted mt-0.5 flex flex-wrap items-baseline justify-between gap-x-3 text-xs">
            <span>{line.rows ? `${line.rows} selected` : 'None selected'}</span>
            {#if line.credit && line.credit.after !== line.credit.now}
              <span
                >now <Money value={line.credit.now} currency={line.currency} />
                <Money value={line.credit.after - line.credit.now} currency={line.currency} signed tone /></span
              >
            {:else if !line.credit && line.change}
              <span
                >now <Money value={line.current} currency={line.currency} />
                <Money value={line.change} currency={line.currency} signed tone /></span
              >
            {/if}
          </div>
          {#if line.credit}
            <p class="muted mt-0.5 text-xs">Balance <Money value={line.after} currency={line.currency} /></p>
          {/if}
          {#if screenshot}{@render check(screenshot)}{/if}
        </li>
      {/each}
      {#each checksOnly as line (line.account)}
        <li class="rounded-lg bg-stone-900/5 px-3 py-2 dark:bg-white/5">
          <span class="font-medium">{line.account}</span>
          <p class="muted text-xs">Sheet <Money value={line.projected} currency={line.currency} /></p>
          {@render check(line)}
        </li>
      {/each}
    </ul>
  {:else}
    <p class="muted text-sm">No account to show yet.</p>
  {/if}
</Card>
