<script lang="ts">
  import type { Snippet } from 'svelte';
  import { STATUS, type StatusTone } from '../status';
  import Icon from './Icon.svelte';

  // A small square tag. With `status` it adds the status icon and falls back
  // to the status label, so it never reads by colour alone; without it, it is
  // a neutral tag for categories.
  let { status, children }: { status?: StatusTone; children?: Snippet } = $props();

  const token = $derived(status ? STATUS[status] : null);
</script>

<span class="tag whitespace-nowrap {token ? token.badge : STATUS.neutral.badge}" data-status={status}
  >{#if token}<Icon
      name={token.icon}
      size={12}
    />{/if}{#if children}{@render children()}{:else if token}{token.label}{/if}</span
>
