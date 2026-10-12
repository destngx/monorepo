<script lang="ts">
  import { radarPoints } from './geometry';

  interface Props {
    axes: { label: string; fraction: number }[];
    size?: number;
    label?: string;
  }

  let { axes, size = 280, label = 'Radar' }: Props = $props();

  const c = $derived(size / 2);
  const radius = $derived(size / 2 - 48);
  const polygon = (fractions: number[]) =>
    radarPoints(fractions, c, c, radius)
      .map(([x, y]) => `${x},${y}`)
      .join(' ');
  const rings = [0.25, 0.5, 0.75, 1];
  const outer = $derived(
    radarPoints(
      axes.map(() => 1),
      c,
      c,
      radius,
    ),
  );
  const labelPoints = $derived(
    radarPoints(
      axes.map(() => 1),
      c,
      c,
      radius + 20,
    ),
  );
</script>

{#if axes.length >= 3}
  <svg viewBox="0 0 {size} {size}" class="h-auto w-full max-w-[320px]" role="img" aria-label={label}>
    {#each rings as ring (ring)}
      <polygon points={polygon(axes.map(() => ring))} fill="none" class="stroke-slate-900/10 dark:stroke-white/10" />
    {/each}
    {#each outer as [x, y], i (i)}
      <line x1={c} y1={c} x2={x} y2={y} class="stroke-slate-900/10 dark:stroke-white/10" />
    {/each}
    <polygon
      points={polygon(axes.map((a) => a.fraction))}
      fill="#6366f1"
      fill-opacity="0.25"
      stroke="#6366f1"
      stroke-width="2"
      stroke-linejoin="round"
    />
    {#each labelPoints as [x, y], i (i)}
      <text
        {x}
        {y}
        dy="0.32em"
        text-anchor={Math.abs(x - c) < 4 ? 'middle' : x > c ? 'start' : 'end'}
        class="fill-slate-600 text-[11px] dark:fill-slate-300">{axes[i].label}</text
      >
    {/each}
  </svg>
{/if}
