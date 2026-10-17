<script lang="ts">
  import { mask } from '$lib/state/mask.svelte';
  import MaskToken from './MaskToken.svelte';

  // A value hidden whole by the privacy mask: sensitive text (lender names,
  // memos) or a preformatted amount (chart readouts, compact table cells).
  // Masked, it is a clickable token that reveals everything (WM-UI-2).
  interface Props {
    text: string;
    /** Rendered inside a link: the token is the pointer-only span form. */
    inLink?: boolean;
    class?: string;
  }

  let { text, inLink = false, class: className = '' }: Props = $props();
</script>

{#if mask.revealed}<span class={className}>{text}</span>{:else}<MaskToken {inLink} class={className} />{/if}
