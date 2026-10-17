<script lang="ts">
  import { TIMEFRAMES, type Timeframe } from '$lib/api/types';
  import { TIMEFRAME_LABELS } from '../market';

  // Bind with a getter/setter pair to route changes through the URL.
  let { value = $bindable() }: { value: Timeframe } = $props();
</script>

<div class="glass inline-flex p-1" role="group" aria-label="Timeframe">
  {#each TIMEFRAMES as timeframe (timeframe)}
    <button
      type="button"
      class="rounded-lg px-3 py-1 text-sm font-medium transition active:scale-95 {timeframe === value
        ? 'bg-indigo-600 text-white shadow-sm hover:bg-indigo-500'
        : 'text-slate-700 hover:bg-slate-900/5 dark:text-slate-200 dark:hover:bg-white/10'}"
      aria-pressed={timeframe === value}
      onclick={() => (value = timeframe)}>{TIMEFRAME_LABELS[timeframe]}</button
    >
  {/each}
</div>
