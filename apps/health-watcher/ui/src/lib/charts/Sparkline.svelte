<script lang="ts">
  import type { TrendPoint } from '$lib/api/types';
  import { extent, linePath, scale, toDay } from './geometry';
  import { SERIES } from './palette';

  // A small trend beside a value that is already written out, so it is decorative.
  let { points, width = 120, height = 32 }: { points: TrendPoint[]; width?: number; height?: number } = $props();

  const pad = 4;
  const coords = $derived.by(() => {
    const numeric = points.filter((p) => p.value !== undefined);
    const days = numeric.map((p) => toDay(p.date));
    const x = scale([days[0], days.at(-1) ?? days[0]], [pad, width - pad]);
    const y = scale(extent(numeric.map((p) => p.value as number)), [height - pad, pad]);
    return numeric.map((p, i) => [x(days[i]), y(p.value as number)] as [number, number]);
  });
</script>

{#if coords.length > 1}
  <svg viewBox="0 0 {width} {height}" {width} {height} class="block overflow-visible" aria-hidden="true">
    <path
      d={linePath(coords)}
      fill="none"
      stroke={SERIES}
      stroke-width="2"
      stroke-linejoin="round"
      stroke-linecap="round"
    />
    <circle
      cx={coords.at(-1)?.[0]}
      cy={coords.at(-1)?.[1]}
      r="3.5"
      fill={SERIES}
      stroke="var(--surface)"
      stroke-width="2"
    />
  </svg>
{/if}
