<script lang="ts">
  import type { ReadingState } from '$lib/reading';
  import Spinner from './Spinner.svelte';

  // An inbox item's OCR and AI reading status (HW-READ-6).
  let { status, showDetail = false }: { status: ReadingState; showDetail?: boolean } = $props();

  const tones = {
    neutral: 'bg-slate-500/10 text-slate-700 dark:text-slate-300',
    busy: 'bg-sky-500/10 text-sky-800 dark:text-sky-200',
    warn: 'bg-amber-500/15 text-amber-800 dark:text-amber-200',
    ok: 'bg-teal-600/10 text-teal-700 dark:text-teal-300',
    error: 'bg-rose-500/10 text-rose-700 dark:text-rose-300',
  };
</script>

<span class="inline-flex max-w-full flex-col gap-1">
  <span
    class="inline-flex w-fit items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium {tones[status.tone]}"
    title={showDetail ? undefined : status.detail || undefined}
  >
    {#if status.tone === 'busy'}<Spinner size={12} />{/if}{status.label}
  </span>
  {#if showDetail && status.detail}
    <span class="muted text-xs whitespace-pre-line">{status.detail}</span>
  {/if}
</span>
