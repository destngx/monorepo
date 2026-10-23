<script lang="ts">
  import type { AccountGroup, CreditStanding } from '$lib/accounts';
  import type { Account, FlexibleSaving, TrendPoint } from '$lib/api/types';
  import { Icon } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import AccountRow from './AccountRow.svelte';
  import type { AccountHandler } from './handlers';

  interface Props {
    group: AccountGroup;
    accounts: Account[];
    open: boolean;
    /** Group total in VND. */
    total: number;
    histories: Record<string, TrendPoint[]>;
    /** Card accounts' credit standings; null while the card set-up loads. */
    credits: Map<string, CreditStanding> | null;
    /** Linked wallet products by account name (D41); empty until the rates and links load. */
    wallets: Map<string, FlexibleSaving>;
    ontoggle: () => void;
    onedit: AccountHandler;
    ondelete: AccountHandler;
  }

  let { group, accounts, open, total, histories, credits, wallets, ontoggle, onedit, ondelete }: Props = $props();
</script>

<section class="panel mt-4">
  <!-- The total sits beside the toggle: an amount is its own mask button, never nested in another. -->
  <div class="flex items-center gap-3 pr-5">
    <button
      type="button"
      class="row-link flex min-w-0 flex-1 items-center gap-2 rounded-2xl px-5 py-4 text-left font-semibold focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent-500"
      aria-expanded={open}
      title={open ? `Collapse ${group.label}` : `Expand ${group.label}`}
      onclick={ontoggle}
    >
      <Icon name="chevronDown" size={16} class="transition {open ? '' : '-rotate-90'}" />
      {group.label}
      <span class="muted text-xs font-normal">({accounts.length})</span>
    </button>
    {#if group.subtotal}<Money value={total} class="font-semibold" />{/if}
  </div>
  {#if open}
    <ul class="divide-y divide-stone-900/5 border-t border-stone-900/5 dark:divide-white/5 dark:border-white/5">
      {#each accounts as account (account.name)}
        <AccountRow
          {account}
          history={histories[account.name] ?? []}
          credit={credits ? (credits.get(account.name) ?? null) : undefined}
          wallet={wallets.get(account.name)}
          {onedit}
          {ondelete}
        />
      {/each}
    </ul>
  {/if}
</section>
