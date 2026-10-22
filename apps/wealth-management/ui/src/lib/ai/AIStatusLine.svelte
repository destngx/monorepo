<script lang="ts">
  import { base } from '$app/paths';
  import { Icon } from '@libs/ui-kit';
  import type { AITask } from './task.svelte';

  // Shared idle / loading / error states of an AI card.
  let { task, label = 'Generate', hint }: { task: AITask<unknown>; label?: string; hint?: string } = $props();
</script>

{#if task.status === 'idle'}
  <div class="flex flex-col items-start gap-2">
    {#if hint}<p class="muted text-sm">{hint}</p>{/if}
    <button type="button" class="btn-primary" onclick={() => task.run()}
      ><Icon name="sparkles" size={16} /> {label}</button
    >
  </div>
{:else if task.status === 'loading'}
  <div class="space-y-2" role="status" aria-label="Generating">
    <div class="h-3 w-3/4 animate-pulse rounded bg-stone-900/10 dark:bg-white/10"></div>
    <div class="h-3 w-full animate-pulse rounded bg-stone-900/10 dark:bg-white/10"></div>
    <div class="h-3 w-2/3 animate-pulse rounded bg-stone-900/10 dark:bg-white/10"></div>
  </div>
{:else if task.status === 'error'}
  <div class="flex flex-col items-start gap-2 text-sm" role="alert">
    <p class="text-rose-700 dark:text-rose-300">{task.error}</p>
    {#if task.hint}
      <p class="muted">
        {task.hint}
        <a class="text-accent-600 hover:underline dark:text-accent-400" href="{base}/settings">Open settings</a>
      </p>
    {/if}
    <button type="button" class="btn-ghost" onclick={() => task.run(true)}
      ><Icon name="refresh" size={16} /> Retry</button
    >
  </div>
{/if}
