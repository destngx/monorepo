<script lang="ts">
  import { STATUS, usageTone } from '$lib/status';

  interface Props {
    percent: number;
    /** Tailwind background class for the fill; derived from thresholds when omitted. */
    color?: string;
    label?: string;
    /** `sm` is the thin bar for dense lists. */
    size?: 'sm' | 'md';
  }

  let { percent, color, label, size = 'md' }: Props = $props();

  const fill = $derived(color ?? STATUS[usageTone(percent)].fill);
</script>

<div
  class="{size === 'sm' ? 'h-1.5' : 'h-2.5'} w-full overflow-hidden rounded-sm border border-edge bg-surface"
  role="progressbar"
  aria-label={label}
  aria-valuemin={0}
  aria-valuemax={100}
  aria-valuenow={Math.round(Math.max(0, Math.min(percent, 100)))}
>
  <div class="h-full transition-[width] {fill}" style:width="{Math.max(0, Math.min(percent, 100))}%"></div>
</div>
