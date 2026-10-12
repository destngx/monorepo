<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';

  // Card tags get the card's colour (WM-TX-3): UNIQ green, Platinum blue.
  let { tag }: { tag: string } = $props();
  const lower = $derived(tag.toLowerCase());
  const tone = $derived(lower.includes('uniq') ? 'green' : lower.includes('platinum') ? 'blue' : 'neutral');
</script>

<Badge {tone}>{tag}</Badge>
