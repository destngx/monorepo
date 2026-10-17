<script lang="ts">
  import { maskSegments } from '$lib/ai/moneyMask';
  import { mask } from '$lib/state/mask.svelte';
  import MaskToken from './MaskToken.svelte';

  // Text that may quote amounts (AI prose, headlines, hints): while masked each
  // amount maskMoney would hide is a clickable token; revealed, the text is
  // shown unchanged (WM-UI-2, WM-UI-15). Use maskText for plain strings
  // (attributes, <title>, aria).
  interface Props {
    text: string | null | undefined;
    /** Rendered inside a link: tokens are the pointer-only span form. */
    inLink?: boolean;
  }

  let { text, inLink = false }: Props = $props();
</script>

{#if mask.revealed || !text}{text ??
    ''}{:else}{#each maskSegments(text) as segment, index (index)}{#if segment.masked}<MaskToken
        {inLink}
      />{:else}{segment.text}{/if}{/each}{/if}
