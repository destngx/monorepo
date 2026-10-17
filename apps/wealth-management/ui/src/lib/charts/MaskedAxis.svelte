<script lang="ts">
  import MaskToken from '$lib/components/MaskToken.svelte';
  import { mask } from '$lib/state/mask.svelte';

  // While amounts are masked, the y-axis gutter of a chart is one clickable
  // mask token (WM-UI-15): an HTML button laid over the masked tick labels, so
  // it is keyboard reachable and sits outside the chart's role="img" (whose
  // children are presentational). It covers only the gutter, never the plot,
  // so the hover crosshair is unaffected. Place it in a `relative` wrapper with
  // the svg's exact size.
  interface Props {
    /** The svg viewBox size and padding, to place the gutter. */
    width: number;
    height: number;
    pad: { top: number; bottom: number; left: number };
  }

  let { width, height, pad }: Props = $props();

  // The tick labels sit on the plot's top and bottom edges, so reach half a line past them.
  const style = $derived(
    [
      'margin:0',
      'padding:0',
      'left:0',
      `top:${(Math.max(pad.top - 8, 0) / height) * 100}%`,
      `height:${((height - pad.top - pad.bottom + 16) / height) * 100}%`,
      `width:${(pad.left / width) * 100}%`,
    ].join(';'),
  );
</script>

{#if !mask.revealed}
  <MaskToken label="Hidden axis amounts, click to reveal" class="absolute" {style}>
    {#snippet children()}{/snippet}
  </MaskToken>
{/if}
