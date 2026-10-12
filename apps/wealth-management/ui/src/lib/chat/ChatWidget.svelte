<script lang="ts">
  import Icon from '$lib/components/Icon.svelte';
  import type { ChatPageContext } from '$lib/api/types';
  import { base } from '$app/paths';
  import ChatPanel from './ChatPanel.svelte';

  // Floating assistant on every page except /chat (WM-AI-11).
  let { page }: { page: ChatPageContext } = $props();
  let open = $state(false);
</script>

<div class="fixed right-4 bottom-4 z-40 flex flex-col items-end gap-3">
  {#if open}
    <div
      class="glass flex h-[min(34rem,75vh)] w-[min(24rem,calc(100vw-2rem))] flex-col shadow-xl"
      role="dialog"
      aria-label="Assistant"
    >
      <div class="flex items-center justify-between border-b border-slate-900/10 px-3 py-2 dark:border-white/10">
        <span class="text-sm font-semibold">Assistant</span>
        <div class="flex">
          <a
            href="{base}/chat"
            class="btn-ghost px-2"
            aria-label="Open full page"
            title="Open full page"
            onclick={() => (open = false)}><Icon name="chat" size={16} /></a
          >
          <button type="button" class="btn-ghost px-2" aria-label="Close assistant" onclick={() => (open = false)}
            ><Icon name="x" size={16} /></button
          >
        </div>
      </div>
      <ChatPanel {page} compact class="min-h-0 flex-1" />
    </div>
  {/if}
  <button
    type="button"
    class="grid size-12 place-items-center rounded-full bg-indigo-600 text-white shadow-lg hover:bg-indigo-500"
    aria-label={open ? 'Hide assistant' : 'Ask the assistant'}
    aria-expanded={open}
    onclick={() => (open = !open)}
  >
    <Icon name={open ? 'x' : 'sparkles'} />
  </button>
</div>
