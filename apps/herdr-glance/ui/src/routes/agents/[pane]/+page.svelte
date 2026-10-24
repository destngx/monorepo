<script lang="ts">
  import { tick } from 'svelte';
  import { base } from '$app/paths';
  import { Badge, Button, Icon, toasts } from '@libs/ui-kit';
  import { agentName, statusOf } from '$lib/agents';
  import { api } from '$lib/api';
  import ConnectionDot from '$lib/components/ConnectionDot.svelte';
  import { KEY_BUTTONS } from '$lib/keys';
  import { tidyScreen } from '$lib/screen';
  import { live } from '$lib/state/live.svelte';

  let { data } = $props();

  const agent = $derived(live.find(data.paneId));
  const status = $derived(agent ? statusOf(agent.status) : null);
  const blocked = $derived(agent?.status === 'blocked');
  // Output changes fast while an agent works or waits, slowly otherwise.
  const refreshEvery = $derived(agent?.status === 'working' || agent?.status === 'blocked' ? 2000 : 8000);

  let text = $state('');
  let screenError = $state('');
  let screenLoaded = $state(false);
  let message = $state('');
  let scroller = $state<HTMLElement>();
  // Stay at the newest line unless the reader scrolled up to read.
  let follow = $state(true);

  async function loadScreen(signal?: AbortSignal) {
    try {
      const res = await api.screen(data.paneId, signal);
      const next = tidyScreen(res.text);
      screenError = '';
      screenLoaded = true;
      if (next === text) return;
      text = next;
      if (follow) {
        await tick();
        scroller?.scrollTo({ top: scroller.scrollHeight });
      }
    } catch (error) {
      if (signal?.aborted) return;
      screenError = error instanceof Error ? error.message : String(error);
    }
  }

  // Reads the output now, then on a timer while the page is visible. A state
  // change from the live stream restarts it, so a new question shows at once.
  $effect(() => {
    void agent?.status;
    const every = refreshEvery;
    const controller = new AbortController();
    let timer: ReturnType<typeof setTimeout> | undefined;
    const run = async () => {
      if (document.visibilityState === 'visible') await loadScreen(controller.signal);
      if (!controller.signal.aborted) timer = setTimeout(run, every);
    };
    void run();
    return () => {
      controller.abort();
      clearTimeout(timer);
    };
  });

  function onScroll() {
    if (!scroller) return;
    follow = scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < 48;
  }

  function jumpToLatest() {
    follow = true;
    scroller?.scrollTo({ top: scroller.scrollHeight, behavior: 'smooth' });
  }

  // After a key or a prompt, read again soon so the reply shows without waiting for the timer.
  function readSoon() {
    follow = true;
    setTimeout(() => void loadScreen(), 400);
  }

  async function press(keys: string[]) {
    await api.keys(data.paneId, keys);
    readSoon();
  }

  async function send() {
    const value = message.trim();
    if (!value) return;
    await api.prompt(data.paneId, value);
    message = '';
    readSoon();
  }

  function onKeydown(event: KeyboardEvent) {
    // Return makes a new line on the phone; Cmd or Ctrl with Return sends from a keyboard.
    if (event.key === 'Enter' && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      void send().catch((error: unknown) => toasts.error(error, 'Could not send the message'));
    }
  }
</script>

<svelte:head><title>{agent ? agentName(agent) : 'Agent'} · Agents</title></svelte:head>

<div class="mx-auto flex h-dvh max-w-3xl flex-col">
  <header
    class="z-30 flex shrink-0 items-center gap-2 border-b border-rule bg-canvas/90 px-2 pt-[max(0.5rem,env(safe-area-inset-top))] pb-2"
  >
    <a href="{base}/" class="icon-btn hit-area size-10" aria-label="All agents" title="All agents">
      <Icon name="chevronLeft" size={22} />
    </a>
    <div class="min-w-0 flex-1">
      <h1 class="truncate font-sans text-base font-semibold">{agent ? agentName(agent) : data.paneId}</h1>
      <p class="muted flex min-w-0 items-center gap-2 text-xs">
        {#if agent && status}
          <Badge status={status.tone}>{status.label}</Badge>
          <span class="truncate">{agent.workspace || agent.workspaceId} · {agent.kind || 'agent'}</span>
        {:else if live.loaded}
          <span>Closed</span>
        {/if}
      </p>
    </div>
    <ConnectionDot connection={live.connection} />
  </header>

  {#if live.loaded && !agent}
    <div class="flex flex-1 flex-col items-center justify-center gap-3 px-6 text-center">
      <p class="font-medium">This agent is gone</p>
      <p class="muted text-sm">Its pane was closed in herdr.</p>
      <a class="btn-primary" href="{base}/">See all agents</a>
    </div>
  {:else}
    <div class="relative min-h-0 flex-1">
      <div
        bind:this={scroller}
        onscroll={onScroll}
        class="h-full overflow-y-auto overscroll-contain px-3 py-3"
        aria-label="Agent output"
        role="log"
      >
        {#if !screenLoaded && !screenError}
          <div role="status" aria-busy="true" class="space-y-2">
            <span class="sr-only">Loading output…</span>
            {#each [0, 1, 2, 3, 4] as i (i)}<div class="skeleton h-4" style="width: {90 - i * 12}%"></div>{/each}
          </div>
        {:else}
          <pre
            class="font-mono text-[12px] leading-[1.45] break-words whitespace-pre-wrap text-ink-900 dark:text-ink-100">{text ||
              'No output yet.'}</pre>
        {/if}
      </div>
      {#if screenError}
        <p role="alert" class="panel absolute inset-x-3 top-3 px-3 py-2 text-sm">
          Could not load the output: {screenError}
        </p>
      {/if}
      {#if !follow}
        <button type="button" class="btn absolute right-3 bottom-3 px-3 py-1.5 text-sm" onclick={jumpToLatest}>
          <Icon name="arrowDown" size={16} />Latest
        </button>
      {/if}
    </div>

    <footer class="shrink-0 border-t border-rule bg-surface px-2 pt-2 pb-[max(0.5rem,env(safe-area-inset-bottom))]">
      {#if blocked}
        <p class="mb-2 px-1 text-sm font-medium text-warn-700 dark:text-warn-300">
          <Icon name="alert" size={14} class="inline align-[-2px]" /> It is asking you something. Answer with the keys.
        </p>
      {/if}
      <div class="-mx-2 flex gap-1.5 overflow-x-auto px-2 pb-2" role="group" aria-label="Keys">
        {#each KEY_BUTTONS as key (key.label)}
          <Button
            size="sm"
            class="min-h-11 min-w-11 shrink-0 justify-center border-ink-300! bg-canvas font-mono dark:border-ink-600!"
            label={key.name}
            onclick={() => press(key.keys)}
            errorContext="Could not press {key.label}">{key.label}</Button
          >
        {/each}
      </div>
      <form
        class="flex items-end gap-2"
        onsubmit={(e) => {
          e.preventDefault();
        }}
      >
        <label for="message" class="sr-only">Message to the agent</label>
        <textarea
          id="message"
          bind:value={message}
          onkeydown={onKeydown}
          class="field max-h-40 min-h-11 flex-1 resize-none py-2.5 text-base [field-sizing:content]"
          rows="1"
          placeholder={blocked ? 'Answer the question first' : 'Message the agent'}
          enterkeyhint="enter"
          autocapitalize="sentences"
          maxlength="8000"
        ></textarea>
        <Button
          variant="primary"
          size="icon"
          class="size-11 shrink-0"
          icon="send"
          label="Send"
          disabled={!message.trim()}
          onclick={send}
          errorContext="Could not send the message"
        />
      </form>
    </footer>
  {/if}
</div>
