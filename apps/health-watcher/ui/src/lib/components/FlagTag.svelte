<script lang="ts">
  import { flagLabel } from '$lib/format';
  import { Icon, type IconName } from '@libs/ui-kit';

  // The paper's own flag (HW-MET-5) as an icon and a word, never colour alone.
  let { flag }: { flag?: string } = $props();

  const tone: Record<string, string> = {
    H: 'text-danger-700 dark:text-danger-300',
    L: 'text-info-700 dark:text-info-300',
    '*': 'text-warn-700 dark:text-warn-300',
  };
  const icon: Record<string, IconName> = { H: 'arrowUp', L: 'arrowDown', '*': 'alert' };
</script>

{#if flag}
  <span class="tag shrink-0 {tone[flag] ?? ''}" title="Flagged on the paper"
    ><Icon name={icon[flag] ?? 'alert'} size={12} />{flagLabel(flag)}</span
  >
{/if}
