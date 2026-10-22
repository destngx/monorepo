<script lang="ts">
  import type { UploadQueue } from '$lib/uploadQueue.svelte';
  import Icon from './Icon.svelte';

  // The photos waiting for Send (HW-IMP-11). Once Send is pressed each shows
  // its own upload bar, a tick once it is on the Mac, and Failed when it did
  // not get there; the next Send tries it again.
  let { queue, disabled = false }: { queue: UploadQueue; disabled?: boolean } = $props();
</script>

<div class="grid grid-cols-3 gap-2 sm:grid-cols-6">
  {#each queue.entries as e, i (e.key)}
    <div class="relative aspect-[3/4] overflow-hidden rounded-xl border border-ink-900/10 dark:border-ink-50/10">
      {#if e.file.type.startsWith('image/') && e.file.type !== 'image/heic'}
        <img
          src={e.url}
          alt="Photo {i + 1}"
          class="size-full object-cover {e.state === 'waiting' || e.state === 'sent' ? '' : 'opacity-70'}"
        />
      {:else}
        <div class="grid size-full place-items-center p-2 text-center text-xs break-all">{e.file.name}</div>
      {/if}
      <button
        type="button"
        class="icon-btn absolute top-1 right-1 size-7 bg-ink-50/90 dark:bg-ink-900/90"
        aria-label="Remove photo {i + 1}"
        {disabled}
        onclick={() => queue.remove(e.key)}><Icon name="x" size={14} /></button
      >
      {#if e.state === 'sent'}
        <span
          class="absolute bottom-1 left-1 grid size-6 place-items-center rounded-full bg-accent-600 text-ink-50"
          title="On the Mac"><Icon name="check" size={14} /></span
        >
      {:else if e.state === 'duplicate'}
        <span class="absolute inset-x-1 bottom-1 rounded bg-ink-900/70 px-1 text-center text-[11px] text-ink-50"
          >Already stored</span
        >
      {:else if e.state === 'failed'}
        <span
          class="absolute inset-x-1 bottom-1 rounded bg-danger-600 px-1 py-0.5 text-center text-[11px] font-medium text-ink-50"
          title={e.error}>Failed</span
        >
      {:else if e.state !== 'waiting'}
        <div
          class="absolute inset-x-1 bottom-1 h-1.5 overflow-hidden rounded-full bg-ink-900/20"
          role="progressbar"
          aria-label="Photo {i + 1} upload"
          aria-valuemin={0}
          aria-valuemax={100}
          aria-valuenow={Math.round(e.progress * 100)}
        >
          <div class="h-full bg-accent-600 transition-[width]" style="width: {Math.round(e.progress * 100)}%"></div>
        </div>
      {/if}
    </div>
  {/each}
</div>
