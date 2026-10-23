<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { CardSuggestion } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Button, Card } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { MCC_STATUS, parseCardPayment } from './cards';

  // WM-CARD-13: which held card earns the most on a payment about to be made.
  let form = $state({ payee: '', amount: '', mcc: '' });
  let busy = $state(false);
  let error = $state<string | null>(null);
  // Which field the last refused submit was about, marked invalid until the next submit.
  let invalid = $state<'amount' | 'mcc' | 'payee' | null>(null);
  let suggestions = $state<CardSuggestion[] | null>(null);
  let controller: AbortController | null = null;

  async function submit(event: SubmitEvent) {
    event.preventDefault();
    const parsed = parseCardPayment(form);
    if ('error' in parsed) {
      error = parsed.error;
      invalid = parsed.field;
      return;
    }
    invalid = null;
    controller?.abort();
    const current = new AbortController();
    controller = current;
    busy = true;
    error = null;
    try {
      const result = await api.suggestCards(parsed.query);
      if (!current.signal.aborted) suggestions = result;
    } catch (failure) {
      if (!current.signal.aborted) {
        suggestions = null;
        error = failure instanceof Error ? `Suggestion failed: ${failure.message}` : 'Suggestion failed.';
      }
    } finally {
      if (controller === current) busy = false;
    }
  }
</script>

<Card
  foldable
  open={false}
  title="Best card for a payment"
  subtitle="What the payment would add to each card's current period, caps and spend tiers included"
>
  <form class="grid grid-cols-1 gap-3 sm:grid-cols-[2fr_1fr_1fr_auto] sm:items-end" onsubmit={submit} novalidate>
    <div>
      <label class="label" for="best-card-payee">Payee</label>
      <input
        id="best-card-payee"
        class="field"
        bind:value={form.payee}
        autocomplete="off"
        aria-invalid={invalid === 'payee' || undefined}
      />
    </div>
    <div>
      <label class="label" for="best-card-amount">Amount</label>
      <input
        id="best-card-amount"
        class="field"
        inputmode="numeric"
        bind:value={form.amount}
        aria-invalid={invalid === 'amount' || undefined}
      />
    </div>
    <div>
      <label class="label" for="best-card-mcc">MCC (if known)</label>
      <input
        id="best-card-mcc"
        class="field"
        inputmode="numeric"
        maxlength="4"
        bind:value={form.mcc}
        aria-invalid={invalid === 'mcc' || undefined}
      />
    </div>
    <Button type="submit" variant="primary" {busy} busyText="Checking…">Suggest</Button>
  </form>
  {#if error}<p role="alert" class="mt-3 text-sm text-orange-600 dark:text-orange-400">{error}</p>{/if}
  {#if suggestions}
    {#if suggestions.length === 0}
      <p class="muted mt-4 text-sm">No held card has programme terms in effect today.</p>
    {:else}
      {@const mcc = suggestions[0].mcc}
      <p class="muted mt-4 text-xs">
        {#if mcc.status === 'unknown'}
          The MCC is unknown, so only fallback rules apply. Map the payee in the Merchants tab or enter the MCC.
        {:else}
          MCC {String(mcc.code).padStart(4, '0')} ({MCC_STATUS[mcc.status].label}).
        {/if}
      </p>
      <ol class="mt-2 divide-y divide-stone-900/5 text-sm dark:divide-white/5">
        {#each suggestions as suggestion, index (suggestion.card)}
          <li class="flex flex-wrap items-center justify-between gap-3 py-2.5">
            <div>
              <p class="font-medium">
                {suggestion.card}
                {#if index === 0 && suggestion.fitsLimit}<Badge status="good">Best</Badge>{/if}
                {#if !suggestion.fitsLimit}<Badge status="critical">Over the bank limit</Badge>{/if}
              </p>
              <p class="muted text-xs">
                {suggestion.bank} · {suggestion.rule ? `rule ${suggestion.rule}` : 'no rule applies'} · room
                <Money value={suggestion.remainingLimit} />
              </p>
            </div>
            <div class="text-right">
              <Money value={suggestion.cashback.total} />
              {#if suggestion.cashback.estimated !== 0}
                <p class="muted text-xs"><Money value={suggestion.cashback.confirmed} /> confirmed</p>
              {/if}
            </div>
          </li>
        {/each}
      </ol>
    {/if}
  {/if}
</Card>
