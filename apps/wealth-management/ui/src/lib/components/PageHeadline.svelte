<script lang="ts">
  import type { Snippet } from 'svelte';
  import { STATUS, type StatusTone } from '$lib/status';
  import Icon from './Icon.svelte';

  // The page hierarchy (ROADMAP P2): every page leads with one headline answer
  // to its question, then the evidence (children), then the detail below it.
  // The status shows as the edge colour with its icon, never by colour alone.
  interface Props {
    /** The question the page answers, e.g. "Am I okay?". */
    question: string;
    tone: StatusTone;
    /** The answer as text; use `answer` for one with amounts. */
    title?: string;
    answer?: Snippet;
    /** Evidence under the answer. */
    children?: Snippet;
    /** Controls on the right of the question, e.g. a status badge. */
    actions?: Snippet;
    /** Id of the answer heading (the section's label); unique per page. */
    id?: string;
    class?: string;
  }

  let {
    question,
    tone,
    title,
    answer,
    children,
    actions,
    id = 'page-headline',
    class: className = '',
  }: Props = $props();
  const style = $derived(STATUS[tone]);
</script>

<section class="panel border-l-[6px] p-5 {style.edge} {className}" aria-labelledby={id} data-status={tone}>
  <div class="flex flex-wrap items-start justify-between gap-2">
    <p class="muted text-[0.7rem] font-semibold tracking-[0.12em] uppercase">{question}</p>
    {#if actions}<div class="flex flex-wrap items-center gap-2">{@render actions()}</div>{/if}
  </div>
  <h2 {id} class="mt-1 flex flex-wrap items-center gap-2 font-serif text-xl font-semibold {style.ink}">
    <Icon name={style.icon} />
    <span class="sr-only">{style.label}:</span>
    {#if answer}{@render answer()}{:else}{title}{/if}
  </h2>
  {#if children}<div class="mt-1 text-sm">{@render children()}</div>{/if}
</section>
