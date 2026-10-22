<script lang="ts">
  import { Sparkline } from '@libs/ui-kit';
  import { EXPENSE, INCOME } from './palette';

  // A trend coloured by its direction: income green when it ends at or above where it started.
  interface Props {
    values: number[];
    width?: number;
    height?: number;
    label?: string;
    /** Stretch to the container's width; the line keeps its stroke width. */
    fluid?: boolean;
  }

  let { values, width = 120, height = 32, label = 'Trend', fluid = false }: Props = $props();

  const rising = $derived(values.length < 2 || values[values.length - 1] >= values[0]);
</script>

<Sparkline
  {values}
  {width}
  {height}
  {label}
  {fluid}
  color={rising ? INCOME : EXPENSE}
  strokeWidth={1.75}
  area
  empty="No history"
/>
