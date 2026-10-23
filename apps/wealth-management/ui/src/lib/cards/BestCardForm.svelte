<script lang="ts">
  import { openOnPendingFill } from '$lib/fill/pending.svelte';
  import FillBox from '$lib/fill/FillBox.svelte';
  import FilledMark from '$lib/fill/FilledMark.svelte';
  import { applyFill, asText } from '$lib/fill/fill';
  import { FillMarks } from '$lib/fill/marks.svelte';
  import { api } from '$lib/api/endpoints';
  import type { CardSuggestion } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Button, Card } from '@libs/ui-kit';
  import Money from '$lib/components/Money.svelte';
  import { MCC_STATUS, parseCardPayment } from './cards';
  import MerchantFormModal from './MerchantFormModal.svelte';

  // WM-CARD-13: which held card earns the most on a payment about to be made.
  let form = $state({ payee: '', amount: '', mcc: '' });
  let busy = $state(false);
  let error = $state<string | null>(null);
  // Which field the last refused submit was about, marked invalid until the next submit.
  let invalid = $state<'amount' | 'mcc' | 'payee' | null>(null);
  let suggestions = $state<CardSuggestion[] | null>(null);
  let controller: AbortController | null = null;
  // Fields AI filled from the Describe it box (WM-FILL-6).
  const marks = new FillMarks();
  // Saves the payee's MCC for next time (WM-FILL-14).
  let mapping = $state<{ pattern: string; mcc: number | null } | null>(null);
  // Folded until used, or until a chat fill opens it (WM-FILL-7).
  let unfolded = $state(false);
  openOnPendingFill('best-card', () => (unfolded = true));

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
  bind:open={unfolded}
  title="Best card for a payment"
  subtitle="What the payment would add to each card's current period, caps and spend tiers included"
>
  <FillBox
    form="best-card"
    class="mb-3"
    onfill={(result) => {
      form = applyFill(form, result.values, { amount: asText, mcc: asText });
      marks.take({ ...result, values: { ...result.values, amount: form.amount, mcc: form.mcc } });
    }}
  />
  <form class="grid grid-cols-1 gap-3 sm:grid-cols-[2fr_1fr_1fr_auto] sm:items-end" onsubmit={submit} novalidate>
    <div>
      <label class="label" for="best-card-payee">Payee<FilledMark show={marks.has('payee', form.payee)} /></label>
      <input
        id="best-card-payee"
        class="field"
        bind:value={form.payee}
        autocomplete="off"
        aria-invalid={invalid === 'payee' || undefined}
      />
    </div>
    <div>
      <label class="label" for="best-card-amount">Amount<FilledMark show={marks.has('amount', form.amount)} /></label>
      <input
        id="best-card-amount"
        class="field"
        inputmode="numeric"
        bind:value={form.amount}
        aria-invalid={invalid === 'amount' || undefined}
      />
    </div>
    <div>
      <label class="label" for="best-card-mcc">MCC (if known)<FilledMark show={marks.has('mcc', form.mcc)} /></label>
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
          The MCC is unknown, so only fallback rules apply. Map the payee or enter the MCC.
        {:else}
          MCC {String(mcc.code).padStart(4, '0')} ({MCC_STATUS[mcc.status].label}).
        {/if}
        {#if form.payee.trim() && (mcc.status === 'unknown' || form.mcc.trim())}
          <button
            type="button"
            class="link ml-1"
            onclick={() =>
              (mapping = { pattern: form.payee.trim(), mcc: mcc.status === 'unknown' ? null : (mcc.code ?? null) })}
            >Save the payee's MCC</button
          >
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

<MerchantFormModal open={mapping !== null} initial={mapping ?? undefined} onclose={() => (mapping = null)} />
