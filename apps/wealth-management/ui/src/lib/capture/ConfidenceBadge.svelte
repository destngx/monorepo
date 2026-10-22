<script lang="ts">
  import type { CaptureReason } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import { Modal } from '@libs/ui-kit';

  // A suggestion's overall confidence (WM-CAP-26); clicking it lists the
  // reasons behind the number and the reading's notes.
  interface Props {
    overall: number;
    reasons: CaptureReason[];
    notes?: string[];
  }

  let { overall, reasons, notes = [] }: Props = $props();
  let open = $state(false);

  // Above 80 starts selected, like the engine's CaptureSelectScore.
  const tone = $derived(overall > 80 ? 'good' : overall >= 50 ? 'warning' : 'critical');

  /** The reading is the starting score; every other effect is points taken off. */
  const effectText = (effect: number, index: number) =>
    index === 0 && effect > 0 ? `${effect}%` : effect < 0 ? `−${-effect}` : `+${effect}`;
</script>

<button
  type="button"
  class="rounded-full transition hover:ring-2 hover:ring-accent-400/60"
  aria-label="Confidence {overall}%: show why"
  title="Why {overall}%?"
  onclick={() => (open = true)}
>
  <Badge status={tone}>{overall}%</Badge>
</button>

<Modal {open} title="Confidence {overall}%" onclose={() => (open = false)}>
  <ul class="divide-y divide-stone-900/5 text-sm dark:divide-white/5">
    {#each reasons as reason, index (reason.text)}
      <li class="flex items-start justify-between gap-4 py-2">
        <span class={reason.effect ? '' : 'muted'}>{reason.text}</span>
        {#if reason.effect}
          <span class="font-medium whitespace-nowrap tabular-nums">{effectText(reason.effect, index)}</span>
        {/if}
      </li>
    {/each}
  </ul>
  <p class="muted mt-3 text-xs">Rows above 80% start selected.</p>
  {#if notes.length}
    <h3 class="mt-4 text-xs font-medium uppercase">Details</h3>
    {#each notes as note (note)}<p class="muted mt-1 text-xs"><MaskedText text={note} /></p>{/each}
  {/if}
</Modal>
