<script lang="ts">
  import Icon from '$lib/components/Icon.svelte';
  import AIStatusLine from './AIStatusLine.svelte';
  import Markdown from './Markdown.svelte';
  import { AITask } from './task.svelte';

  // Icon button that opens an AI explanation in a popover; the request runs
  // on first open and the answer is kept until regenerated (WM-UI-8).
  interface Props {
    /** Popover heading and dialog name. */
    title: string;
    /** The button's accessible name. */
    label: string;
    icon?: 'sparkles' | 'info';
    load: (force: boolean, signal: AbortSignal) => Promise<string>;
  }

  let { title, label, icon = 'sparkles', load }: Props = $props();
  let open = $state(false);
  let button: HTMLButtonElement | undefined = $state();
  let popover: HTMLDivElement | undefined = $state();
  let placement = $state('');
  const task = $derived(new AITask(load));

  const GAP = 8;
  const MARGIN = 16;
  const WIDTH = 384;

  // The popover lives at the end of <body>: cards blur their backdrop, which
  // traps any z-index inside them, so an in-card popover hides behind the
  // next card.
  function portal(node: HTMLElement) {
    document.body.appendChild(node);
    return { destroy: () => node.remove() };
  }

  // Anchors the popover under the button (above it when there is no room),
  // kept inside the viewport.
  function place() {
    if (!button) return;
    const rect = button.getBoundingClientRect();
    const width = Math.min(WIDTH, window.innerWidth - 2 * MARGIN);
    const left = Math.min(Math.max(rect.right - width, MARGIN), window.innerWidth - MARGIN - width);
    const below = window.innerHeight - rect.bottom - GAP - MARGIN;
    const above = rect.top - GAP - MARGIN;
    const vertical =
      below >= 240 || below >= above
        ? `top: ${rect.bottom + GAP}px; max-height: ${below}px;`
        : `bottom: ${window.innerHeight - rect.top + GAP}px; max-height: ${above}px;`;
    placement = `left: ${left}px; width: ${width}px; ${vertical}`;
  }

  $effect(() => {
    if (!open) return;
    place();
    window.addEventListener('scroll', place, true);
    window.addEventListener('resize', place);
    return () => {
      window.removeEventListener('scroll', place, true);
      window.removeEventListener('resize', place);
    };
  });

  // A press anywhere outside the button and the popover closes it.
  $effect(() => {
    if (!open) return;
    const onPointerDown = (event: PointerEvent) => {
      const target = event.target as Node;
      if (!button?.contains(target) && !popover?.contains(target)) open = false;
    };
    document.addEventListener('pointerdown', onPointerDown);
    return () => document.removeEventListener('pointerdown', onPointerDown);
  });

  // The popover sits at the end of <body>, far from the button in tab order: focus moves into it on
  // open, and back to the button when Escape or Close shuts it.
  function focusOnOpen(node: HTMLElement) {
    node.focus();
  }

  function close() {
    open = false;
    button?.focus();
  }

  function toggle() {
    open = !open;
    if (open && task.status === 'idle') void task.run();
  }
</script>

<svelte:window
  onkeydown={(event) => {
    if (open && event.key === 'Escape') close();
  }}
/>

<div class="inline-block">
  <button
    bind:this={button}
    type="button"
    class="btn-ghost hit-area px-1.5 text-accent-600 dark:text-accent-400"
    aria-label={label}
    aria-expanded={open}
    title={label}
    onclick={toggle}
  >
    <Icon name={icon} size={icon === 'info' ? 14 : 16} />
  </button>
  {#if open}
    <div
      use:portal
      bind:this={popover}
      {@attach focusOnOpen}
      tabindex="-1"
      class="panel fixed z-50 overflow-y-auto p-4 text-left text-sm font-normal normal-case shadow-hard-lg outline-none"
      style={placement}
      role="dialog"
      aria-label={title}
    >
      <div class="mb-2 flex items-center justify-between gap-2">
        <span class="text-xs font-semibold tracking-wide uppercase">{title}</span>
        <div class="flex">
          {#if task.status === 'done'}
            <button
              type="button"
              class="btn-ghost hit-area px-1.5"
              aria-label="Regenerate"
              title="Regenerate"
              onclick={() => task.run(true)}><Icon name="refresh" size={14} /></button
            >
          {/if}
          <button type="button" class="btn-ghost hit-area px-1.5" aria-label="Close" title="Close" onclick={close}
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
