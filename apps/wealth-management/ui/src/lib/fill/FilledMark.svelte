<script lang="ts">
  import { Icon } from '@libs/ui-kit';

  // Marks a field AI filled until it is edited (WM-FILL-6).
  let { show }: { show: boolean } = $props();
</script>

{#if show}
  <span
    class="ml-1 inline-flex items-center gap-0.5 align-middle text-[10px] font-semibold tracking-wide text-accent-700 normal-case dark:text-accent-300"
    title="Filled by AI. Check it before saving."
  >
    <Icon name="sparkles" size={11} />
    <span>AI</span>
    <span class="sr-only">filled, check before saving</span>
  </span>
{/if}
