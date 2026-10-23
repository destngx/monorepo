<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { api } from '$lib/api/endpoints';
  import type { CardConfig, CardHolding, CardInput, FillResult } from '$lib/api/types';
  import FillBox from '$lib/fill/FillBox.svelte';
  import FilledMark from '$lib/fill/FilledMark.svelte';
  import { applyFill, asNumber, asText } from '$lib/fill/fill';
  import { FillMarks } from '$lib/fill/marks.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { Button, Modal } from '@libs/ui-kit';

  // Adds or edits a Cards row (WM-FILL-12). AI reads a statement or a
  // sentence into it; the engine refuses a row the strict read would reject.
  interface Props {
    open: boolean;
    /** The card to edit by name; absent adds one. */
    editing?: string;
    onclose: () => void;
  }

  let { open, editing, onclose }: Props = $props();

  interface Form {
    name: string;
    program: string;
    keyword: string;
    expiry: string;
    statementDay: number | null;
    dueDay: number | null;
    last4: string;
    default: boolean;
    closedFrom: string;
    limit: number | null;
  }

  const empty = (): Form => ({
    name: '',
    program: '',
    keyword: '',
    expiry: '',
    statementDay: null,
    dueDay: null,
    last4: '',
    default: false,
    closedFrom: '',
    limit: null,
  });
  const fromHolding = (card: CardHolding): Form => ({
    ...empty(),
    name: card.name,
    program: card.program,
    keyword: card.keyword,
    expiry: card.expiry,
    statementDay: card.statementDay || null,
    dueDay: card.dueDay || null,
    last4: card.last4 ?? '',
    default: card.default ?? false,
    closedFrom: card.closedFrom ?? '',
  });

  let form = $state<Form>(empty());
  let config = $state<CardConfig | null>(null);
  let loadError = $state('');
  let saving = $state(false);
  const marks = new FillMarks();

  $effect(() => {
    if (!open) {
      form = empty();
      marks.clear();
      return;
    }
    const name = editing;
    loadError = '';
    api.cardPrograms().then(
      (loaded) => {
        config = loaded;
        const card = name && loaded.cards.find((held) => held.name === name);
        if (card) form = fromHolding(card);
      },
      (error: unknown) => (loadError = error instanceof Error ? error.message : String(error)),
    );
  });

  const programs = $derived(config?.programs.map((program) => program.name) ?? []);
  const valid = $derived(form.name.trim() !== '' && form.program.trim() !== '');

  function fill(result: FillResult) {
    form = applyFill(form, result.values, {
      expiry: asText,
      last4: asText,
      keyword: asText,
      closedFrom: asText,
      statementDay: asNumber,
      dueDay: asNumber,
      limit: asNumber,
      default: (value) => value === true,
    });
    marks.take(result);
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (!valid) return;
    saving = true;
    const input: CardInput = {
      name: form.name.trim(),
      program: form.program.trim(),
      keyword: form.keyword.trim(),
      expiry: form.expiry.trim() || undefined,
      statementDay: form.statementDay ?? 0,
      dueDay: form.dueDay ?? 0,
      default: form.default || undefined,
      closedFrom: form.closedFrom || undefined,
      last4: form.last4.trim() || undefined,
      limit: form.limit ?? undefined,
    };
    try {
      if (editing) await api.updateCard(editing, input);
      else await api.createCard(input);
      toasts.success(editing ? `Saved ${input.name}.` : `Added ${input.name}.`);
      onclose();
      if (editing && editing !== input.name) {
        await goto(`${base}/accounts/credit-cards/${encodeURIComponent(input.name)}`, { invalidateAll: true });
      } else {
        await invalidateAll();
      }
    } catch (error) {
      toasts.error(error, 'Could not save the card');
    } finally {
      saving = false;
    }
  }
</script>

<Modal {open} title={editing ? `Edit ${editing}` : 'Add a credit card'} {onclose} busy={saving}>
  <FillBox form="card" class="mb-4" current={() => ({ ...form })} onfill={fill} />
  {#if loadError}
    <p class="mb-3 text-sm text-rose-600 dark:text-rose-400" role="alert">
      Could not read the card tabs: {loadError}
    </p>
  {/if}
  <form id="card-form" class="grid grid-cols-1 gap-4 sm:grid-cols-2" onsubmit={save}>
    <div>
      <label class="label" for="card-name">Card *<FilledMark show={marks.has('name', form.name)} /></label>
      <input id="card-name" class="field" bind:value={form.name} required />
    </div>
    <div>
      <label class="label" for="card-program">Programme *<FilledMark show={marks.has('program', form.program)} /></label
      >
      <select id="card-program" class="field" bind:value={form.program} required>
        {#if !programs.includes(form.program)}<option value={form.program}>{form.program || 'Pick a programme'}</option
          >{/if}
        {#each programs as program (program)}<option value={program}>{program}</option>{/each}
      </select>
    </div>
    <div>
      <label class="label" for="card-keyword">Tag keyword<FilledMark show={marks.has('keyword', form.keyword)} /></label
      >
      <input id="card-keyword" class="field" bind:value={form.keyword} aria-describedby="card-keyword-hint" />
      <p id="card-keyword-hint" class="muted mt-1 text-xs">
        A tag with this word marks a payment as made with the card.
      </p>
    </div>
    <div>
      <label class="label" for="card-last4">Last 4 digits<FilledMark show={marks.has('last4', form.last4)} /></label>
      <input id="card-last4" class="field" inputmode="numeric" maxlength="4" bind:value={form.last4} />
    </div>
    <div>
      <label class="label" for="card-statement"
        >Statement day<FilledMark show={marks.has('statementDay', form.statementDay)} /></label
      >
      <input id="card-statement" type="number" min="1" max="31" class="field" bind:value={form.statementDay} />
    </div>
    <div>
      <label class="label" for="card-due">Due day<FilledMark show={marks.has('dueDay', form.dueDay)} /></label>
      <input id="card-due" type="number" min="1" max="31" class="field" bind:value={form.dueDay} />
    </div>
    <div>
      <label class="label" for="card-expiry">Expiry<FilledMark show={marks.has('expiry', form.expiry)} /></label>
      <input id="card-expiry" class="field" placeholder="MM/YYYY" bind:value={form.expiry} />
    </div>
    <div>
      <label class="label" for="card-closed"
        >Closed from<FilledMark show={marks.has('closedFrom', form.closedFrom)} /></label
      >
      <input id="card-closed" type="date" class="field" bind:value={form.closedFrom} />
    </div>
    <div>
      <label class="label" for="card-limit"
        >Credit limit (VND)<FilledMark show={marks.has('limit', form.limit)} /></label
      >
      <input
        id="card-limit"
        type="number"
        min="0"
        step="1"
        class="field"
        bind:value={form.limit}
        aria-describedby="card-limit-hint"
      />
      <p id="card-limit-hint" class="muted mt-1 text-xs">
        The bank's limit, shared by its cards. Blank leaves it as it is.
      </p>
    </div>
    <label class="flex items-center gap-2 self-center text-sm">
      <input type="checkbox" bind:checked={form.default} />
      Default card<FilledMark show={marks.has('default', form.default)} />
    </label>
  </form>
  {#snippet footer()}
    <Button disabled={saving} onclick={onclose}>Cancel</Button>
    <Button type="submit" form="card-form" variant="primary" busy={saving} busyText="Saving…" disabled={!valid}
      >{editing ? 'Save' : 'Add'}</Button
    >
  {/snippet}
</Modal>
