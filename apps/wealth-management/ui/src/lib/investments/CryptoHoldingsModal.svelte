<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { FillResult, ValuedHolding } from '$lib/api/types';
  import FillBox from '$lib/fill/FillBox.svelte';
  import { asNumber, asText, notesFor } from '$lib/fill/fill';
  import { toasts } from '$lib/state/toasts.svelte';
  import { Button, Modal } from '@libs/ui-kit';
  import { quantity as formatQuantity } from './holdings';

  // Sets coin balances in the Crypto tab (WM-FILL-11). AI reads exchange or
  // wallet screenshots into one row per coin; the user checks them and
  // saves them in one write.
  interface Props {
    open: boolean;
    /** The crypto holdings the sheet lists now. */
    holdings: ValuedHolding[];
    onclose: () => void;
  }

  let { open, holdings, onclose }: Props = $props();

  interface Row {
    id: number;
    platform: string;
    symbol: string;
    quantity: number | null;
    note: string;
    notes: string[];
  }

  let rows = $state<Row[]>([]);
  let saving = $state(false);
  let nextId = 0;

  $effect(() => {
    if (!open) rows = [];
  });

  const fold = (text: string) => text.trim().toLowerCase();
  const platforms = $derived([...new Set(holdings.map((holding) => holding.platform).filter(Boolean))]);
  const held = (row: Row) =>
    holdings.find(
      (holding) => fold(holding.platform) === fold(row.platform) && fold(holding.symbol) === fold(row.symbol),
    );
  const ready = $derived(
    rows.filter((row) => row.platform.trim() && row.symbol.trim() && row.quantity !== null && row.quantity >= 0),
  );

  const blank = (): Row => ({
    id: nextId++,
    platform: platforms[0] ?? '',
    symbol: '',
    quantity: null,
    note: '',
    notes: [],
  });

  function fill(result: FillResult) {
    rows = (result.rows ?? []).map((row, index) => ({
      id: nextId++,
      platform: asText(row.platform),
      symbol: asText(row.symbol).toUpperCase(),
      quantity: asNumber(row.quantity),
      note: asText(row.note),
      notes: ['platform', 'symbol', 'quantity', 'note']
        .flatMap((field) => notesFor(result.notes, field, index))
        .map((note) => note.message),
    }));
  }

  async function saveAll() {
    saving = true;
    try {
      const { holdings: saved } = await api.setCryptoHoldings(
        ready.map((row) => ({
          platform: row.platform.trim(),
          symbol: row.symbol.trim(),
          quantity: row.quantity!,
          note: row.note.trim() || undefined,
        })),
      );
      const added = saved.filter((holding) => holding.created).length;
      toasts.success(
        `Saved ${saved.length} ${saved.length === 1 ? 'balance' : 'balances'}${added ? `, ${added} new` : ''}.`,
      );
      onclose();
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not save the balances');
    } finally {
      saving = false;
    }
  }
</script>

<Modal {open} title="Update crypto balances" {onclose} busy={saving}>
  <FillBox form="crypto-holdings" rowsLabel="coin" onfill={fill} />
  <datalist id="crypto-platforms">
    {#each platforms as platform (platform)}<option value={platform}></option>{/each}
  </datalist>
  {#if rows.length}
    <ul class="mt-4 space-y-3" aria-label="Balances to save">
      {#each rows as row (row.id)}
        {@const now = held(row)}
        <li class="rounded-sm border border-edge p-3">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-[1fr_7rem_9rem_auto] sm:items-end">
            <div class="min-w-0">
              <label class="label" for="coin-platform-{row.id}">Platform</label>
              <input id="coin-platform-{row.id}" class="field" list="crypto-platforms" bind:value={row.platform} />
            </div>
            <div>
              <label class="label" for="coin-symbol-{row.id}">Coin</label>
              <input id="coin-symbol-{row.id}" class="field uppercase" bind:value={row.symbol} />
            </div>
            <div>
              <label class="label" for="coin-quantity-{row.id}">Quantity</label>
              <input
                id="coin-quantity-{row.id}"
                type="number"
                min="0"
                step="any"
                class="field"
                bind:value={row.quantity}
              />
            </div>
            <Button
              size="icon"
              icon="x"
              label="Drop the {row.symbol || 'new'} row"
              onclick={() => (rows = rows.filter((other) => other.id !== row.id))}
            />
          </div>
          <p class="muted mt-1 text-xs">
            {now ? `Now ${formatQuantity(now.quantity)} in the sheet.` : 'Not in the sheet yet: adds a row.'}
          </p>
          {#each row.notes as note, index (index)}
            <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">{note}</p>
          {/each}
        </li>
      {/each}
    </ul>
  {/if}
  <Button class="mt-3" size="sm" icon="plus" onclick={() => (rows = [...rows, blank()])}>Add a coin</Button>
  {#snippet footer()}
    <Button disabled={saving} onclick={onclose}>Cancel</Button>
    <Button variant="primary" busy={saving} busyText="Saving…" disabled={!ready.length} onclick={saveAll}
      >{ready.length > 1 ? `Save ${ready.length} balances` : 'Save'}</Button
    >
  {/snippet}
</Modal>
