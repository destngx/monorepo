<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { FillResult } from '$lib/api/types';
  import Money from '$lib/components/Money.svelte';
  import FillBox from '$lib/fill/FillBox.svelte';
  import FilledMark from '$lib/fill/FilledMark.svelte';
  import { asNumber, asText } from '$lib/fill/fill';
  import { FillMarks } from '$lib/fill/marks.svelte';
  import { todayISO } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';
  import { Button, Modal } from '@libs/ui-kit';
  import { quantity } from './holdings';

  // Records a fund certificate trade in the sheet's History and Total Unit
  // (WM-FILL-10). AI reads an order confirmation or a sentence into it.
  interface Props {
    open: boolean;
    /** Fund codes held now, offered first. */
    codes: string[];
    onclose: () => void;
  }

  let { open, codes, onclose }: Props = $props();

  interface Form {
    code: string;
    date: string;
    side: 'buy' | 'sell';
    units: number | null;
    /** Money paid or received: only shown as the price per unit. */
    amount: number | null;
  }

  const empty = (): Form => ({ code: '', date: todayISO(), side: 'buy', units: null, amount: null });
  let form = $state<Form>(empty());
  let saving = $state(false);
  const marks = new FillMarks();
  $effect(() => {
    if (!open) {
      form = empty();
      marks.clear();
    }
  });

  const signedUnits = $derived(form.units === null ? null : form.side === 'sell' ? -form.units : form.units);
  const valid = $derived(form.code.trim() !== '' && form.date !== '' && form.units !== null && form.units > 0);

  function fill(result: FillResult) {
    const values = result.values;
    const units = asNumber(values.units);
    form = {
      ...form,
      code: 'code' in values ? asText(values.code).toUpperCase() : form.code,
      date: typeof values.date === 'string' ? values.date : form.date,
      side: units === null ? form.side : units < 0 ? 'sell' : 'buy',
      units: units === null ? form.units : Math.abs(units),
      amount: 'amount' in values ? asNumber(values.amount) : form.amount,
    };
    marks.take({ ...result, values: { ...values, code: asText(values.code).toUpperCase() } });
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (!valid || signedUnits === null) return;
    saving = true;
    try {
      const result = await api.recordFundTrade({ code: form.code.trim(), date: form.date, units: signedUnits });
      toasts.success(
        `${result.created ? 'Added' : 'Recorded'} ${result.code}: ${quantity(result.units)} units held now.`,
      );
      onclose();
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not record the trade');
    } finally {
      saving = false;
    }
  }
</script>

<Modal {open} title="Record a fund trade" {onclose} busy={saving}>
  <FillBox
    form="fund-trade"
    class="mb-4"
    current={() => ({ code: form.code, date: form.date, units: signedUnits, amount: form.amount })}
    onfill={fill}
  />
  <form id="fund-trade-form" class="grid grid-cols-1 gap-4 sm:grid-cols-2" onsubmit={save}>
    <div>
      <label class="label" for="trade-code">Fund code *<FilledMark show={marks.has('code', form.code)} /></label>
      <input id="trade-code" class="field uppercase" list="trade-codes" bind:value={form.code} required />
      <datalist id="trade-codes">
        {#each codes as code (code)}<option value={code}></option>{/each}
      </datalist>
    </div>
    <div>
      <label class="label" for="trade-date">Date *<FilledMark show={marks.has('date', form.date)} /></label>
      <input id="trade-date" type="date" class="field" max={todayISO()} bind:value={form.date} required />
    </div>
    <div>
      <label class="label" for="trade-side">Trade</label>
      <select id="trade-side" class="field" bind:value={form.side}>
        <option value="buy">Bought</option>
        <option value="sell">Sold</option>
      </select>
    </div>
    <div>
      <label class="label" for="trade-units">Units *<FilledMark show={marks.has('units', signedUnits)} /></label>
      <input
        id="trade-units"
        type="number"
        min="0.01"
        step="0.01"
        class="field"
        bind:value={form.units}
        required
        aria-describedby="trade-units-hint"
      />
      <p id="trade-units-hint" class="muted mt-1 text-xs">Rounded to 2 decimals, as the sheet keeps them.</p>
    </div>
    <div class="sm:col-span-2">
      <label class="label" for="trade-amount">Amount (VND)<FilledMark show={marks.has('amount', form.amount)} /></label>
      <input
        id="trade-amount"
        type="number"
        min="0"
        step="1"
        class="field"
        bind:value={form.amount}
        aria-describedby="trade-amount-hint"
      />
      <p id="trade-amount-hint" class="muted mt-1 text-xs">
        {#if form.amount && form.units}
          <Money value={form.amount / form.units} /> per unit. The sheet keeps units only.
        {:else}
          Optional, to check the price per unit. The sheet keeps units only.
        {/if}
      </p>
    </div>
  </form>
  {#snippet footer()}
    <Button disabled={saving} onclick={onclose}>Cancel</Button>
    <Button type="submit" form="fund-trade-form" variant="primary" busy={saving} busyText="Saving…" disabled={!valid}
      >Record</Button
    >
  {/snippet}
</Modal>
