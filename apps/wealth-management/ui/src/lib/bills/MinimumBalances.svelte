<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { SafeToSpend } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import Money from '$lib/components/Money.svelte';
  import { formatDate } from '$lib/format';
  import { STATUS } from '$lib/status';
  import { minimumRows, minimumStatus } from './bills';
  import { parseFloatDraft } from './idleCash';

  // Minimum balance to keep (D35): a bank may reward an account whose
  // average daily balance over the month stays at or above a minimum. Each
  // account can keep one, with a note; that money is never ready to spend
  // nor idle, and this month's average is tracked against it. Saved to the
  // Preferences tab with the setting's ETag; one change runs at a time,
  // since each change moves the ETag. The amount input starts empty so no
  // amount shows unmasked.
  let { safe }: { safe: SafeToSpend } = $props();

  const rows = $derived(minimumRows(safe));
  let editing = $state<string | null>(null);
  let pending = $state<string | null>(null);
  let draft = $state('');
  let noteDraft = $state('');
  let touched = $state(false);
  const parsed = $derived(parseFloatDraft(draft));
  const problem = $derived(touched && draft.trim() !== '' ? parsed.error : null);

  function edit(account: string, note: string) {
    editing = editing === account ? null : account;
    draft = '';
    noteDraft = note;
    touched = false;
  }

  async function save(account: string, minimum: number | null, note: string) {
    pending = account;
    try {
      await api.setAccountMinimum({ account, minimum, note: note.trim(), etag: safe.minimumBalance.etag });
      editing = null;
      draft = '';
      await invalidateAll();
    } finally {
      pending = null;
    }
  }

  const waits = (account: string) => pending !== null && pending !== account;
  const inputId = (account: string) => `minimum-${account.replace(/\W+/g, '-').toLowerCase()}`;
</script>

<section class="mt-4 text-sm" aria-labelledby="minimum-balances-title" data-testid="minimum-balances">
  <h3 id="minimum-balances-title" class="text-xs font-semibold tracking-wide uppercase">Minimum balance to keep</h3>
  <p class="muted mt-1 text-xs">
    For a bank reward on the average monthly balance. The minimum is kept back from ready money and is never idle: the
    reward depends on the whole month's average, so a dip late in the month cannot be made up.
    {#if safe.reserved > 0}Kept back now: <Money value={safe.reserved} />.{/if}
  </p>
  <ul class="mt-1 divide-y divide-slate-900/5 dark:divide-white/5">
    {#each rows as row (row.account)}
      {@const track = row.track}
      <li class="py-1.5" data-testid="minimum-row">
        <div class="flex flex-wrap items-center justify-between gap-2">
          <span class="min-w-0">
            <span class="block truncate">{row.account}</span>
            {#if track}
              <span class="muted block text-xs">
                Keeps <Money value={track.minimum} />{#if track.note}, {track.note}{/if}. Average so far
                <Money value={track.averageSoFar} />, month end at today's balance <Money
                  value={track.projectedAverage}
                />.
                {#if track.status === 'at_risk'}Hold <Money value={track.requiredBalance} /> until {formatDate(
                    track.monthEnd,
                  )}: <Money value={track.extraNeeded} /> more.{/if}
              </span>
            {/if}
          </span>
          <span class="flex items-center gap-2">
            {#if track}
              {@const status = minimumStatus(track)}
              <Badge status={status.tone}>{status.label}</Badge>
            {/if}
            <Button
              size="sm"
              variant="ghost"
              aria-expanded={editing === row.account}
              aria-controls={inputId(row.account)}
              label={track ? `Change the minimum of ${row.account}` : `Set a minimum for ${row.account}`}
              disabled={waits(row.account)}
              onclick={() => edit(row.account, track?.note ?? '')}>{track ? 'Change' : 'Set minimum'}</Button
            >
          </span>
        </div>
        {#if editing === row.account}
          <div class="mt-1 flex flex-wrap items-center gap-2">
            <label class="sr-only" for={inputId(row.account)}>Minimum for {row.account} in VND</label>
            <input
              id={inputId(row.account)}
              class="field w-40"
              inputmode="numeric"
              autocomplete="off"
              placeholder="Minimum in VND"
              aria-invalid={problem ? true : undefined}
              bind:value={draft}
              oninput={() => (touched = true)}
            />
            <label class="sr-only" for="{inputId(row.account)}-note">Note for {row.account}</label>
            <input
              id="{inputId(row.account)}-note"
              class="field w-48"
              autocomplete="off"
              maxlength="200"
              placeholder="Note, e.g. monthly reward"
              bind:value={noteDraft}
            />
            <Button
              size="sm"
              variant="primary"
              busy={pending === row.account}
              busyText="Saving…"
              disabled={parsed.value === null || waits(row.account)}
              success={parsed.value === 0
                ? `${row.account} keeps no minimum.`
                : `Minimum for ${row.account} saved to your sheet.`}
              errorContext="Could not save the minimum for {row.account}"
              onclick={() => save(row.account, parsed.value, noteDraft)}>Save</Button
            >
            {#if track}
              <Button
                size="sm"
                busyText="Removing…"
                disabled={waits(row.account)}
                success="{row.account} keeps no minimum."
                errorContext="Could not remove the minimum for {row.account}"
                onclick={() => save(row.account, null, '')}>Remove</Button
              >
            {/if}
          </div>
          {#if problem}
            <p class="mt-1 text-xs {STATUS.warning.ink}" role="alert">{problem}</p>
          {/if}
        {/if}
      </li>
    {:else}
      <li class="muted py-1.5">No account can keep a minimum.</li>
    {/each}
  </ul>
  {#if safe.unknownMinimum.length}
    <ul class="mt-2 space-y-1 text-xs" aria-label="Minimums to fix">
      {#each safe.unknownMinimum as name (name)}
        <li class="flex items-center justify-between gap-2 {STATUS.warning.ink}">
          <span class="flex items-start gap-1">
            <Icon name={STATUS.warning.icon} size={14} />“{name}” has a minimum but is not an account; ignored.
          </span>
          <Button
            size="sm"
            label="Remove the minimum of {name}"
            busy={pending === name}
            busyText="Removing…"
            disabled={waits(name)}
            success="“{name}” removed."
            errorContext="Could not remove “{name}”"
            onclick={() => save(name, null, '')}>Remove</Button
          >
        </li>
      {/each}
    </ul>
  {/if}
</section>
