<script lang="ts">
  import type { AccountGroup } from '$lib/accounts';
  import type { Account, TrendPoint } from '$lib/api/types';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import AccountRow from './AccountRow.svelte';
  import type { AccountHandler } from './handlers';

  interface Props {
    group: AccountGroup;
    accounts: Account[];
    open: boolean;
    /** Group total in the display currency, or null when it cannot be converted. */
    total: number | null;
    currency: string;
    histories: Record<string, TrendPoint[]>;
    ontoggle: () => void;
    onedit: AccountHandler;
    ondelete: AccountHandler;
  }

  let { group, accounts, open, total, currency, histories, ontoggle, onedit, ondelete }: Props = $props();
</script>

<section class="glass mt-4">
  <button
    type="button"
    class="flex w-full items-center justify-between gap-3 px-5 py-4 text-left"
    aria-expanded={open}
    onclick={ontoggle}
  >
    <span class="flex items-center gap-2 font-semibold">
      <Icon name="chevronDown" size={16} class="transition {open ? '' : '-rotate-90'}" />
      {group.label}
      <span class="muted text-xs font-normal">({accounts.length})</span>
    </span>
    {#if group.subtotal}<Money value={total} {currency} class="font-semibold" />{/if}
  </button>
  {#if open}
    <ul class="divide-y divide-slate-900/5 border-t border-slate-900/5 dark:divide-white/5 dark:border-white/5">
      {#each accounts as account (account.name)}
        <AccountRow {account} history={histories[account.name] ?? []} {onedit} {ondelete} />
      {/each}
    </ul>
  {/if}
</section>
