<script lang="ts">
  import { areaPath, linePath, linePoints } from './geometry';

  interface Props {
    values: number[];
    width?: number;
    height?: number;
    label?: string;
  }

  let { values, width = 120, height = 32, label = 'Trend' }: Props = $props();

  const points = $derived(linePoints(values, width, height));
  const rising = $derived(values.length < 2 || values[values.length - 1] >= values[0]);
  const color = $derived(rising ? '#10b981' : '#f97316');
  const id = `spark-${Math.random().toString(36).slice(2, 9)}`;
</script>

{#if values.length > 1}
  <svg viewBox="0 0 {width} {height}" {width} {height} role="img" aria-label={label} class="overflow-visible">
    <defs>
      <linearGradient {id} x1="0" x2="0" y1="0" y2="1">
        <stop offset="0%" stop-color={color} stop-opacity="0.35" />
        <stop offset="100%" stop-color={color} stop-opacity="0" />
      </linearGradient>
    </defs>
    <path d={areaPath(points, height)} fill="url(#{id})" />
    <path d={linePath(points)} fill="none" stroke={color} stroke-width="1.75" stroke-linejoin="round" />
  </svg>
{:else}
  <span class="muted text-xs">No history</span>
{/if}
