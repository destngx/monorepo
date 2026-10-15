<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import { cardTagTone } from '$lib/cards/cards';

  // Card tags get the card's colour (WM-TX-3); keywords come from the Cards tab.
  let { tag, cardKeywords }: { tag: string; cardKeywords: string[] } = $props();
  const tone = $derived(cardTagTone(tag, cardKeywords));
</script>

<Badge {tone}>{tag}</Badge>
