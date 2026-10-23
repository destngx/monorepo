<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { FillResult, MerchantInput } from '$lib/api/types';
  import FillBox from '$lib/fill/FillBox.svelte';
  import FilledMark from '$lib/fill/FilledMark.svelte';
  import { applyFill, asNumber, asText } from '$lib/fill/fill';
  import { FillMarks } from '$lib/fill/marks.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { Button, Modal } from '@libs/ui-kit';

  // Saves the MCC a payee is billed with (WM-FILL-14), e.g. from a receipt.
  // The same pattern and start date replace the saved row.
  interface Props {
    open: boolean;
    /** Starts the form, e.g. from a best-card answer. */
    initial?: { pattern?: string; mcc?: number | null };
    onclose: () => void;
  }

  let { open, initial, onclose }: Props = $props();

  interface Form {
    pattern: string;
    mcc: number | null;
    from: string;
    confidence: 'assumed' | 'confirmed';
    note: string;
  }

  const start = (): Form => ({
    pattern: initial?.pattern ?? '',
    mcc: initial?.mcc ?? null,
    from: '',
    confidence: 'assumed',
    note: '',
  });
  let form = $state<Form>({ pattern: '', mcc: null, from: '', confidence: 'assumed', note: '' });
  let saving = $state(false);
  const marks = new FillMarks();
  $effect(() => {
    if (open) form = start();
    else marks.clear();
  });

  const valid = $derived(form.pattern.trim() !== '' && form.mcc !== null && form.mcc >= 1 && form.mcc <= 9999);

  function fill(result: FillResult) {
    form = applyFill(form, result.values, {
      pattern: asText,
      mcc: asNumber,
      from: asText,
      note: asText,
      confidence: (value) => (value === 'confirmed' ? 'confirmed' : 'assumed'),
    });
    marks.take(result);
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (!valid) return;
    saving = true;
    const input: MerchantInput = {
      pattern: form.pattern.trim(),
      mcc: form.mcc!,
      from: form.from || undefined,
      confidence: form.confidence,
      note: form.note.trim() || undefined,
    };
    try {
      await api.saveMerchant(input);
      toasts.success(`Saved MCC ${String(input.mcc).padStart(4, '0')} for ${input.pattern}.`);
      onclose();
      await invalidateAll();
    } catch (error) {
      toasts.error(error, 'Could not save the MCC');
    } finally {
      saving = false;
    }
  }
</script>

<Modal {open} title="Payee MCC" {onclose} busy={saving}>
  <FillBox form="merchant" class="mb-4" current={() => ({ ...form })} onfill={fill} />
  <form id="merchant-form" class="grid grid-cols-1 gap-4 sm:grid-cols-2" onsubmit={save}>
    <div class="sm:col-span-2">
      <label class="label" for="merchant-pattern"
        >Payee pattern *<FilledMark show={marks.has('pattern', form.pattern)} /></label
      >
      <input
        id="merchant-pattern"
        class="field"
        bind:value={form.pattern}
        required
        aria-describedby="merchant-pattern-hint"
      />
      <p id="merchant-pattern-hint" class="muted mt-1 text-xs">
        Payees containing this text take the MCC; case and accents are ignored.
      </p>
    </div>
    <div>
      <label class="label" for="merchant-mcc">MCC *<FilledMark show={marks.has('mcc', form.mcc)} /></label>
      <input id="merchant-mcc" type="number" min="1" max="9999" step="1" class="field" bind:value={form.mcc} required />
    </div>
    <div>
      <label class="label" for="merchant-confidence"
        >Confidence<FilledMark show={marks.has('confidence', form.confidence)} /></label
      >
      <select id="merchant-confidence" class="field" bind:value={form.confidence}>
        <option value="assumed">Assumed</option>
        <option value="confirmed">Confirmed on a receipt or statement</option>
      </select>
    </div>
    <div>
      <label class="label" for="merchant-from">From<FilledMark show={marks.has('from', form.from)} /></label>
      <input
        id="merchant-from"
        type="date"
        class="field"
        bind:value={form.from}
        aria-describedby="merchant-from-hint"
      />
      <p id="merchant-from-hint" class="muted mt-1 text-xs">Only when the MCC changed on a date.</p>
    </div>
    <div>
      <label class="label" for="merchant-note">Source<FilledMark show={marks.has('note', form.note)} /></label>
      <input id="merchant-note" class="field" bind:value={form.note} />
    </div>
  </form>
  {#snippet footer()}
    <Button disabled={saving} onclick={onclose}>Cancel</Button>
    <Button type="submit" form="merchant-form" variant="primary" busy={saving} busyText="Saving…" disabled={!valid}
      >Save</Button
    >
  {/snippet}
</Modal>
