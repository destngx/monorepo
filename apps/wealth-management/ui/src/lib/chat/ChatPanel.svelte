<script lang="ts">
  import Markdown from '$lib/ai/Markdown.svelte';
  import type { ChatPageContext } from '$lib/api/types';
  import Icon from '$lib/components/Icon.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { tick } from 'svelte';
  import { chat } from './chat.svelte';
  import { canRetry, suggestionsFor } from './history';

  interface Props {
    page: ChatPageContext;
    /** Compact layout for the floating widget. */
    compact?: boolean;
    class?: string;
  }

  let { page, compact = false, class: className = '' }: Props = $props();

  const toolLabels: Record<string, string> = {
    getAccountBalances: 'Reading balances',
    getRecentTransactions: 'Searching transactions',
    getBudgetStatus: 'Checking budget',
    getFinancialHealth: 'Scoring health',
    getLoans: 'Reading loans',
    getGoals: 'Reading goals',
  };

  const COPIED_MS = 1500;

  let draft = $state('');
  let scroller = $state<HTMLDivElement>();
  /** Index of the message just copied, to flash a check mark. */
  let copied = $state<number | null>(null);
  let copiedTimer: ReturnType<typeof setTimeout> | undefined;
  const suggestions = $derived(suggestionsFor(page.path));
  const retryable = $derived(!chat.streaming && canRetry(chat.entries));

  // Keep the newest message in view while streaming.
  $effect(() => {
    const last = chat.entries.at(-1);
    void last?.content;
    void chat.entries.length;
    void tick().then(() => scroller?.scrollTo({ top: scroller.scrollHeight }));
  });

  function send(text: string) {
    if (chat.streaming || !text.trim()) return;
    draft = '';
    void chat.send(text, page);
  }

  async function copy(text: string, index: number) {
    try {
      await navigator.clipboard.writeText(text);
    } catch (error) {
      toasts.error(error, 'Copy failed');
      return;
    }
    copied = index;
    clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => (copied = null), COPIED_MS);
  }

  function onkeydown(event: KeyboardEvent) {
    if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
      event.preventDefault();
      send(draft);
    }
  }
</script>

{#snippet actions(text: string, index: number, retry: boolean)}
  {@const writing = chat.streaming && index === chat.entries.length - 1}
  {#if (text && !writing) || retry}
    <div
      class="muted flex gap-0.5 transition-opacity focus-within:opacity-100 group-hover:opacity-100 {retry
        ? ''
        : 'opacity-0 [@media(hover:none)]:opacity-100'}"
    >
      {#if text && !writing}
        <button
          type="button"
          class="rounded-md p-1 hover:bg-slate-900/5 dark:hover:bg-white/5"
          aria-label={copied === index ? 'Copied' : 'Copy message'}
          title={copied === index ? 'Copied' : 'Copy'}
          onclick={() => copy(text, index)}><Icon name={copied === index ? 'check' : 'copy'} size={14} /></button
        >
      {/if}
      {#if retry}
        <button
          type="button"
          class="rounded-md p-1 hover:bg-slate-900/5 dark:hover:bg-white/5"
          aria-label="Retry"
          title="Retry"
          onclick={() => chat.retry(page)}><Icon name="refresh" size={14} /></button
        >
      {/if}
    </div>
  {/if}
{/snippet}

<div class="flex min-h-0 flex-col {className}">
  <div
    bind:this={scroller}
    class="min-h-0 flex-1 space-y-3 overflow-y-auto {compact ? 'p-3' : 'p-1'}"
    aria-live="polite"
  >
    {#if chat.entries.length === 0}
      <div class="flex h-full flex-col items-center justify-center gap-3 py-6 text-center">
        <span class="grid size-10 place-items-center rounded-xl bg-indigo-600/10 text-indigo-600 dark:text-indigo-400"
          ><Icon name="sparkles" /></span
        >
        <p class="muted max-w-xs text-sm">Ask about your balances, spending, budget, loans or goals.</p>
        <div class="flex flex-wrap justify-center gap-2">
          {#each suggestions as suggestion (suggestion)}
            <button
              type="button"
              class="btn-ghost border border-slate-900/10 text-xs dark:border-white/10"
              onclick={() => send(suggestion)}>{suggestion}</button
            >
          {/each}
        </div>
      </div>
    {:else}
      {#each chat.entries as entry, index (index)}
        {#if entry.role === 'user'}
          <div class="group flex flex-col items-end gap-1">
            <p
              class="max-w-[85%] rounded-2xl rounded-br-md bg-indigo-600 px-3 py-2 text-sm whitespace-pre-wrap text-white"
            >
              {entry.content}
            </p>
            {@render actions(entry.content, index, false)}
          </div>
        {:else}
          <div class="group max-w-[95%] space-y-1.5">
            {#if entry.tools?.length}
              <div class="flex flex-wrap gap-1.5">
                {#each entry.tools as tool, toolIndex (toolIndex)}
                  <span
                    class="inline-flex items-center gap-1 rounded-full bg-slate-900/5 px-2 py-0.5 text-xs dark:bg-white/5 {tool.status ===
                    'error'
                      ? 'text-rose-600 dark:text-rose-400'
                      : 'muted'}"
                  >
                    <Icon
                      name={tool.status === 'running' ? 'clock' : tool.status === 'ok' ? 'check' : 'alert'}
                      size={12}
                    />
                    {toolLabels[tool.name] ?? tool.name}
                  </span>
                {/each}
              </div>
            {/if}
            {#if entry.content}
              <div class="rounded-2xl rounded-bl-md bg-slate-900/5 px-3 py-2 dark:bg-white/5">
                <Markdown text={entry.content} />
              </div>
            {:else if chat.streaming && index === chat.entries.length - 1 && !entry.error}
              <div class="flex gap-1 px-3 py-3" role="status" aria-label="Assistant is typing">
                <span class="size-1.5 animate-bounce rounded-full bg-slate-400"></span>
                <span class="size-1.5 animate-bounce rounded-full bg-slate-400 [animation-delay:150ms]"></span>
                <span class="size-1.5 animate-bounce rounded-full bg-slate-400 [animation-delay:300ms]"></span>
              </div>
            {/if}
            {#if entry.error}
              <p class="text-xs text-rose-600 dark:text-rose-400" role="alert">{entry.error}</p>
            {/if}
            {@render actions(entry.content, index, retryable && index === chat.entries.length - 1)}
          </div>
        {/if}
      {/each}
    {/if}
  </div>

  <form
    class="flex items-end gap-2 border-t border-slate-900/10 pt-3 dark:border-white/10 {compact ? 'px-3 pb-3' : ''}"
    onsubmit={(event) => {
      event.preventDefault();
      send(draft);
    }}
  >
    {#if chat.entries.length && !chat.streaming}
      <button
        type="button"
        class="btn-ghost px-2"
        aria-label="Clear conversation"
        title="Clear conversation"
        onclick={() => chat.clear()}><Icon name="trash" size={16} /></button
      >
    {/if}
    <label class="sr-only" for="chat-input-{compact ? 'widget' : 'page'}">Message</label>
    <textarea
      id="chat-input-{compact ? 'widget' : 'page'}"
      class="field max-h-32 min-h-10 flex-1 resize-none"
      rows="1"
      placeholder="Ask about your finances…"
      bind:value={draft}
      {onkeydown}
    ></textarea>
    {#if chat.streaming}
      <button type="button" class="btn-ghost px-2" aria-label="Stop" onclick={() => chat.stop()}
        ><Icon name="stop" size={16} /></button
      >
    {:else}
      <button type="submit" class="btn-primary px-3" aria-label="Send" disabled={!draft.trim()}
        ><Icon name="send" size={16} /></button
      >
    {/if}
  </form>
</div>
