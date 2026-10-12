<script lang="ts">
  import { MASK } from '$lib/format';
  import { mask } from '$lib/state/mask.svelte';

  // Sensitive text (lender names) that hides with the privacy mask.
  let { text, class: className = '' }: { text: string; class?: string } = $props();
</script>

<span class={className}>{mask.revealed ? text : MASK}</span>
