<script lang="ts">
  import Markdown from '$lib/ai/Markdown.svelte';
  import type { ChatPageContext } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import MaskedText from '$lib/components/MaskedText.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import Spinner from '$lib/components/Spinner.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { tick } from 'svelte';
  import { chat } from './chat.svelte';
  import { removedByRetry, retryFrom, suggestionsFor, type ChatEntry } from './history';

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
    getMoneyLent: 'Reading money lent',
    getGoals: 'Reading goals',
  };

  const COPIED_MS = 1500;

  let confirmClear = $state(false);
  let scroller = $state<HTMLDivElement>();
  /** Index of the message just copied, to flash a check mark. */
  let copied = $state<number | null>(null);
  let copiedTimer: ReturnType<typeof setTimeout> | undefined;
  /** The message being rewritten and its draft text. */
  let editing = $state<{ index: number; text: string } | null>(null);
  /** A retry or an edited resend waiting for confirmation, since both drop messages. */
  let pending = $state<{ index: number; text?: string } | null>(null);
  const pendingRemoved = $derived(pending ? removedByRetry(chat.entries, pending.index) : 0);
  const suggestions = $derived(suggestionsFor(page.path));
  // The reply being streamed, if any: it shows a typing indicator, then a caret.
  const streamingIndex = $derived(chat.streaming ? chat.entries.length - 1 : -1);

  // Keep the newest message in view while streaming.
  $effect(() => {
    const last = chat.entries.at(-1);
    void last?.content;
    void chat.entries.length;
    void tick().then(() => scroller?.scrollTo({ top: scroller.scrollHeight }));
  });

  function send(text: string) {
    if (chat.streaming || !text.trim()) return;
    chat.draft = '';
    editing = null;
    void chat.send(text, page);
  }

  function confirmPending() {
    if (!pending) return;
    const { index, text } = pending;
    pending = null;
    editing = null;
    if (text === undefined) void chat.retry(index, page);
    else void chat.editAndRetry(index, text, page);
  }

  function startEdit(entry: ChatEntry, index: number) {
    editing = { index, text: entry.content };
  }

  /**
   * Sends the rewritten message after confirmation; the replies after it are
   * dropped since they no longer match.
   */
  function saveEdit() {
    if (!editing?.text.trim()) return;
    pending = { ...editing };
  }

  function onEditKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      event.preventDefault();
      editing = null;
    } else if (event.key === 'Enter' && !event.shiftKey && !event.isComposing) {
      event.preventDefault();
      saveEdit();
    }
  }

  function focusEnd(node: HTMLTextAreaElement) {
    node.focus();
    node.setSelectionRange(node.value.length, node.value.length);
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
      send(chat.draft);
    }
  }
</script>

<!-- One toolbar per message: copy and edit on yours, copy and retry on replies. Older messages show it faded until hovered. -->
{#snippet actions(entry: ChatEntry, index: number)}
  {#if !chat.streaming && editing?.index !== index}
    <div
      class="flex gap-0.5 transition-opacity duration-200 focus-within:opacity-100 group-hover:opacity-100 {index ===
      chat.entries.length - 1
        ? ''
        : 'opacity-40 [@media(hover:none)]:opacity-100'}"
    >
      {#if entry.content}
        <button
          type="button"
          class="icon-btn {copied === index
            ? 'text-emerald-600 hover:text-emerald-600 dark:text-emerald-400 dark:hover:text-emerald-400'
            : ''}"
          aria-label={copied === index ? 'Copied' : 'Copy message'}
          title={copied === index ? 'Copied' : 'Copy'}
          onclick={() => copy(entry.content, index)}
          ><Icon name={copied === index ? 'check' : 'copy'} size={14} /></button
        >
      {/if}
      <!-- A failed reply has its Retry next to the error instead. -->
      {#if entry.role === 'assistant' && !entry.error && retryFrom(chat.entries, index)}
        <button
          type="button"
          class="group/retry icon-btn"
          aria-label="Retry"
          title="Retry"
          onclick={() => (pending = { index })}
          ><span class="transition-transform duration-300 group-hover/retry:rotate-180"
            ><Icon name="refresh" size={14} /></span
          ></button
        >
      {/if}
      {#if entry.role === 'user'}
        <button
          type="button"
          class="icon-btn"
          aria-label="Edit message"
          title="Edit"
          onclick={() => startEdit(entry, index)}><Icon name="pencil" size={14} /></button
        >
      {/if}
    </div>
  {/if}
{/snippet}

{#snippet editor()}
  {#if editing}
    <form
      class="w-full space-y-2"
      onsubmit={(event) => {
        event.preventDefault();
        saveEdit();
      }}
    >
      <label class="sr-only" for="chat-edit-{compact ? 'widget' : 'page'}">Edit message</label>
      <textarea
        id="chat-edit-{compact ? 'widget' : 'page'}"
        class="field max-h-64 min-h-20 w-full resize-y text-sm"
        bind:value={editing.text}
        onkeydown={onEditKeydown}
        {@attach focusEnd}
      ></textarea>
      <div class="flex justify-end gap-2">
        <Button size="sm" onclick={() => (editing = null)}>Cancel</Button>
        <Button
          type="submit"
          variant="primary"
          size="sm"
          title="Answer again; removes later messages"
          disabled={!editing.text.trim()}>Save &amp; send</Button
        >
      </div>
    </form>
  {/if}
{/snippet}

<Modal
  open={pending !== null}
  title={pending?.text === undefined ? 'Retry this message?' : 'Send the edited message?'}
  onclose={() => (pending = null)}
>
  <p class="text-sm">
    {#if pendingRemoved === 0}
      The assistant will answer this message again.
    {:else}
      This removes {pendingRemoved === 1 ? '1 message' : `${pendingRemoved} messages`} from the conversation and the assistant
      will answer again. This can't be undone.
    {/if}
  </p>
  {#snippet footer()}
    <Button onclick={() => (pending = null)}>Cancel</Button>
    <Button variant="primary" onclick={confirmPending}>{pending?.text === undefined ? 'Retry' : 'Send'}</Button>
  {/snippet}
</Modal>

<Modal open={confirmClear} title="Clear the conversation?" onclose={() => (confirmClear = false)}>
  <p class="text-sm">Every message is removed from this browser. This can't be undone.</p>
  {#snippet footer()}
    <Button onclick={() => (confirmClear = false)}>Cancel</Button>
    <Button
      variant="danger"
      onclick={() => {
        confirmClear = false;
        editing = null;
        chat.clear();
      }}>Clear</Button
    >
  {/snippet}
</Modal>

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
        <p class="muted max-w-xs text-sm">Ask about your balances, spending, budget, money lent or goals.</p>
        <div class="flex flex-wrap justify-center gap-2">
          {#each suggestions as suggestion (suggestion)}
            <Button
              size="sm"
              class="border border-slate-900/10 hover:border-indigo-400/50 dark:border-white/10"
              onclick={() => send(suggestion)}>{suggestion}</Button
            >
          {/each}
        </div>
      </div>
    {:else}
      {#each chat.entries as entry, index (index)}
        {#if entry.role === 'user'}
          <div class="group flex flex-col items-end gap-1">
            {#if editing?.index === index}
              {@render editor()}
            {:else}
              <!-- Amounts the user typed hide with the privacy mask too (WM-UI-2). -->
              <p
                class="max-w-[85%] rounded-2xl rounded-br-md bg-indigo-600 px-3 py-2 text-sm whitespace-pre-wrap text-white"
              >
                <MaskedText text={entry.content} />
              </p>
            {/if}
            {@render actions(entry, index)}
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
                    {#if tool.status === 'running'}
                      <Spinner size={12} />
                    {:else}
                      <Icon name={tool.status === 'ok' ? 'check' : 'alert'} size={12} />
                    {/if}
                    {toolLabels[tool.name] ?? tool.name}
                  </span>
                {/each}
              </div>
            {/if}
            {#if entry.content}
              <!-- Markdown turns amounts into mask tokens while masked (WM-UI-15). -->
              <div class="rounded-2xl rounded-bl-md bg-slate-900/5 px-3 py-2 dark:bg-white/5">
                <Markdown text={entry.content} />
                {#if index === streamingIndex}
                  <span
                    class="mt-1 inline-flex items-center gap-1.5 text-xs text-slate-500 dark:text-slate-400"
                    role="status"
                    ><span class="inline-block h-3 w-1.5 rounded-sm bg-indigo-500 motion-safe:animate-pulse"
                    ></span>Answering…</span
                  >
                {/if}
              </div>
            {:else if index === streamingIndex && !entry.error}
              <div class="flex items-center gap-1 px-3 py-3" role="status" aria-label="Assistant is typing">
                <span class="size-1.5 rounded-full bg-slate-400 motion-safe:animate-bounce"></span>
                <span class="size-1.5 rounded-full bg-slate-400 [animation-delay:150ms] motion-safe:animate-bounce"
                ></span>
                <span class="size-1.5 rounded-full bg-slate-400 [animation-delay:300ms] motion-safe:animate-bounce"
                ></span>
                <span class="muted ml-1.5 text-xs" aria-hidden="true">Thinking…</span>
              </div>
            {/if}
            {#if entry.error}
              <div
                class="flex flex-wrap items-center gap-2 rounded-xl bg-rose-500/10 px-3 py-1.5 text-xs text-rose-700 dark:text-rose-300"
                role="alert"
              >
                <Icon name="alert" size={14} />
                <span class="min-w-0 flex-1 break-words">{entry.error}</span>
                {#if !chat.streaming && retryFrom(chat.entries, index)}
                  <Button size="sm" icon="refresh" onclick={() => (pending = { index })}>Retry</Button>
                {/if}
              </div>
            {/if}
            {@render actions(entry, index)}
          </div>
        {/if}
      {/each}
    {/if}
  </div>

  <form
    class="flex items-end gap-2 border-t border-slate-900/10 pt-3 dark:border-white/10 {compact ? 'px-3 pb-3' : ''}"
    onsubmit={(event) => {
      event.preventDefault();
      send(chat.draft);
    }}
  >
    {#if chat.entries.length && !chat.streaming}
      <Button size="icon" icon="trash" label="Clear conversation" onclick={() => (confirmClear = true)} />
    {/if}
    <label class="sr-only" for="chat-input-{compact ? 'widget' : 'page'}">Message</label>
    <textarea
      id="chat-input-{compact ? 'widget' : 'page'}"
      class="field max-h-32 min-h-10 flex-1 resize-none"
      rows="1"
      placeholder="Ask about your finances…"
      bind:value={chat.draft}
      {onkeydown}
    ></textarea>
    <!-- While a reply streams, Send stays busy and Stop ends the reply. -->
    {#if chat.streaming}
      <Button size="icon" icon="stop" label="Stop" title="Stop the answer" onclick={() => chat.stop()} />
    {/if}
    <Button
      type="submit"
      variant="primary"
      size="icon"
      icon="send"
      class="px-3"
      label="Send"
      title={chat.streaming ? 'Answering…' : 'Send'}
      busy={chat.streaming}
      disabled={!chat.streaming && !chat.draft.trim()}
    />
  </form>
</div>
