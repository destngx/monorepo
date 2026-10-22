<script lang="ts">
  import { base } from '$app/paths';
  import type { AIUsageWindow } from '$lib/api/types';
  import { formatDateTime, formatPercent } from '$lib/format';
  import { gaugeWindows, leftColor, leftPercent, resetsIn, windowLabel } from './usage';
  import { AI_USAGE_REFRESH_MS, aiUsage } from './usage.svelte';

  // Menu bar gauge of the OpenAI quota left (WM-SET-4): the outer ring is the
  // weekly window, the inner ring the 5-hour window; each arc is the share
  // left, coloured from green (plenty) to red (none).
  const RADIUS = { outer: 13, inner: 8 } as const;
  const STROKE = 3.5;

  type Ring = { key: 'outer' | 'inner'; radius: number; window: AIUsageWindow };

  function toRings(windows: AIUsageWindow[]): Ring[] {
    const { weekly, fiveHour } = gaugeWindows(windows);
    const rings: Ring[] = [];
    if (weekly) rings.push({ key: 'outer', radius: RADIUS.outer, window: weekly });
    if (fiveHour) rings.push({ key: 'inner', radius: RADIUS.inner, window: fiveHour });
    return rings;
  }

  const rings = $derived(toRings(aiUsage.usage?.windows ?? []));
  const unavailable = $derived(aiUsage.error !== '');
  // API-key accounts report token totals, not limits: nothing to gauge.
  const visible = $derived(rings.length > 0 || unavailable);
  const summary = $derived(
    unavailable
      ? `AI usage unavailable: ${aiUsage.error}`
      : `AI quota left: ${rings.map((ring) => `${windowLabel(ring.window)} ${formatPercent(leftPercent(ring.window), 0)}`).join(', ')}`,
  );

  $effect(() => {
    void aiUsage.load();
    const refresh = () => {
      if (document.visibilityState === 'visible') void aiUsage.load();
    };
    const onVisible = () => {
      if (Date.now() - aiUsage.loadedAt.getTime() >= AI_USAGE_REFRESH_MS) refresh();
    };
    const timer = setInterval(refresh, AI_USAGE_REFRESH_MS);
    document.addEventListener('visibilitychange', onVisible);
    return () => {
      clearInterval(timer);
      document.removeEventListener('visibilitychange', onVisible);
    };
  });
</script>

{#if visible}
  <a
    href="{base}/settings"
    class="group relative grid size-9 place-items-center rounded-xl hover:bg-stone-900/5 dark:hover:bg-white/5"
    aria-label={summary}
  >
    <svg viewBox="0 0 32 32" class="size-7 -rotate-90" aria-hidden="true">
      {#if unavailable}
        {#each [RADIUS.outer, RADIUS.inner] as radius (radius)}
          <circle
            cx="16"
            cy="16"
            r={radius}
            fill="none"
            stroke="currentColor"
            stroke-width={STROKE}
            stroke-dasharray="2 3"
            class="opacity-30"
          />
        {/each}
      {:else}
        {#each rings as ring (ring.key)}
          {@const left = leftPercent(ring.window)}
          <circle
            cx="16"
            cy="16"
            r={ring.radius}
            fill="none"
            stroke="currentColor"
            stroke-width={STROKE}
            class="opacity-10"
          />
          {#if left > 0}
            <circle
              cx="16"
              cy="16"
              r={ring.radius}
              fill="none"
              stroke={leftColor(left)}
              stroke-width={STROKE}
              stroke-linecap="round"
              pathLength="100"
              stroke-dasharray="{left} 100"
              class="transition-[stroke-dasharray]"
            />
          {/if}
        {/each}
      {/if}
    </svg>

    <!-- Details on hover and keyboard focus -->
    <div
      class="panel pointer-events-none absolute top-full right-0 z-40 mt-2 hidden w-64 p-3 text-left text-xs group-hover:block group-focus-visible:block"
      aria-hidden="true"
    >
      <p class="mb-2 font-semibold">
        AI quota left{#if aiUsage.usage?.provider}&nbsp;({aiUsage.usage.provider}){/if}
      </p>
      {#if unavailable}
        <p class="text-rose-600 dark:text-rose-400">Unavailable: {aiUsage.error}</p>
      {:else}
        <ul class="space-y-2">
          {#each rings as ring (ring.key)}
            {@const left = leftPercent(ring.window)}
            {@const reset = resetsIn(ring.window.resetsAt, aiUsage.loadedAt)}
            <li>
              <div class="flex items-center justify-between gap-2">
                <span class="flex items-center gap-1.5">
                  <span class="size-2.5 rounded-full" style:background-color={leftColor(left)}></span>
                  {windowLabel(ring.window)}
                  <span class="muted">({ring.key} ring)</span>
                </span>
                <span class="tabular font-medium">{formatPercent(left, 0)} left</span>
              </div>
              {#if ring.window.resetsAt}
                <p class="muted mt-0.5 pl-4">
                  Resets <span class="tabular">{formatDateTime(ring.window.resetsAt)}</span
                  >{#if reset}&nbsp;({reset}){/if}
                </p>
              {/if}
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  </a>
{/if}
