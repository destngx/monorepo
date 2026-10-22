<script lang="ts">
  import {
    AI_USAGE_REFRESH_MS,
    gaugeWindows,
    leftPercent,
    resetsIn,
    windowLabel,
    type AIUsageSource,
    type AIUsageWindow,
    type FormatReset,
    type LeftColor,
  } from './usage';

  // Menu bar gauge of the AI quota left: the outer ring is the weekly window,
  // the inner ring the 5-hour window; each arc is the share left, coloured by
  // the app's scale from plenty to none. A tap or click opens the details, so
  // they work on a phone too; Escape or a tap elsewhere closes them.
  //
  // Styling relies on the app's `panel`, `muted`, `link`, `tabular` and
  // `icon-btn` utilities, and the app's Tailwind must scan this library.
  interface Props {
    source: AIUsageSource;
    /** See leftColorScale. */
    leftColor: LeftColor;
    formatReset: FormatReset;
    /** Classes for the "Unavailable" line. */
    errorClass: string;
    /** Link to the AI settings in the details; omitted when absent. */
    settingsHref?: string;
    /** Classes for the wrapper the details are positioned against. */
    class?: string;
  }

  let { source, leftColor, formatReset, errorClass, settingsHref, class: rootClass = 'relative' }: Props = $props();

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

  const percent = (left: number) => `${left.toFixed(0)}%`;

  const enabled = $derived(source.enabled ?? true);
  const rings = $derived(toRings(source.usage?.windows ?? []));
  const unavailable = $derived(enabled && source.error !== '');
  // API-key accounts report token totals, not limits: nothing to gauge.
  const visible = $derived(enabled && (rings.length > 0 || unavailable));
  const summary = $derived(
    unavailable
      ? `AI usage unavailable: ${source.error}`
      : `AI quota left: ${rings.map((ring) => `${windowLabel(ring.window)} ${percent(leftPercent(ring.window))}`).join(', ')}`,
  );

  let open = $state(false);
  let root: HTMLElement | undefined = $state();
  const panelId = $props.id();

  $effect(() => {
    if (!open) return;
    const onPointer = (event: PointerEvent) => {
      if (root && !root.contains(event.target as Node)) open = false;
    };
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') open = false;
    };
    window.addEventListener('pointerdown', onPointer);
    window.addEventListener('keydown', onKey);
    return () => {
      window.removeEventListener('pointerdown', onPointer);
      window.removeEventListener('keydown', onKey);
    };
  });

  $effect(() => {
    void source.load();
    const refresh = () => {
      if (document.visibilityState === 'visible') void source.load();
    };
    const onVisible = () => {
      if (Date.now() - source.loadedAt.getTime() >= AI_USAGE_REFRESH_MS) refresh();
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
  <div class={rootClass} bind:this={root}>
    <button
      type="button"
      class="icon-btn size-9"
      aria-label={summary}
      aria-expanded={open}
      aria-controls={panelId}
      onclick={() => (open = !open)}
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
    </button>

    {#if open}
      <div
        id={panelId}
        class="panel absolute top-full right-0 z-40 mt-2 w-72 max-w-[calc(100vw-1.5rem)] p-3 text-left text-xs"
      >
        <p class="mb-2 font-semibold">
          AI quota left{#if source.usage?.provider}&nbsp;({source.usage.provider}){/if}
        </p>
        {#if unavailable}
          <p class={errorClass}>Unavailable: {source.error}</p>
        {:else}
          <ul class="space-y-2">
            {#each rings as ring (ring.key)}
              {@const left = leftPercent(ring.window)}
              {@const reset = resetsIn(ring.window.resetsAt, source.loadedAt)}
              <li>
                <div class="flex items-center justify-between gap-2">
                  <span class="flex items-center gap-1.5">
                    <span class="size-2.5 rounded-full" style:background-color={leftColor(left)}></span>
                    {windowLabel(ring.window)}
                    <span class="muted">({ring.key} ring)</span>
                  </span>
                  <span class="tabular font-medium">{percent(left)} left</span>
                </div>
                {#if ring.window.resetsAt}
                  <p class="muted mt-0.5 pl-4">
                    Resets <span class="tabular">{formatReset(ring.window.resetsAt)}</span
                    >{#if reset}&nbsp;({reset}){/if}
                  </p>
                {/if}
              </li>
            {/each}
          </ul>
        {/if}
        {#if settingsHref}
          <a class="link mt-3 inline-block" href={settingsHref} onclick={() => (open = false)}>AI settings</a>
        {/if}
      </div>
    {/if}
  </div>
{/if}
