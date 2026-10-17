<script lang="ts">
  import type { Snippet } from 'svelte';
  import { MASK } from '$lib/format';
  import { MASK_TOKEN_CLASS, MASK_TOKEN_LABEL, MASK_TOKEN_TITLE, maskToggle } from './maskTokens';

  // The clickable mask (WM-UI-2, WM-UI-15); see maskTokens.ts.
  interface Props {
    /** Inside a link or other interactive element: a pointer-only span, not a nested button. */
    inLink?: boolean;
    /** Accessible name; defaults to "Hidden amount, click to reveal". */
    label?: string;
    class?: string;
    /** Inline style, e.g. the position of an overlay. */
    style?: string;
    /** Replaces the visible •••••• (e.g. an empty hit area over a chart axis). */
    children?: Snippet;
  }

  let { inLink = false, label = MASK_TOKEN_LABEL, class: className = '', style, children }: Props = $props();
</script>

{#if inLink}
  <!-- Pointer-only by design: a focusable control inside a link is nested interactive content. -->
  <span data-mask-toggle class="{MASK_TOKEN_CLASS} {className}" {style} title={MASK_TOKEN_TITLE} {@attach maskToggle}
    >{#if children}{@render children()}{:else}{MASK}{/if}</span
  >
{:else}
  <button
    type="button"
    data-mask-toggle
    class="{MASK_TOKEN_CLASS} {className}"
    {style}
    aria-label={label}
    title={MASK_TOKEN_TITLE}
    {@attach maskToggle}
    >{#if children}{@render children()}{:else}{MASK}{/if}</button
  >
{/if}
