<script lang="ts">
  import { clampScore } from '../news';

  // A small labelled 0-100 meter for model confidence and impact.
  let { label, value }: { label: string; value: number } = $props();
  const percent = $derived(clampScore(value));
</script>

<div class="flex items-center gap-1.5 text-xs" title="{label} {Math.round(percent)}/100">
  <span class="muted w-16 shrink-0">{label}</span>
  <div
    class="h-1.5 w-16 overflow-hidden rounded-full bg-slate-900/10 dark:bg-white/10"
    role="meter"
    aria-label={label}
    aria-valuemin={0}
    aria-valuemax={100}
    aria-valuenow={Math.round(percent)}
  >
    <div class="h-full rounded-full bg-indigo-500" style:width="{percent}%"></div>
  </div>
  <span class="tabular w-8 text-right">{Math.round(percent)}%</span>
</div>
