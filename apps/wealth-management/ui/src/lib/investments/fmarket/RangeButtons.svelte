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

<div class="flex flex-wrap gap-0.5 rounded-xl bg-stone-900/5 p-0.5 dark:bg-white/5" role="group" aria-label="Range">
  {#each ranges as range (range)}
    <button
      type="button"
      class="rounded-lg px-2 py-1 text-xs font-medium transition active:scale-95 disabled:opacity-50 {range === value
        ? 'bg-stone-900 text-stone-50 dark:bg-stone-100 dark:text-stone-900'
        : 'muted hover:bg-white/60 hover:text-stone-900 dark:hover:bg-white/10 dark:hover:text-white'}"
      aria-pressed={range === value}
      {disabled}
      onclick={() => onchange(range)}>{range}</button
    >
  {/each}
</div>
