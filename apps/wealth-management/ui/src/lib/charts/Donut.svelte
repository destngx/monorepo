<script lang="ts">
  import type { Snippet } from 'svelte';
  import { arcPath, slices } from './geometry';
  import { PALETTE } from './palette';

  interface Props {
    items: { label: string; value: number; color?: string }[];
    size?: number;
    thickness?: number;
    label?: string;
    center?: Snippet;
  }

  let { items, size = 180, thickness = 26, label = 'Breakdown', center }: Props = $props();

  const r = $derived(size / 2);
  const arcs = $derived(
    slices(items.map((item) => item.value)).map((slice, i) => ({
      ...slice,
      ...items[i],
      color: items[i].color ?? PALETTE[i % PALETTE.length],
    })),
  );
</script>

<div class="relative inline-block" style:width="{size}px" style:height="{size}px">
  <svg viewBox="0 0 {size} {size}" width={size} height={size} role="img" aria-label={label}>
    <circle
      cx={r}
      cy={r}
      r={r - thickness / 2}
      fill="none"
      stroke-width={thickness}
      class="stroke-slate-900/5 dark:stroke-white/5"
    />
    {#each arcs as arc (arc.label)}
      {#if arc.fraction > 0}
        <path d={arcPath(r, r, r, r - thickness, arc.start, arc.end)} fill={arc.color}>
          <title>{arc.label}: {(arc.fraction * 100).toFixed(1)}%</title>
        </path>
      {/if}
    {/each}
  </svg>
  {#if center}
    <div class="absolute inset-0 flex flex-col items-center justify-center text-center">{@render center()}</div>
  {/if}
</div>
