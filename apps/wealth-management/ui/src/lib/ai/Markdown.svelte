<script lang="ts">
  import { renderMarkdown } from './markdown';

  let { text, class: className = '' }: { text: string; class?: string } = $props();
</script>

<!-- renderMarkdown escapes all input; see markdown.ts -->
<!-- eslint-disable-next-line svelte/no-at-html-tags -->
<div class="md {className}">{@html renderMarkdown(text)}</div>
