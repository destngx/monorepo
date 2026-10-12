<script lang="ts">
  import { accountIcon, creditRemaining, creditUtilization, isCreditAccount } from '$lib/accounts';
  import type { Account, TrendPoint } from '$lib/api/types';
  import Sparkline from '$lib/charts/Sparkline.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatDate, formatPercent } from '$lib/format';
  import type { AccountHandler } from './handlers';

  interface Props {
    account: Account;
    /** Balance history, oldest first. */
    history: TrendPoint[];
    onedit: AccountHandler;
    ondelete: AccountHandler;
  }

  let { account, history, onedit, ondelete }: Props = $props();

  const utilization = $derived(isCreditAccount(account) ? creditUtilization(account) : null);
  const recent = $derived(history.slice(-60));
</script>

<li class="grid items-center gap-3 px-5 py-3 sm:grid-cols-[minmax(0,1fr)_8rem_12rem_auto]">
  <div class="flex min-w-0 items-center gap-3">
    <span class="grid size-9 shrink-0 place-items-center rounded-xl bg-slate-900/5 dark:bg-white/5"
      ><Icon name={accountIcon(account.type)} size={16} /></span
    >
    <div class="min-w-0">
      <p class="truncate font-medium">{account.name}</p>
      <p class="muted truncate text-xs">
        {account.type}{account.currency !== 'VND' ? ` · ${account.currency}` : ''}{account.dueDate
          ? ` · target ${formatDate(account.dueDate)}`
          : ''}{account.note && !isCreditAccount(account) ? ` · ${account.note}` : ''}
      </p>
    </div>
  </div>
  <div class="hidden sm:block">
    <Sparkline values={recent.map((p) => p.value)} width={128} height={32} label="{account.name} 60-day balance" />
  </div>
  <div class="text-right">
    <Money value={account.balance} currency={account.currency} class="font-medium" />
    {#if account.clearedBalance !== account.balance}
      <p class="muted text-xs">
        Cleared <Money value={account.clearedBalance} currency={account.currency} />
      </p>
    {/if}
    {#if isCreditAccount(account)}
      <p class="muted text-xs">Available <Money value={creditRemaining(account)} /></p>
    {/if}
    {#if utilization !== null}
      <div class="mt-1 flex items-center gap-2">
        <ProgressBar
          percent={utilization}
          color={utilization > 80 ? 'bg-orange-500' : 'bg-emerald-500'}
          label="Utilisation"
        />
        <span class="muted text-xs">{formatPercent(utilization, 0)}</span>
      </div>
    {:else if account.goalProgress !== null && account.goalAmount}
      <div class="mt-1 flex items-center gap-2">
        <ProgressBar percent={account.goalProgress} color="bg-indigo-500" label="Goal progress" />
        <span class="muted text-xs">{formatPercent(account.goalProgress, 0)}</span>
      </div>
    {/if}
  </div>
  <div class="flex justify-end">
    <button type="button" class="btn-ghost px-2" aria-label="Edit {account.name}" onclick={() => onedit(account)}
      ><Icon name="pencil" size={15} /></button
    >
    <button type="button" class="btn-ghost px-2" aria-label="Delete {account.name}" onclick={() => ondelete(account)}
      ><Icon name="trash" size={15} /></button
    >
  </div>
</li>
