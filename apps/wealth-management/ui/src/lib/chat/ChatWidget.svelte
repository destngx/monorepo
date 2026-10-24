<script lang="ts">
  import { Icon } from '@libs/ui-kit';
  import type { ChatPageContext } from '$lib/api/types';
  import { base } from '$app/paths';
  import { tick } from 'svelte';
  import ChatPanel from './ChatPanel.svelte';

  // Floating assistant on every page except /chat (WM-AI-11).
  let { page }: { page: ChatPageContext } = $props();
  let open = $state(false);
  let toggle = $state<HTMLButtonElement>();
  let panel = $state<HTMLDivElement>();

  // It stays open on clicks elsewhere so the page can be read and copied from while chatting.
  async function show() {
    open = true;
    await tick();
    // Not on touch screens, where focusing the box opens the keyboard over the answers.
    if (!globalThis.matchMedia?.('(pointer: coarse)').matches) panel?.querySelector('textarea')?.focus();
  }

  function hide() {
    // Focus would fall to the page when the panel unmounts; it goes back to the toggle.
    if (panel?.contains(document.activeElement)) toggle?.focus();
    open = false;
  }

  function onkeydown(event: KeyboardEvent) {
    // An open confirm dialog or an edit box handles its own Escape.
    if (!open || event.key !== 'Escape' || event.defaultPrevented || document.querySelector('dialog[open]')) return;
    event.preventDefault();
    hide();
  }
</script>

<svelte:window {onkeydown} />

<div
  class="fixed right-[max(1rem,env(safe-area-inset-right))] bottom-tabbar z-40 lg:bottom-4 flex flex-col items-end gap-3"
>
  {#if open}
    <div
      class="panel flex h-[min(34rem,calc(100dvh-var(--spacing-tabbar)-env(safe-area-inset-top)-9rem))] w-[min(24rem,calc(100vw-2rem))] flex-col shadow-hard-lg lg:h-[min(34rem,75vh)]"
      role="dialog"
      aria-label="Assistant"
      bind:this={panel}
    >
      <div class="flex items-center justify-between border-b border-stone-900/10 px-3 py-2 dark:border-white/10">
        <span class="text-sm font-semibold">Assistant</span>
        <div class="flex">
          <a href="{base}/chat" class="btn-ghost px-2" aria-label="Open full page" title="Open full page" onclick={hide}
            ><Icon name="chat" size={16} /></a
          >
          <button
            type="button"
            class="btn-ghost px-2"
            aria-label="Close assistant"
            title="Close assistant"
            onclick={hide}><Icon name="x" size={16} /></button
          >
        </div>
      </div>
      <ChatPanel {page} compact class="min-h-0 flex-1" />
    </div>
  {/if}
  <button
    type="button"
    class="grid size-12 place-items-center rounded-lg border-[1.5px] border-edge bg-stone-900 text-accent-200 shadow-hard-sm transition hover:-translate-x-px hover:-translate-y-px hover:shadow-hard active:translate-x-0.5 active:translate-y-0.5 active:shadow-none motion-reduce:translate-none dark:bg-stone-100 dark:text-accent-800"
    title={open ? 'Hide assistant' : 'Ask the assistant'}
    aria-label={open ? 'Hide assistant' : 'Ask the assistant'}
    aria-expanded={open}
    bind:this={toggle}
    onclick={() => (open ? hide() : void show())}
  >
    <Icon name={open ? 'x' : 'sparkles'} />
  </button>
</div>
