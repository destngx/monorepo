<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { SafeToSpend } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Button, Icon } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { EXCLUSION_LABELS } from './bills';

  // What safe to spend counts (D33): ready money by account, each with a
  // ready-to-use switch, the accounts left out with the reason, and the
  // bills and card payments due before payday. An account with a minimum
  // balance to keep (D35) shows the part kept back. A switch saves to the
  // Preferences tab with the setting's ETag; one change runs at a time,
  // since each change moves the ETag.
  let { safe }: { safe: SafeToSpend } = $props();
  let pending = $state<string | null>(null);

  async function setReady(account: string, ready: boolean) {
    pending = account;
    try {
      await api.setAccountReady({ account, ready, etag: safe.readiness.etag });
      await invalidateAll();
    } finally {
      pending = null;
    }
  }

  const waits = (account: string) => pending !== null && pending !== account;
  const dueLabel = $derived(
    safe.untilIncome ? `Due before ${formatDate(safe.until)}` : `Due by ${formatDate(safe.until)}`,
  );
</script>

<div class="mt-4 grid grid-cols-1 gap-4 text-sm sm:grid-cols-2" data-testid="ready-money">
  <section aria-labelledby="ready-money-title">
    <h3 id="ready-money-title" class="text-xs font-semibold tracking-wide uppercase">Ready money</h3>
    <ul class="mt-1 divide-y divide-stone-900/5 dark:divide-white/5">
      {#each safe.readyAccounts as account (account.name)}
        <li class="flex items-center justify-between gap-2 py-1.5">
          <span class="min-w-0">
            <span class="block truncate">{account.name}</span>
            {#if account.reserved > 0}
              <span class="muted text-xs" data-testid="reserved"
                ><Money value={account.reserved} /> kept as its minimum, <Money value={account.spendable} /> counts</span
              >
            {/if}
          </span>
          <span class="flex items-center gap-2">
            <Money value={account.balance} />
            <Button
              size="sm"
              role="switch"
              aria-checked="true"
              label="{account.name} is ready to use"
              busy={pending === account.name}
              busyText="Saving…"
              disabled={waits(account.name)}
              success="{account.name} is left out of safe to spend."
              errorContext="Could not update {account.name}"
              onclick={() => setReady(account.name, false)}>Ready</Button
            >
          </span>
        </li>
      {:else}
        <li class="muted py-1.5">No account is ready to use.</li>
      {/each}
      {#each safe.excludedAccounts as account (account.name)}
        <li class="flex items-center justify-between gap-2 py-1.5">
          <span class="min-w-0">
            <span class="muted block truncate">{account.name}</span>
            <span class="muted text-xs">Left out: {EXCLUSION_LABELS[account.reason]}</span>
          </span>
          <span class="flex items-center gap-2">
            <span class="muted"><Money value={account.balance} /></span>
            {#if account.reason === 'not_ready'}
              <Button
                size="sm"
                role="switch"
                aria-checked="false"
                label="{account.name} is not ready to use"
                busy={pending === account.name}
                busyText="Saving…"
                disabled={waits(account.name)}
                success="{account.name} counts in safe to spend again."
                errorContext="Could not update {account.name}"
                onclick={() => setReady(account.name, true)}>Not ready</Button
              >
            {:else}
              <Badge>Crypto</Badge>
            {/if}
          </span>
        </li>
      {/each}
    </ul>
    {#if safe.unknownNotReady.length}
      <ul class="mt-2 space-y-1 text-xs" aria-label="Marked accounts to fix">
        {#each safe.unknownNotReady as name (name)}
          <li class="flex items-center justify-between gap-2 {STATUS.warning.ink}">
            <span class="flex items-start gap-1">
              <Icon name={STATUS.warning.icon} size={14} />“{name}” is marked not ready but is not an account; ignored.
            </span>
            <Button
              size="sm"
              label="Remove {name} from the not-ready accounts"
              busy={pending === name}
              busyText="Removing…"
              disabled={waits(name)}
              success="“{name}” removed."
              errorContext="Could not remove “{name}”"
              onclick={() => setReady(name, true)}>Remove</Button
            >
          </li>
        {/each}
      </ul>
    {/if}
  </section>
  <section aria-labelledby="due-before-title">
    <h3 id="due-before-title" class="text-xs font-semibold tracking-wide uppercase">{dueLabel}</h3>
    <ul class="mt-1 divide-y divide-stone-900/5 dark:divide-white/5">
      {#each safe.counted as occurrence (occurrence.name + occurrence.expectedDate)}
        <li class="flex justify-between gap-2 py-1.5">
          <span class="min-w-0 truncate"
            >{occurrence.name}, {formatDate(occurrence.expectedDate)}{#if occurrence.status === 'overdue'}, overdue{:else if occurrence.daysUntil < 0},
              late{/if}</span
          ><Money value={occurrence.remaining} />
        </li>
      {:else}
        <li class="muted py-1.5">No unpaid bills or card payments by then.</li>
      {/each}
    </ul>
  </section>
</div>
