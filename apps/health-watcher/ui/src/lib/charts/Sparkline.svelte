<script lang="ts">
  import { Sparkline, toDay } from '@libs/ui-kit';
  import type { TrendPoint } from '$lib/api/types';
  import { SERIES } from './palette';

  // A small trend beside a value that is already written out, so it is decorative.
  let { points, width = 120, height = 32 }: { points: TrendPoint[]; width?: number; height?: number } = $props();

  const numeric = $derived(points.filter((p) => p.value !== undefined));
</script>

<Sparkline
  values={numeric.map((p) => p.value as number)}
  x={numeric.map((p) => toDay(p.date))}
  {width}
  {height}
  pad={4}
  color={SERIES}
  dot
  class="block"
/>
