<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import type { DepositRate, IdleCash } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Card, Icon } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { STATUS } from '$lib/status';
  import AccountProducts from './AccountProducts.svelte';
  import FoldRow from '$lib/components/FoldRow.svelte';
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
    place,
    rateLabel,
    sentence,
  } from './idleCash';

  // ROADMAP R6, D34: is ready money beyond the day-to-day float sitting idle?
  // Only instant-access options are compared (no-term or 1-month deposits),
  // so the money stays within reach. The answer first, then the emergency
  // fund, the settings and the math. Advice is about rate only: goal money
  // stays pooled (D31). The card stays short: the answer and four figures
  // show, and each detail folds into its own row.
  let { idle }: { idle: IdleCash } = $props();

  const verdict = $derived(IDLE_CASH_VERDICT[idle.verdict]);
  const lines = $derived(keepLiquidLines(idle));
  const holding = $derived(idleAccounts(idle));
  const c = $derived(idle.components);
  const hasMoves = $derived(idle.moves.length > 0 || idle.switches.length > 0 || idle.emergency.moves.length > 0);
  const moveGroups = $derived([
    { title: 'Idle cash', moves: idle.moves },
    { title: 'Switch your wallet balance', moves: idle.switches },
    { title: 'Emergency fund', moves: idle.emergency.moves },
  ]);
  // The accounts whose rate is this product's live rate.
  const ownProducts = (option: DepositRate) =>
    option.productId
      ? [...idle.accounts, ...idle.emergency.accounts]
          .filter((account) => account.productId === option.productId)
          .map((account) => account.name)
      : [];
  const moveCount = $derived(idle.moves.length + idle.switches.length + idle.emergency.moves.length);
  const months = $derived(c.bufferMonths === 1 ? '1 month' : `${Number(c.bufferMonths.toFixed(1))} months`);
</script>

<Card title="Idle cash" subtitle="Ready money beyond your day-to-day float" class="mt-4">
  {#snippet actions()}
    <Badge status={verdict.tone}>{verdict.label}</Badge>
  {/snippet}

  <div class="border-l-4 pl-3 {STATUS[verdict.tone].edge}">
    <p class="text-sm" data-testid="idle-headline">
      {#if idle.verdict === 'nothing_idle'}
        Nothing sits idle: your day-to-day float (<Money value={c.dayToDayFloat} />) and what must stay liquid (<Money
          value={c.keepLiquid}
        />) cover your ready money (<Money value={c.readyMoney} />).
      {:else if idle.best && idle.extraPerYear > 0}
        Keep <strong><Money value={keepReady(idle)} /></strong> ready in your everyday accounts. The other
        <strong><Money value={idle.idle} /></strong> could earn about
        <strong><Money value={idle.extraPerYear} /></strong>
        a year more in {place(idle.best)} (<span class="tabular">{rateLabel(idle.best.rate)}</span>, {depositTerms(
          idle.best,
        )}). It stays reachable.
      {:else if idle.best}
        About <strong><Money value={idle.idle} /></strong> sits idle at
        <strong class="tabular">{rateLabel(idle.currentRate)}</strong>, already at or above the best instant-access rate
        (<span class="tabular">{rateLabel(idle.best.rate)}</span>, {place(idle.best)}).
      {:else}
        About <strong><Money value={idle.idle} /></strong> sits idle at
        <strong class="tabular">{rateLabel(idle.currentRate)}</strong>; there is no instant-access rate (no-term or
        1-month deposit) to compare with.
      {/if}
    </p>
    {#if idle.best && idle.verdict === 'worth_moving'}
      <p class="muted mt-1 text-xs" data-testid="idle-ladder">
        {#if idle.ladder}
          Ladder idea: split it into {idle.ladder.parts} parts of about <Money value={idle.ladder.partAmount} />, opened
          {idle.ladder.spacingDays} days apart, so one part matures every {idle.ladder.spacingDays} days and some money is
          always close to hand. An early withdrawal earns the no-term rate.
        {:else if idle.best.access === 'one_month'}
          One 1-month deposit is enough at this size; it rolls monthly and can be withdrawn early at the no-term rate.
        {:else if idle.best.access === 'flexible'}
          An earning balance can be spent or withdrawn any time, so no ladder is needed.
        {:else}
          A no-term deposit can be withdrawn any time, so no ladder is needed.
        {/if}
      </p>
    {/if}
    {#if idle.verdict !== 'worth_moving'}
      <ul class="mt-1 space-y-0.5 text-xs {STATUS[verdict.tone].ink}">
        {#each idle.reasons as reason (reason)}
          <!-- Reasons can quote amounts ("about 150.000 VND a year"): masked like any amount. -->
          <li><MaskedText text={reason} /></li>
        {/each}
      </ul>
    {/if}
  </div>

  <dl class="mt-3 grid grid-cols-2 gap-x-4 gap-y-2 text-sm sm:grid-cols-4" data-testid="idle-figures">
    <div class="min-w-0">
      <dt class="label">Ready money</dt>
      <dd class="tabular font-semibold"><Money value={c.readyMoney} /></dd>
    </div>
    <div class="min-w-0">
      <dt class="label">Keep ready</dt>
      <dd class="tabular font-semibold"><Money value={keepReady(idle)} /></dd>
    </div>
    <div class="min-w-0">
      <dt class="label">Idle</dt>
      <dd class="tabular font-semibold"><Money value={idle.idle} /></dd>
    </div>
    <div class="min-w-0">
      <dt class="label">Extra a year</dt>
      <dd class="tabular font-semibold">
        {#if idle.best}<Money value={idle.extraPerYear} />{:else}<span class="muted">n/a</span>{/if}
      </dd>
    </div>
  </dl>

  {#if idle.issues.length}
    <ul class="mt-2 space-y-0.5 text-xs {STATUS.warning.ink}" aria-label="Idle cash settings to fix">
      {#each idle.issues as issue (issue)}<li>{issue}</li>{/each}
    </ul>
  {/if}

  <div
    class="mt-3 divide-y divide-stone-900/5 border-t border-stone-900/5 text-sm dark:divide-white/5 dark:border-white/5"
  >
    <FoldRow title="Emergency fund" money={c.emergencyFund} testid="idle-emergency">
      {#if idle.emergency.accounts.length}
        <p>
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
              {sentence(place(idle.best))} (<span class="tabular">{rateLabel(idle.best.rate)}</span>, {depositTerms(
                idle.best,
              )}) would earn about <Money value={idle.emergency.extraPerYear} /> a year more and keep it within reach. Move
              it only if the new place is as quick to reach in an emergency.</span
            >
          </p>
        {/if}
      {:else}
        <p class="mt-1">
          No emergency account is marked, so the whole target (<Money value={c.emergencyFund} />) is kept from ready
          money. Mark the account that holds it below.
        </p>
      {/if}
    </FoldRow>
    {#if idle.options.length}
      <FoldRow title="Compare instant-access places" count={idle.options.length} testid="idle-options">
        <ul class="divide-y divide-stone-900/5 dark:divide-white/5">
          {#each idle.options as option, index (option.productId ?? `${option.bank}-${option.access}`)}
            {@const own = ownProducts(option)}
            <li class="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 py-1.5">
              <span class="min-w-0">
                <strong>{sentence(place(option))}</strong><span class="muted">, {depositTerms(option)}</span>
                {#if option.url}
                  <a class="link text-xs" href={option.url} target="_blank" rel="noopener noreferrer"
                    >Open<span class="sr-only"> {place(option)} (new tab)</span></a
                  >
                {/if}
              </span>
              <span class="flex flex-wrap items-center gap-1.5">
                {#if index === 0}<Badge status="good">Best</Badge>{/if}
                {#if own.length}<Badge tone="blue">Used by {own.join(', ')}</Badge>{/if}
                {#if option.backing === 'business_cooperation'}<Badge status="warning">not a bank deposit</Badge>{/if}
                {#if option.stale}<Badge status="warning">stale</Badge>{/if}
                <span class="tabular font-medium">{rateLabel(option.rate)}</span>
              </span>
            </li>
          {/each}
        </ul>
      </FoldRow>
    {/if}
    {#if hasMoves}
      <FoldRow title="How to move" count={moveCount} testid="idle-moves">
        {#each moveGroups as group (group.title)}
          {#if group.moves.length}
            <h4 class="mt-1 text-xs font-medium first:mt-0">{group.title}</h4>
            <ul class="mt-1 space-y-3">
              {#each group.moves as move (move.from + move.to)}
                <li data-testid="idle-move">
                  <p>
                    <strong>{move.from}</strong> ({rateLabel(move.fromRate)}) to <strong>{move.to}</strong>
                    ({rateLabel(move.toRate)}): <Money value={move.amount} />,
                    <span class="tabular">{Number(move.rateGapPoints.toFixed(2))}</span>
                    points more, about <Money value={move.extraPerMonth} /> a month and <Money
                      value={move.extraPerYear}
                    /> a year.
                  </p>
                  <p class="mt-0.5">
                    Moving today ({move.today.weekday}): {move.today.daysWithoutInterest}
                    {move.today.daysWithoutInterest === 1 ? 'day' : 'days'} without interest, about
                    <Money value={move.today.interestLost} /> lost.
                  </p>
                  <p class="mt-0.5">
                    Best day, {move.bestDay.weekday}
                    {move.bestDay.date}: {move.bestDay.daysWithoutInterest}
                    {move.bestDay.daysWithoutInterest === 1 ? 'day' : 'days'}, <Money
                      value={move.bestDay.interestLost}
                    />
                    lost, earned back in {move.bestDay.paybackDays}
                    {move.bestDay.paybackDays === 1 ? 'day' : 'days'}.
                  </p>
                  {#if move.worstCaseExtraDays > 0}
                    <p class="mt-0.5 flex items-start gap-1 {STATUS.warning.ink}">
                      <Icon name={STATUS.warning.icon} size={14} />
                      <span
                        >Worst case the money is in transit {move.worstCaseExtraDays}
                        {move.worstCaseExtraDays === 1 ? 'day' : 'days'} longer.</span
                      >
                    </p>
                  {/if}
                  <ol class="mt-1 list-decimal space-y-0.5 pl-5">
                    {#each move.steps as step (step)}
                      <li><MaskedText text={step} /></li>
                    {/each}
                  </ol>
                </li>
              {/each}
            </ul>
          {/if}
        {/each}
      </FoldRow>
    {/if}
    <FoldRow title="Settings">
      <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
        <DayToDayFloatField
          saved={idle.emergency.settings.dayToDayFloat}
          effective={c.dayToDayFloat}
          defaultHint="the spending buffer, {months} of average expense"
          onsaved={() => invalidateAll()}
        />
        <EmergencyAccounts {idle} />
        <AccountProducts {idle} />
      </div>
    </FoldRow>
    <FoldRow title="Show the math">
      <dl class="mt-2 grid max-w-md grid-cols-[1fr_auto] gap-x-6 gap-y-1">
        <dt>
          Ready money (without emergency accounts{#if c.minimumReserved > 0}, above minimum balances{/if})
        </dt>
        <dd class="text-right"><Money value={c.readyMoney} /></dd>
        {#if c.minimumReserved > 0}
          <dt class="muted text-xs">Kept as minimum balances (never idle)</dt>
          <dd class="muted text-right text-xs"><Money value={c.minimumReserved} /></dd>
        {/if}
        {#each lines as line (line.label)}
          <dt>{line.label}</dt>
          <dd class="text-right">− <Money value={line.amount} /></dd>
        {/each}
        <dt class="border-t border-stone-900/10 pt-1 font-semibold dark:border-white/10">Idle</dt>
        <dd class="border-t border-stone-900/10 pt-1 text-right font-semibold dark:border-white/10">
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
                >{account.name}, {rateLabel(account.rate)}{#if account.product}
                  <span class="muted">(live rate of {account.product})</span>{:else if !account.rateFromNote}
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
      <p class="muted mt-1 text-xs">{idle.productRule}</p>
      {#if idle.productsIssue}<p class="muted mt-1 text-xs">{idle.productsIssue}</p>{/if}
      <p class="muted mt-1 text-xs">
        {idle.thresholdRule} This is about the rate only: goal money stays pooled, it does not need its own account.
      </p>
    </FoldRow>
  </div>
</Card>
