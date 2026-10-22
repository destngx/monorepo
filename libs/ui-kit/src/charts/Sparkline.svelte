<script lang="ts">
  import { areaPath, linePath, linePoints } from './geometry';
  import { PRIMARY } from './palette';

  // A small trend line. With `label` it is an image with that name; without
  // one it is decoration beside a value that is already written out.
  interface Props {
    values: number[];
    /** Where each value sits along x (e.g. day numbers); defaults to evenly spaced. */
    x?: number[];
    width?: number;
    height?: number;
    /** Space kept clear on every side, in viewBox units. */
    pad?: number;
    color?: string;
    strokeWidth?: number;
    /** A fill under the line that fades out towards the baseline. */
    area?: boolean;
    /** A dot on the last value. */
    dot?: boolean;
    label?: string;
    /** Shown instead of the line when there are fewer than two values. */
    empty?: string;
    /** Stretch to the container's width; the line keeps its stroke width. */
    fluid?: boolean;
    class?: string;
  }

  let {
    values,
    x,
    width = 120,
    height = 32,
    pad = 2,
    color = PRIMARY,
    strokeWidth = 2,
    area = false,
    dot = false,
    label,
    empty,
    fluid = false,
    class: className = '',
  }: Props = $props();

  const points = $derived(linePoints(values, width, height, pad, x));
  const last = $derived(points.at(-1));
  const id = `spark-${Math.random().toString(36).slice(2, 9)}`;
</script>

{#if values.length > 1}
  <svg
    viewBox="0 0 {width} {height}"
    width={fluid ? '100%' : width}
    {height}
    preserveAspectRatio={fluid ? 'none' : undefined}
    role={label ? 'img' : undefined}
    aria-label={label}
    aria-hidden={label ? undefined : 'true'}
    class="overflow-visible {className}"
    data-sparkline
  >
    {#if area}
      <defs>
        <linearGradient {id} x1="0" x2="0" y1="0" y2="1">
          <stop offset="0%" stop-color={color} stop-opacity="0.35" />
          <stop offset="100%" stop-color={color} stop-opacity="0" />
        </linearGradient>
      </defs>
      <path d={areaPath(points, height)} fill="url(#{id})" />
    {/if}
    <path
      d={linePath(points)}
      fill="none"
      stroke={color}
      stroke-width={strokeWidth}
      stroke-linejoin="round"
      stroke-linecap="round"
      vector-effect={fluid ? 'non-scaling-stroke' : undefined}
    />
    {#if dot && last}
      <circle cx={last[0]} cy={last[1]} r="3.5" fill={color} stroke="var(--color-surface)" stroke-width="2" />
    {/if}
  </svg>
{:else if empty}
  <span class="muted text-xs">{empty}</span>
{/if}
