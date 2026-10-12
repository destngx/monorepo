<script lang="ts">
  import { RANGES, type Range, type RangeHandler } from '../fmarket';

  interface Props {
    value: Range;
    onchange: RangeHandler;
    ranges?: Range[];
    disabled?: boolean;
  }

  let { value, onchange, ranges = RANGES, disabled = false }: Props = $props();
</script>

<div class="flex flex-wrap gap-0.5 rounded-xl bg-slate-900/5 p-0.5 dark:bg-white/5" role="group" aria-label="Range">
  {#each ranges as range (range)}
    <button
      type="button"
      class="rounded-lg px-2 py-1 text-xs font-medium transition {range === value
        ? 'bg-white text-indigo-600 shadow-sm dark:bg-slate-800 dark:text-indigo-300'
        : 'muted hover:text-slate-900 dark:hover:text-white'}"
      aria-pressed={range === value}
      {disabled}
      onclick={() => onchange(range)}>{range}</button
    >
  {/each}
</div>
