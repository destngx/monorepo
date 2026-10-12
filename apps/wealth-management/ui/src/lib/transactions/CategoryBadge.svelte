<script lang="ts">
  import Badge from '$lib/components/Badge.svelte';
  import type { CategoryType } from '$lib/api/types';

  let { name, type }: { name: string; type?: CategoryType } = $props();
  const tone = $derived(type === 'income' ? 'green' : type === 'expense' ? 'orange' : 'neutral');
</script>

<Badge {tone}>{name || 'Uncategorised'}</Badge>
