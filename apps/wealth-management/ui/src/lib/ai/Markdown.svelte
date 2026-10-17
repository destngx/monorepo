<script lang="ts">
  import { maskTokenClicks, tokenizeMaskedHtml } from '$lib/components/maskTokens';
  import { mask } from '$lib/state/mask.svelte';
  import { renderMarkdown } from './markdown';
  import { maskText } from './moneyMask';

  let { text, class: className = '' }: { text: string; class?: string } = $props();

  // Model prose can quote amounts; they hide with the privacy mask (WM-UI-2)
  // and each hidden amount is a clickable token (WM-UI-15), added after the
  // markdown is rendered and escaped.
  const html = $derived(tokenizeMaskedHtml(renderMarkdown(maskText(text, mask.revealed))));
</script>

<!-- renderMarkdown escapes all input and tokenizeMaskedHtml only swaps the
     constant mask for constant markup; see markdown.ts and maskTokens.ts. The
     tokens are native buttons, so one delegated click listener is enough. -->
<!-- eslint-disable-next-line svelte/no-at-html-tags -->
<div class="md {className}" {@attach maskTokenClicks}>{@html html}</div>
