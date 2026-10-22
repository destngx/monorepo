<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { IdleCash } from '$lib/api/types';
  import { Button, Icon } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { STATUS } from '$lib/status';
  import { rateLabel } from './idleCash';

  // Emergency accounts (D34): the everyday ready accounts and the emergency
  // ones, each with an emergency switch saved to the Preferences tab with the
  // setting's ETag. One change runs at a time, since each change moves the
  // ETag. Emergency accounts cover the emergency fund first and are never idle.
  let { idle }: { idle: IdleCash } = $props();
  let pending = $state<string | null>(null);

  async function setEmergency(account: string, emergency: boolean) {
    pending = account;
    try {
      await api.setEmergencyAccount({ account, emergency, etag: idle.emergency.settings.etag });
      await invalidateAll();
    } finally {
      pending = null;
    }
  }

  const waits = (account: string) => pending !== null && pending !== account;
</script>

<section aria-labelledby="emergency-accounts-title" data-testid="emergency-accounts">
  <h3 id="emergency-accounts-title" class="text-xs font-semibold tracking-wide uppercase">Emergency accounts</h3>
  <p class="muted text-xs">
    Mark the accounts that hold your emergency fund: they cover its target first and are never called idle.
  </p>
  <ul class="mt-1 divide-y divide-stone-900/5 dark:divide-white/5">
    {#each idle.emergency.accounts as account (account.name)}
      <li class="flex items-center justify-between gap-2 py-1.5">
        <span class="min-w-0 truncate">{account.name}, {rateLabel(account.rate)}</span>
        <span class="flex items-center gap-2">
          <Money value={account.balance} />
          <Button
            size="sm"
            role="switch"
            aria-checked="true"
            label="{account.name} holds the emergency fund"
            busy={pending === account.name}
            busyText="Saving…"
            disabled={waits(account.name)}
            success="{account.name} is no longer an emergency account."
            errorContext="Could not update {account.name}"
            onclick={() => setEmergency(account.name, false)}>Emergency</Button
          >
        </span>
      </li>
    {/each}
    {#each idle.accounts as account (account.name)}
      <li class="flex items-center justify-between gap-2 py-1.5">
        <span class="min-w-0 truncate">{account.name}</span>
        <span class="flex items-center gap-2">
          <Money value={account.balance} />
          <Button
            size="sm"
            role="switch"
            aria-checked="false"
            label="{account.name} is an everyday account"
            busy={pending === account.name}
            busyText="Saving…"
            disabled={waits(account.name)}
            success="{account.name} now holds the emergency fund."
            errorContext="Could not update {account.name}"
            onclick={() => setEmergency(account.name, true)}>Everyday</Button
          >
        </span>
      </li>
    {:else}
      {#if !idle.emergency.accounts.length}<li class="muted py-1.5">No ready account to mark.</li>{/if}
    {/each}
  </ul>
  {#if idle.emergency.unknown.length}
    <ul class="mt-2 space-y-1 text-xs" aria-label="Emergency accounts to fix">
      {#each idle.emergency.unknown as name (name)}
        <li class="flex items-center justify-between gap-2 {STATUS.warning.ink}">
          <span class="flex items-start gap-1">
            <Icon name={STATUS.warning.icon} size={14} />“{name}” is marked as an emergency account but is not an
            account; ignored.
          </span>
          <Button
            size="sm"
            label="Remove {name} from the emergency accounts"
            busy={pending === name}
            busyText="Removing…"
            disabled={waits(name)}
            success="“{name}” removed."
            errorContext="Could not remove “{name}”"
            onclick={() => setEmergency(name, false)}>Remove</Button
          >
        </li>
      {/each}
    </ul>
  {/if}
</section>
