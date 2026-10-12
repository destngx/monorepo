<script lang="ts">
  import type { PulseQuote, Timeframe } from '$lib/api/types';
  import { formatSignedPercent } from '$lib/format';
  import { changeClass, formatPrice } from '../market';
  import { symbolHref } from './link';

  // Largest moves over the timeframe.
  interface Props {
    movers: PulseQuote[];
    timeframe: Timeframe;
  }

  let { movers, timeframe }: Props = $props();
</script>

<div class="mb-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
  {#each movers as mover (mover.symbol)}
    <a
      href={symbolHref(mover.symbol, timeframe)}
      class="glass block p-3 transition hover:ring-2 hover:ring-indigo-500/40"
    >
      <p class="muted text-xs font-medium tracking-wide uppercase">Top mover</p>
      <p class="mt-1 flex items-baseline justify-between gap-2">
        <span class="font-semibold">{mover.symbol}</span>
        <span class="tabular text-sm font-semibold {changeClass(mover.changePct)}"
          >{formatSignedPercent(mover.changePct, 2)}</span
        >
      </p>
      <p class="muted truncate text-xs">{mover.name} · {formatPrice(mover.price)}</p>
    </a>
  {/each}
</div>
