<script lang="ts">
  import Button from '$lib/components/Button.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import SkeletonText from '$lib/components/skeleton/SkeletonText.svelte';
  import ProgressBar from '$lib/components/ProgressBar.svelte';
  import { formatCompact, formatDate, formatDateTime, formatPercent } from '$lib/format';
  import { resetsIn, windowLabel } from '@libs/ui-kit';
  import { usageColor } from './usage';
  import { aiUsage } from './usage.svelte';

  // OpenAI quota through the ai-gateway GET /v1/usage (WM-SET-4).
  const usage = $derived(aiUsage.usage);
  const error = $derived(aiUsage.error);
  const loading = $derived(aiUsage.loading);
  const loadedAt = $derived(aiUsage.loadedAt);
  const load = () => void aiUsage.load();

  $effect(() => {
    load();
  });
</script>

<section class="mt-4" aria-busy={loading}>
  <div class="flex items-center justify-between gap-2">
    <h3 class="label">
      Provider usage{#if usage?.provider}&nbsp;({usage.provider}){/if}
    </h3>
    <Button size="sm" icon="refresh" busy={loading} busyText="Refreshing…" onclick={load}>Refresh usage</Button>
  </div>
  {#if usage}
    {#if usage.windows.length}
      <ul class="mt-2 space-y-2.5 text-sm">
        {#each usage.windows as window (window.name)}
          {@const reset = resetsIn(window.resetsAt, loadedAt)}
          <li>
            <div class="mb-1 flex items-baseline justify-between gap-2">
              <span>{windowLabel(window)}</span>
              <span class="tabular text-xs">
                {formatPercent(window.usedPercent, 0)} used
                <span class="muted">· {formatPercent(Math.max(0, 100 - window.usedPercent), 0)} left</span>
              </span>
            </div>
            <ProgressBar
              percent={window.usedPercent}
              color={usageColor(window.usedPercent)}
              label="{windowLabel(window)} used"
            />
            {#if window.resetsAt}
              <p class="muted mt-1 text-xs">
                Resets <time class="tabular" datetime={window.resetsAt}>{formatDateTime(window.resetsAt)}</time
                >{#if reset}&nbsp;({reset}){/if}
              </p>
            {/if}
          </li>
        {/each}
      </ul>
    {:else if usage.totals}
      <dl class="mt-2 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
        <dt class="muted">Requests</dt>
        <dd class="tabular">{formatCompact(usage.totals.requests)}</dd>
        <dt class="muted">Input tokens</dt>
        <dd class="tabular">{formatCompact(usage.totals.inputTokens)}</dd>
        <dt class="muted">Output tokens</dt>
        <dd class="tabular">{formatCompact(usage.totals.outputTokens)}</dd>
      </dl>
      {#if usage.totals.since}<p class="muted mt-1 text-xs">Since {formatDate(usage.totals.since)}</p>{/if}
    {:else}
      <p class="muted mt-2 text-sm">The provider reported no usage limits.</p>
    {/if}
  {:else if error && !loading}
    <p class="mt-2 flex items-start gap-2 text-sm text-rose-600 dark:text-rose-400">
      <Icon name="alert" size={16} class="mt-0.5 shrink-0" />Usage unavailable: {error}
    </p>
  {:else}
    <SkeletonText lines={2} label="the provider usage" class="mt-2" />
  {/if}
</section>
