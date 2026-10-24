<script lang="ts">
  import type { Connection } from '$lib/state/live.svelte';

  // Whether the page is getting live updates from the laptop, as a dot and a word.
  let { connection }: { connection: Connection } = $props();

  const labels: Record<Connection, string> = { live: 'Live', connecting: 'Connecting…', offline: 'Offline' };
  const dots: Record<Connection, string> = {
    live: 'bg-good-500',
    connecting: 'bg-info-500 motion-safe:animate-pulse',
    offline: 'bg-danger-500',
  };
</script>

<span class="muted inline-flex items-center gap-1.5 text-xs" role="status">
  <span class="size-2 rounded-full {dots[connection]}" aria-hidden="true"></span>{labels[connection]}
</span>
