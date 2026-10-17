<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import type { IdleCash } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import { STATUS } from '$lib/status';
  import DayToDayFloatField from './DayToDayFloatField.svelte';
  import EmergencyAccounts from './EmergencyAccounts.svelte';
  import {
    IDLE_CASH_VERDICT,
    IDLE_EXCLUSION_LABELS,
    depositKind,
    depositTerms,
    floatLabel,
    idleAccounts,
    keepLiquidLines,
    keepReady,
    rateLabel,
    sentence,
  } from './idleCash';

  // ROADMAP R6, D34: is ready money beyond the day-to-day float sitting idle?
  // Only instant-access options are compared (no-term or 1-month deposits),
  // so the money stays within reach. The answer first, then the emergency
  // fund, the settings and the math. Advice is about rate only: goal money
  // stays pooled (D31).
  let { idle }: { idle: IdleCash } = $props();

  const verdict = $derived(IDLE_CASH_VERDICT[idle.verdict]);
  const lines = $derived(keepLiquidLines(idle));
  const holding = $derived(idleAccounts(idle));
  const c = $derived(idle.components);
  const months = $derived(c.bufferMonths === 1 ? '1 month' : `${Number(c.bufferMonths.toFixed(1))} months`);
</script>

<Card
  title="Idle cash"
  subtitle="Ready money beyond your day-to-day float, against instant-access deposits"
  class="mt-4"
>
  {#snippet actions()}
    <Badge status={verdict.tone}>{verdict.label}</Badge>
  {/snippet}

  <div class="border-l-4 pl-3 {STATUS[verdict.tone].edge}">
    <p class="text-lg" data-testid="idle-headline">
      {#if idle.verdict === 'nothing_idle'}
        Nothing sits idle: your day-to-day float (<Money value={c.dayToDayFloat} />) and what must stay liquid (<Money
          value={c.keepLiquid}
        />) cover your ready money (<Money value={c.readyMoney} />).
      {:else if idle.best && idle.extraPerYear > 0}
        Keep <strong><Money value={keepReady(idle)} /></strong> ready in your everyday accounts. The other
        <strong><Money value={idle.idle} /></strong> could earn about
        <strong><Money value={idle.extraPerYear} /></strong>
        a year more in {depositKind(idle.best)} at {idle.best.bank} (<span class="tabular"
          >{rateLabel(idle.best.rate)}</span
        >, {depositTerms(idle.best)}). It stays reachable.
      {:else if idle.best}
        About <strong><Money value={idle.idle} /></strong> sits idle at
        <strong class="tabular">{rateLabel(idle.currentRate)}</strong>, already at or above the best instant-access rate
        (<span class="tabular">{rateLabel(idle.best.rate)}</span>, {depositKind(idle.best)} at {idle.best.bank}).
      {:else}
        About <strong><Money value={idle.idle} /></strong> sits idle at
        <strong class="tabular">{rateLabel(idle.currentRate)}</strong>; there is no instant-access rate (no-term or
        1-month deposit) to compare with.
      {/if}
    </p>
    {#if idle.best && idle.verdict === 'worth_moving'}
      <p class="mt-1 text-sm" data-testid="idle-ladder">
        {#if idle.ladder}
          Ladder idea: split it into {idle.ladder.parts} parts of about <Money value={idle.ladder.partAmount} />, opened
          {idle.ladder.spacingDays} days apart, so one part matures every {idle.ladder.spacingDays} days and some money is
          always close to hand. An early withdrawal earns the no-term rate.
        {:else if idle.best.access === 'one_month'}
          One 1-month deposit is enough at this size; it rolls monthly and can be withdrawn early at the no-term rate.
        {:else}
          A no-term deposit can be withdrawn any time, so no ladder is needed.
        {/if}
      </p>
    {/if}
    {#if idle.verdict !== 'worth_moving'}
      <ul class="mt-1 space-y-0.5 text-sm {STATUS[verdict.tone].ink}">
        {#each idle.reasons as reason (reason)}
          <!-- Reasons can quote amounts ("about 150.000 VND a year"): masked like any amount. -->
          <li><MaskedText text={reason} /></li>
        {/each}
      </ul>
    {/if}
  </div>

  <section class="mt-4 text-sm" aria-labelledby="emergency-fund-title" data-testid="idle-emergency">
    <h3 id="emergency-fund-title" class="text-xs font-semibold tracking-wide uppercase">Emergency fund</h3>
    {#if idle.emergency.accounts.length}
      <p class="mt-1">
        Target <Money value={c.emergencyFund} />: your emergency accounts hold <Money value={c.emergencyInAccounts} />
        {#if c.emergencyFromReady > 0}
          and cover <Money value={c.emergencyCovered} />; the other <Money value={c.emergencyFromReady} /> stays in ready
          money.
        {:else}
          and cover all of it.
        {/if}
      </p>
      {#if idle.emergency.betterRate && idle.best}
        <p class="mt-1 flex items-start gap-1 {STATUS.info.ink}" data-testid="emergency-better-rate">
          <Icon name={STATUS.info.icon} size={14} />
          <span
            >Your emergency accounts earn <span class="tabular">{rateLabel(idle.emergency.currentRate)}</span>.
            {sentence(depositKind(idle.best))} at {idle.best.bank} (<span class="tabular"
              >{rateLabel(idle.best.rate)}</span
            >, {depositTerms(idle.best)}) would earn about <Money value={idle.emergency.extraPerYear} /> a year more and keep
            it within reach. Move it only if the new place is as quick to reach in an emergency.</span
          >
        </p>
      {/if}
    {:else}
      <p class="mt-1">
        No emergency account is marked, so the whole target (<Money value={c.emergencyFund} />) is kept from ready
        money. Mark the account that holds it below.
      </p>
    {/if}
  </section>

  {#if idle.issues.length}
    <ul class="mt-2 space-y-0.5 text-xs {STATUS.warning.ink}" aria-label="Idle cash settings to fix">
      {#each idle.issues as issue (issue)}<li>{issue}</li>{/each}
    </ul>
  {/if}

  <div class="mt-4 grid gap-4 sm:grid-cols-2">
    <DayToDayFloatField
      saved={idle.emergency.settings.dayToDayFloat}
      effective={c.dayToDayFloat}
      defaultHint="the spending buffer, {months} of average expense"
      onsaved={() => invalidateAll()}
    />
    <EmergencyAccounts {idle} />
  </div>

  <details class="mt-3 text-sm">
    <summary class="cursor-pointer text-xs font-medium">Show the math</summary>
    <dl class="mt-2 grid max-w-md grid-cols-[1fr_auto] gap-x-6 gap-y-1">
      <dt>Ready money (without emergency accounts)</dt>
      <dd class="text-right"><Money value={c.readyMoney} /></dd>
      {#each lines as line (line.label)}
        <dt>{line.label}</dt>
        <dd class="text-right">− <Money value={line.amount} /></dd>
      {/each}
      <dt class="border-t border-slate-900/10 pt-1 font-semibold dark:border-white/10">Idle</dt>
      <dd class="border-t border-slate-900/10 pt-1 text-right font-semibold dark:border-white/10">
        <Money value={idle.idle} />
      </dd>
    </dl>
    <p class="muted mt-1 text-xs">{floatLabel(idle)} is money you keep for everyday spending.</p>

    {#if idle.excluded.length}
      <p class="mt-3 text-xs font-medium">Never idle</p>
      <ul class="mt-1 max-w-md space-y-1">
        {#each idle.excluded as account (account.name)}
          <li class="flex justify-between gap-2">
            <span class="muted">{account.name}, {IDLE_EXCLUSION_LABELS[account.reason]}</span>
            <span class="muted"><Money value={account.balance} /></span>
          </li>
        {/each}
      </ul>
    {/if}

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
      <p class="muted mt-1 text-xs">
        Compared with {depositKind(idle.best)}{#if idle.best.accessLabel}: {idle.best.accessLabel}{/if}. {idle.best
          .basis}
      </p>
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
