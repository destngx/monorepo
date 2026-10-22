<script lang="ts">
  import type { Snippet } from 'svelte';
  import { arcPath } from './geometry';

  interface Props {
    /** 0..100 */
    percent: number;
    color: string;
    size?: number;
    thickness?: number;
    label?: string;
    children?: Snippet;
  }

  let { percent, color, size = 140, thickness = 12, label = 'Progress', children }: Props = $props();

  const r = $derived(size / 2);
  const end = $derived((Math.max(0, Math.min(percent, 100)) / 100) * Math.PI * 2);
</script>

<div class="relative inline-block" style:width="{size}px" style:height="{size}px">
  <svg viewBox="0 0 {size} {size}" width={size} height={size} role="img" aria-label="{label}: {Math.round(percent)}%">
    <circle
      cx={r}
      cy={r}
      r={r - thickness / 2}
      fill="none"
      stroke-width={thickness}
      class="stroke-ink-900/10 dark:stroke-ink-50/10"
    />
    {#if end > 0}
      <path d={arcPath(r, r, r, r - thickness, 0, end)} fill={color} />
    {/if}
  </svg>
  {#if children}
    <div class="absolute inset-0 flex flex-col items-center justify-center text-center">{@render children()}</div>
  {/if}
</div>
