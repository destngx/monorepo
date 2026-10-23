<script lang="ts">
  import {
    accountIcon,
    historySpan,
    isCreditAccount,
    noteAmount,
    walletEarningPerDay,
    type CreditStanding,
  } from '$lib/accounts';
  import type { Account, FlexibleSaving, TrendPoint } from '$lib/api/types';
  import { rateLabel } from '$lib/bills/idleCash';
  import Sparkline from '$lib/charts/Sparkline.svelte';
  import { Button, Icon } from '@libs/ui-kit';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatDate, formatPercent } from '$lib/format';
  import type { AccountHandler } from './handlers';

  interface Props {
    account: Account;
    /** Balance history, oldest first: up to 12 months, from where the ledger starts; the sparkline shows all of it. */
    history: TrendPoint[];
    /**
     * A card account's credit line: its standing, null when it has no limit
     * (the note's amount is shown instead), undefined while it loads.
     */
    credit?: CreditStanding | null;
    /** The wallet product a wallet earning balance account is linked to (D41), for its live rate. */
    wallet?: FlexibleSaving;
    onedit: AccountHandler;
    ondelete: AccountHandler;
  }

  let { account, history, credit, wallet, onedit, ondelete }: Props = $props();

  const noted = $derived(isCreditAccount(account) && credit === null ? noteAmount(account.note) : null);
</script>

<li
  class="grid grid-cols-1 items-center transition-colors hover:bg-stone-900/[0.02] dark:hover:bg-white/[0.03] gap-3 px-5 py-3 sm:grid-cols-[minmax(0,1fr)_10rem_12rem_auto] lg:grid-cols-[minmax(0,1fr)_18rem_12rem_auto]"
>
  <div class="flex min-w-0 items-center gap-3">
    <span class="grid size-9 shrink-0 place-items-center rounded-xl bg-stone-900/5 dark:bg-white/5"
      ><Icon name={accountIcon(account.type)} size={16} /></span
    >
    <div class="min-w-0">
      <p class="truncate font-medium">{account.name}</p>
      <p class="muted truncate text-xs">
        {account.type}{account.currency !== 'VND' ? ` · entered in ${account.currency}` : ''}{account.dueDate
          ? ` · target ${formatDate(account.dueDate)}`
          : ''}{#if account.note && !isCreditAccount(account)}
          · <MaskedText text={account.note} />{/if}
      </p>
    </div>
  </div>
  <div class="hidden sm:block">
    <Sparkline
      values={history.map((p) => p.value)}
      width={288}
      height={32}
      fluid
      label="{account.name} balance, {historySpan(history)}"
    />
  </div>
  <div class="text-right">
    <Money value={account.balance} class="font-medium" />
    {#if account.clearedBalance !== account.balance}
      <p class="muted text-xs">
        Cleared <Money value={account.clearedBalance} />
      </p>
    {/if}
    {#if credit}
      <p class="muted text-xs">Available <Money value={credit.available} /> of <Money value={credit.limit} /></p>
      <div class="mt-1 flex items-center gap-2">
        <ProgressBar
          percent={credit.utilization}
          color={credit.utilization > 80 ? 'bg-orange-500' : 'bg-emerald-500'}
          label="Utilisation"
        />
        <span class="muted text-xs">{formatPercent(credit.utilization, 0)}</span>
      </div>
      {#if credit.sharedWith.length}
        <p class="muted text-xs">Limit shared with {credit.sharedWith.join(', ')}</p>
      {/if}
    {:else if noted !== null}
      <p class="muted text-xs">Available <Money value={noted} />, from the note</p>
    {/if}
    {#if wallet}
      <p class="muted text-xs" title="{wallet.provider} {wallet.product}">
        Earns <span class="tabular">{rateLabel(wallet.rate)}</span> a year{wallet.stale ? ' (stale)' : ''}
      </p>
      {#if account.balance > 0}
        <p class="muted text-xs">About <Money value={walletEarningPerDay(account.balance, wallet.rate)} /> a day</p>
      {/if}
    {/if}
  </div>
  <div class="flex justify-end">
    <Button size="icon" icon="pencil" label="Edit {account.name}" onclick={() => onedit(account)} />
    <Button size="icon" icon="trash" label="Delete {account.name}" onclick={() => ondelete(account)} />
  </div>
</li>
