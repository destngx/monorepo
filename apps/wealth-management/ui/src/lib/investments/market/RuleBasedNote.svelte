<script lang="ts">
  import Icon from '$lib/components/Icon.svelte';

  // The one disclaimer a market page shows for its rule-based readings; cards do not repeat it.
  let { class: className = '' }: { class?: string } = $props();
</script>

<p class="muted flex items-start gap-2 text-xs {className}" data-testid="rule-based-note">
  <Icon name="info" size={14} class="mt-px" />
  <span
    >Signals and readings on this page are rule-based, computed from past prices: not a forecast and not advice.</span
  >
</p>
