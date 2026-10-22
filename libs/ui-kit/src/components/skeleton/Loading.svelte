<script lang="ts">
  import type { Snippet } from 'svelte';

  // The accessible frame of every skeleton: a busy status with screen-reader
  // text, and the placeholder shapes hidden from assistive tech.
  interface Props {
    /** What is loading, e.g. "net worth" ("Loading net worth…"). */
    label?: string;
    class?: string;
    children: Snippet;
  }

  let { label = 'content', class: className = '', children }: Props = $props();
</script>

<div role="status" aria-busy="true" aria-live="polite" class={className} data-skeleton>
  <span class="sr-only">Loading {label}…</span>
  <div aria-hidden="true">{@render children()}</div>
</div>
