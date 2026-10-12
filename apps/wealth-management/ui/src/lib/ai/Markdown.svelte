<script lang="ts">
  import { mask } from '$lib/state/mask.svelte';
  import { renderMarkdown } from './markdown';
  import { maskText } from './moneyMask';

  let { text, class: className = '' }: { text: string; class?: string } = $props();

  // Model prose can quote amounts; they hide with the privacy mask (WM-UI-2).
  const html = $derived(renderMarkdown(maskText(text, mask.revealed)));
</script>

<!-- renderMarkdown escapes all input; see markdown.ts -->
<!-- eslint-disable-next-line svelte/no-at-html-tags -->
<div class="md {className}">{@html html}</div>
