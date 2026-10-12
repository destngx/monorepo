<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import Icon from '$lib/components/Icon.svelte';
  import AIStatusLine from './AIStatusLine.svelte';
  import Markdown from './Markdown.svelte';
  import { AITask } from './task.svelte';

  // Sparkle button that explains the data behind a chart (WM-UI-8).
  interface Props {
    chartType: string;
    title: string;
    data: unknown;
    description?: string;
    timeframe?: string;
  }

  let { chartType, title, data, description, timeframe }: Props = $props();
  let open = $state(false);
  const task = $derived(
    new AITask((force, signal) =>
      api.chartInsight({ chartType, title, data, description, timeframe }, { force, signal }),
    ),
  );

  function toggle() {
    open = !open;
    if (open && task.status === 'idle') void task.run();
  }
</script>

<div class="relative">
  <button
    type="button"
    class="btn-ghost px-2 text-indigo-600 dark:text-indigo-400"
    aria-label="Explain {title} with AI"
    aria-expanded={open}
    title="Explain with AI"
    onclick={toggle}
  >
    <Icon name="sparkles" size={16} />
  </button>
  {#if open}
    <div
      class="glass absolute right-0 z-20 mt-2 w-80 max-w-[85vw] p-4 shadow-lg"
      role="dialog"
      aria-label="{title} insight"
    >
      <div class="mb-2 flex items-center justify-between gap-2">
        <span class="text-xs font-semibold tracking-wide uppercase">{title}</span>
        <div class="flex">
          {#if task.status === 'done'}
            <button type="button" class="btn-ghost px-1.5" aria-label="Regenerate" onclick={() => task.run(true)}
              ><Icon name="refresh" size={14} /></button
            >
          {/if}
          <button type="button" class="btn-ghost px-1.5" aria-label="Close" onclick={() => (open = false)}
            ><Icon name="x" size={14} /></button
          >
        </div>
      </div>
      {#if task.status === 'done' && task.value}
        <Markdown text={task.value} />
      {:else}
        <AIStatusLine {task} />
      {/if}
    </div>
  {/if}
</div>
