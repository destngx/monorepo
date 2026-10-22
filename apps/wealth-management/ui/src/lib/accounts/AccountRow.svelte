<script lang="ts">
  import { accountIcon, isCreditAccount, noteAmount, type CreditStanding } from '$lib/accounts';
  import type { Account, TrendPoint } from '$lib/api/types';
  import Sparkline from '$lib/charts/Sparkline.svelte';
  import Button from '$lib/components/Button.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatDate, formatPercent } from '$lib/format';
  import type { AccountHandler } from './handlers';

  interface Props {
    account: Account;
    /** Balance history, oldest first. */
    history: TrendPoint[];
    /**
     * A card account's credit line: its standing, null when it has no limit
     * (the note's amount is shown instead), undefined while it loads.
     */
    credit?: CreditStanding | null;
    onedit: AccountHandler;
    ondelete: AccountHandler;
  }

  let { account, history, credit, onedit, ondelete }: Props = $props();

  const noted = $derived(isCreditAccount(account) && credit === null ? noteAmount(account.note) : null);
  const recent = $derived(history.slice(-60));
</script>

<li
  class="grid items-center transition-colors hover:bg-slate-900/[0.02] dark:hover:bg-white/[0.03] gap-3 px-5 py-3 sm:grid-cols-[minmax(0,1fr)_8rem_12rem_auto]"
>
  <div class="flex min-w-0 items-center gap-3">
    <span class="grid size-9 shrink-0 place-items-center rounded-xl bg-slate-900/5 dark:bg-white/5"
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
    <Sparkline values={recent.map((p) => p.value)} width={128} height={32} label="{account.name} 60-day balance" />
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
  </div>
  <div class="flex justify-end">
    <Button size="icon" icon="pencil" label="Edit {account.name}" onclick={() => onedit(account)} />
    <Button size="icon" icon="trash" label="Delete {account.name}" onclick={() => ondelete(account)} />
  </div>
</li>
