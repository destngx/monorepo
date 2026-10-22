<script lang="ts">
  import { STATUS, usageTone } from '$lib/status';

  interface Props {
    percent: number;
    /** Tailwind background class for the fill; derived from thresholds when omitted. */
    color?: string;
    label?: string;
  }

  let { percent, color, label }: Props = $props();

  const fill = $derived(color ?? STATUS[usageTone(percent)].fill);
</script>

<div
  class="h-2.5 w-full overflow-hidden rounded-sm border border-edge bg-surface"
  role="progressbar"
  aria-label={label}
  aria-valuemin={0}
  aria-valuemax={100}
  aria-valuenow={Math.round(Math.max(0, Math.min(percent, 100)))}
>
  <div class="h-full transition-[width] {fill}" style:width="{Math.max(0, Math.min(percent, 100))}%"></div>
</div>
