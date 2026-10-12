<script lang="ts">
  interface Props {
    percent: number;
    /** Tailwind background class for the fill; derived from thresholds when omitted. */
    color?: string;
    label?: string;
  }

  let { percent, color, label }: Props = $props();

  const fill = $derived(
    color ?? (percent >= 100 ? 'bg-orange-500' : percent >= 80 ? 'bg-amber-400' : 'bg-emerald-500'),
  );
</script>

<div
  class="h-2 w-full overflow-hidden rounded-full bg-slate-900/10 dark:bg-white/10"
  role="progressbar"
  aria-label={label}
  aria-valuemin={0}
  aria-valuemax={100}
  aria-valuenow={Math.round(Math.max(0, Math.min(percent, 100)))}
>
  <div class="h-full rounded-full transition-[width] {fill}" style:width="{Math.max(0, Math.min(percent, 100))}%"></div>
</div>
