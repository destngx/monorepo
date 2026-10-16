<script lang="ts">
  import type { IdleCash } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Money from '$lib/components/Money.svelte';
  import { STATUS } from '$lib/status';
  import { IDLE_CASH_VERDICT, idleAccounts, keepLiquidLines, rateLabel } from './idleCash';

  // ROADMAP R6: does money sit idle in low-yield accounts? The answer first, then the math.
  // Advice is about rate only: goal money stays pooled (D31).
  let { idle }: { idle: IdleCash } = $props();

  const verdict = $derived(IDLE_CASH_VERDICT[idle.verdict]);
  const lines = $derived(keepLiquidLines(idle));
  const holding = $derived(idleAccounts(idle));
</script>

<Card title="Idle cash" subtitle="Spendable cash beyond what you need soon, against the best deposit rate" class="mt-4">
  {#snippet actions()}
    <Badge status={verdict.tone}>{verdict.label}</Badge>
  {/snippet}

  <div class="border-l-4 pl-3 {STATUS[verdict.tone].edge}">
    <p class="text-lg" data-testid="idle-headline">
      {#if idle.verdict === 'nothing_idle'}
        Nothing sits idle: what must stay liquid (<Money value={idle.components.keepLiquid} />) covers your spendable
        cash (<Money value={idle.components.spendableCash} />).
      {:else if idle.best && idle.extraPerYear <= 0}
        About <strong><Money value={idle.idle} /></strong> sits idle at
        <strong class="tabular">{rateLabel(idle.currentRate)}</strong>, already at or above the best published deposit
        rate (<span class="tabular">{rateLabel(idle.best.rate)}</span>, {idle.best.bank}).
      {:else if idle.best}
        About <strong><Money value={idle.idle} /></strong> sits idle at
        <strong class="tabular">{rateLabel(idle.currentRate)}</strong>; a deposit at
        <strong class="tabular">{rateLabel(idle.best.rate)}</strong> ({idle.best.bank}) would earn about
        <strong><Money value={idle.extraPerYear} /></strong> more a year.
      {:else}
        About <strong><Money value={idle.idle} /></strong> sits idle at
        <strong class="tabular">{rateLabel(idle.currentRate)}</strong>; there is no published deposit rate to compare
        with.
      {/if}
    </p>
    {#if idle.verdict !== 'worth_moving'}
      <ul class="mt-1 space-y-0.5 text-sm {STATUS[verdict.tone].ink}">
        {#each idle.reasons as reason (reason)}
          <li>{reason}</li>
        {/each}
      </ul>
    {/if}
  </div>

  <details class="mt-3 text-sm">
    <summary class="cursor-pointer text-xs font-medium">Show the math</summary>
    <dl class="mt-2 grid max-w-md grid-cols-[1fr_auto] gap-x-6 gap-y-1">
      <dt>Spendable cash</dt>
      <dd class="text-right"><Money value={idle.components.spendableCash} /></dd>
      {#each lines as line (line.label)}
        <dt>{line.label}</dt>
        <dd class="text-right">− <Money value={line.amount} /></dd>
      {/each}
      <dt class="border-t border-slate-900/10 pt-1 font-semibold dark:border-white/10">Idle</dt>
      <dd class="border-t border-slate-900/10 pt-1 text-right font-semibold dark:border-white/10">
        <Money value={idle.idle} />
      </dd>
    </dl>

    {#if holding.length}
      <p class="mt-3 text-xs font-medium">Where the idle cash is assumed to sit (lowest rate first)</p>
      <ul class="mt-1 max-w-md space-y-1">
        {#each holding as account (account.name)}
          <li class="flex justify-between gap-2">
            <span
              >{account.name}, {rateLabel(account.rate)}{#if !account.rateFromNote}
                <span class="muted">(no rate in the note)</span>{/if}</span
            >
            <Money value={account.idle} />
          </li>
        {/each}
      </ul>
      <p class="mt-2">Current rate, weighted: <span class="tabular">{rateLabel(idle.currentRate)}</span></p>
    {/if}
    {#if idle.best}
      <p class="mt-1">
        Extra a year = idle × ({rateLabel(idle.best.rate)} − {rateLabel(idle.currentRate)}) =
        <Money value={idle.extraPerYear} />
      </p>
      <p class="muted mt-1 text-xs">{idle.best.basis}</p>
    {:else if idle.ratesIssue}
      <p class="muted mt-1 text-xs">{idle.ratesIssue}</p>
    {/if}
    <p class="muted mt-2 text-xs">{idle.keepLiquidRule}</p>
    <p class="muted mt-1 text-xs">{idle.rateRule}</p>
    <p class="muted mt-1 text-xs">
      {idle.thresholdRule} This is about the rate only: goal money stays pooled, it does not need its own account.
    </p>
  </details>
</Card>
