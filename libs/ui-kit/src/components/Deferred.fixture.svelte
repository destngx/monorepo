<script lang="ts">
  import type { Streamed } from '../load/load';
  import Deferred from './Deferred.svelte';

  // Test fixture: a Deferred section rendering a number (Deferred.test.ts).
  let { data, retry, scope }: { data: Streamed<{ total: number }>; retry?: () => unknown; scope?: unknown } = $props();
</script>

<Deferred {data} {retry} {scope} label="net worth">
  {#snippet children(value)}<p data-testid="value">Total {value.total}</p>{/snippet}
</Deferred>
