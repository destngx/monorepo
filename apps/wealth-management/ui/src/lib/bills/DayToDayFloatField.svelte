<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { Preferences } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Money from '$lib/components/Money.svelte';
  import { STATUS } from '$lib/status';
  import { parseFloatDraft } from './idleCash';

  // The day-to-day float (idleCash.dayToDayFloat, D34): money kept ready in
  // everyday accounts before any cash counts as idle. Saved to the
  // Preferences tab; blank in the sheet means the spending buffer. The input
  // starts empty so the amount is never shown unmasked.
  let {
    saved,
    effective = null,
    defaultHint,
    onsaved,
  }: {
    /** The saved preference; null = the default. */
    saved: number | null;
    /** The float in use (idle cash), shown masked. */
    effective?: number | null;
    /** What the default is, e.g. "the spending buffer (1 month of average expense)". */
    defaultHint: string;
    onsaved: (prefs: Preferences) => unknown;
  } = $props();

  let draft = $state('');
  let touched = $state(false);
  const parsed = $derived(parseFloatDraft(draft));
  const problem = $derived(touched && draft.trim() !== '' ? parsed.error : null);

  async function save(value: number | null) {
    const prefs = await api.savePreferences([{ key: 'idleCash.dayToDayFloat', value }]);
    draft = '';
    touched = false;
    await onsaved(prefs);
  }
</script>

<div class="text-sm" data-testid="day-to-day-float">
  <p>
    Day-to-day float:
    {#if effective !== null}<strong><Money value={effective} /></strong>{:else if saved !== null}<strong
        ><Money value={saved} /></strong
      >{:else}<strong>default</strong>{/if}
    <span class="muted">({saved === null ? defaultHint : 'your setting'})</span>
  </p>
  <div class="mt-1 flex flex-wrap items-center gap-2">
    <label class="sr-only" for="day-to-day-float-input">New day-to-day float in VND</label>
    <input
      id="day-to-day-float-input"
      class="field w-40"
      inputmode="numeric"
      autocomplete="off"
      placeholder="Amount in VND"
      aria-invalid={problem ? true : undefined}
      aria-describedby={problem ? 'day-to-day-float-problem' : undefined}
      bind:value={draft}
      oninput={() => (touched = true)}
    />
    <Button
      size="sm"
      variant="primary"
      busyText="Saving…"
      disabled={parsed.value === null}
      success="Day-to-day float saved to your sheet."
      errorContext="Could not save the day-to-day float"
      onclick={() => save(parsed.value)}>Save float</Button
    >
    {#if saved !== null}
      <Button
        size="sm"
        busyText="Resetting…"
        success="Day-to-day float back to the default."
        errorContext="Could not reset the day-to-day float"
        onclick={() => save(null)}>Use default</Button
      >
    {/if}
  </div>
  {#if problem}
    <p id="day-to-day-float-problem" class="mt-1 text-xs {STATUS.warning.ink}" role="alert">{problem}</p>
  {/if}
</div>
