<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { BudgetLine, FillResult } from '$lib/api/types';
  import Money from '$lib/components/Money.svelte';
  import FillBox from '$lib/fill/FillBox.svelte';
  import { asNumber, notesFor } from '$lib/fill/fill';
  import { formatMonth } from '$lib/format';
  import { toasts } from '$lib/state/toasts.svelte';
  import { Button, Modal } from '@libs/ui-kit';

  // Several limits from one sentence (WM-FILL-5): AI reads the request into
  // rows, the user checks them and saves each through the limit path
  // (WM-BUD-6). A row the engine could not match stays with its note.
  interface Props {
    open: boolean;
    month: string;
    /** The month's expense lines: the categories a limit can go on. */
    lines: BudgetLine[];
    onclose: () => void;
  }

  let { open, month, lines, onclose }: Props = $props();

  interface Row {
    id: number;
    category: string;
    limit: number | null;
    scope: 'month' | 'year';
    notes: string[];
    saved: boolean;
  }

  let rows = $state<Row[]>([]);
  let saving = $state(false);
  let nextId = 0;

  $effect(() => {
    if (!open) rows = [];
  });

  const known = $derived(new Set(lines.map((line) => line.category)));
  const current = $derived(new Map(lines.map((line) => [line.category, line.limit])));
  const pending = $derived(rows.filter((row) => !row.saved));
  const ready = $derived(pending.filter((row) => known.has(row.category) && row.limit !== null && row.limit >= 0));

  function fill(result: FillResult) {
    rows = (result.rows ?? []).map((row, index) => ({
      id: nextId++,
      category: typeof row.category === 'string' ? row.category : '',
      limit: asNumber(row.limit),
      scope: row.scope === 'year' ? 'year' : 'month',
      notes: notesFor(result.notes, 'category', index)
        .concat(notesFor(result.notes, 'limit', index), notesFor(result.notes, 'scope', index))
        .map((note) => note.message),
      saved: false,
    }));
  }

  async function saveAll() {
    saving = true;
    let saved = 0;
    try {
      for (const row of ready) {
        await api.updateBudgetLimit({
          year: Number(month.slice(0, 4)),
          category: row.category,
          // An empty month writes the yearly limit column.
          month: row.scope === 'month' ? month : undefined,
          limit: row.limit!,
        });
        row.saved = true;
        saved++;
      }
      toasts.push('success', `Saved ${saved} ${saved === 1 ? 'limit' : 'limits'}.`);
      if (!rows.some((row) => !row.saved)) onclose();
    } catch (error) {
      toasts.error(error, saved ? `Saved ${saved}, then could not save the next limit` : 'Could not save the limits');
    } finally {
      saving = false;
      if (saved) await invalidateAll();
    }
  }
</script>

<Modal {open} title="Set limits for {formatMonth(month)}" {onclose} busy={saving}>
  <FillBox form="budget-limits" rowsLabel="limit" current={() => ({ month })} onfill={fill} />
  {#if rows.length}
    <ul class="mt-4 space-y-3" aria-label="Limits to save">
      {#each rows as row (row.id)}
        <li class="rounded-sm border border-edge p-3 {row.saved ? 'opacity-60' : ''}">
          <div class="grid grid-cols-1 gap-3 sm:grid-cols-[1fr_9rem_8rem_auto] sm:items-end">
            <div class="min-w-0">
              <label class="label" for="limit-category-{row.id}">Category</label>
              <select
                id="limit-category-{row.id}"
                class="field"
                bind:value={row.category}
                disabled={row.saved}
                aria-invalid={!known.has(row.category)}
              >
                {#if !known.has(row.category)}<option value={row.category}>{row.category || 'Pick a category'}</option
                  >{/if}
                {#each lines as line (line.category)}<option value={line.category}>{line.category}</option>{/each}
              </select>
              {#if known.has(row.category)}
                <p class="muted mt-1 text-xs">Now <Money value={current.get(row.category) ?? 0} /></p>
              {/if}
            </div>
            <div>
              <label class="label" for="limit-value-{row.id}">New limit (VND)</label>
              <input
                id="limit-value-{row.id}"
                type="number"
                min="0"
                step="1"
                class="field"
                bind:value={row.limit}
                disabled={row.saved}
              />
            </div>
            <div>
              <label class="label" for="limit-scope-{row.id}">Applies to</label>
              <select id="limit-scope-{row.id}" class="field" bind:value={row.scope} disabled={row.saved}>
                <option value="month">This month</option>
                <option value="year">Every month</option>
              </select>
            </div>
            {#if row.saved}
              <span class="text-xs font-medium text-emerald-700 dark:text-emerald-300">Saved</span>
            {:else}
              <Button
                size="icon"
                icon="x"
                label="Drop the {row.category || 'unmatched'} limit"
                onclick={() => (rows = rows.filter((other) => other.id !== row.id))}
              />
            {/if}
          </div>
          {#each row.notes as note, index (index)}
            <p class="mt-1 text-xs text-amber-700 dark:text-amber-300">{note}</p>
          {/each}
        </li>
      {/each}
    </ul>
  {/if}
  {#snippet footer()}
    <Button disabled={saving} onclick={onclose}>Cancel</Button>
    <Button variant="primary" busy={saving} busyText="Saving…" disabled={!ready.length} onclick={saveAll}
      >{ready.length > 1 ? `Save ${ready.length} limits` : 'Save'}</Button
    >
  {/snippet}
</Modal>
