<script lang="ts">
  import type { ReadingState } from '$lib/reading';
  import Spinner from './Spinner.svelte';

  // An inbox item's OCR and AI reading status (HW-READ-6).
  let { status, showDetail = false }: { status: ReadingState; showDetail?: boolean } = $props();

  const tones = {
    neutral: 'bg-ink-500/10 text-ink-700 dark:text-ink-300',
    busy: 'bg-info-500/10 text-info-800 dark:text-info-200',
    warn: 'bg-warn-500/15 text-warn-800 dark:text-warn-200',
    ok: 'bg-accent-600/10 text-accent-700 dark:text-accent-300',
    error: 'bg-danger-500/10 text-danger-700 dark:text-danger-300',
  };
</script>

<span class="inline-flex max-w-full flex-col gap-1">
  <span class="tag {tones[status.tone]}" title={showDetail ? undefined : status.detail || undefined}>
    {#if status.tone === 'busy'}<Spinner size={12} />{/if}{status.label}
  </span>
  {#if showDetail && status.detail}
    <span class="muted text-xs whitespace-pre-line">{status.detail}</span>
  {/if}
</span>
