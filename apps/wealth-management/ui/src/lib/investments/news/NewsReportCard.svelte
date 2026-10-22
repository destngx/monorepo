<script lang="ts">
  import type { NewsReport } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import { Button, Icon, SkeletonText, Spinner } from '@libs/ui-kit';
  import { biasTone, countShares, formatTimestamp, sentimentTone, sourceHost } from '../news';
  import ScoreBar from './ScoreBar.svelte';

  interface Props {
    topic: string;
    report?: NewsReport;
    loading?: boolean;
    error?: string;
    onregenerate: () => void;
    onremove?: () => void;
  }

  // One scanned topic (WM-INV-17): the bias comes from the engine's strict
  // plurality of per-article verdicts, never from the model directly.
  let { topic, report, loading = false, error = '', onregenerate, onremove }: Props = $props();

  const shares = $derived(report ? countShares(report.counts) : null);
</script>

<section class="panel p-5" aria-busy={loading}>
  <header class="mb-3 flex flex-wrap items-start justify-between gap-3">
    <div class="min-w-0">
      <h2 class="flex flex-wrap items-center gap-2 text-sm font-semibold text-stone-700 dark:text-stone-200">
        {report?.topic ?? topic}
        {#if report}<Badge tone={biasTone[report.bias]}>{report.bias}</Badge>{/if}
      </h2>
      {#if report}
        <p class="muted mt-0.5 flex flex-wrap items-center gap-x-2 text-xs">
          <span>Search: {report.provider || 'none'}</span>
          {#if report.fallback}
            <span class="text-amber-700 dark:text-amber-300" title={report.fallback}>
              <Icon name="alert" size={12} class="inline" /> fallback: {report.fallback}
            </span>
          {/if}
          <span>· {formatTimestamp(report.generatedAt)}</span>
        </p>
      {/if}
    </div>
    <div class="flex items-center gap-1">
      <Button
        size="sm"
        icon="refresh"
        busy={loading}
        busyText={report ? 'Regenerating…' : 'Scanning…'}
        title={report ? 'Search again and reclassify' : `Scan ${topic} again`}
        onclick={onregenerate}>{report ? 'Regenerate' : 'Retry'}</Button
      >
      {#if onremove && !loading}
        <Button size="icon" icon="x" label="Remove {topic}" onclick={onremove} />
      {/if}
    </div>
  </header>

  {#if error}
    <p
      class="mb-3 flex items-start gap-2 rounded-xl bg-rose-500/10 px-3 py-2 text-sm text-rose-700 dark:text-rose-300"
      role="alert"
    >
      <Icon name="alert" size={16} class="mt-0.5 shrink-0" />
      <span class="min-w-0 break-words">Could not scan {topic}: {error}</span>
    </p>
  {/if}
  {#if loading && report}
    <p class="muted mb-3 flex items-center gap-2 text-xs" role="status">
      <Spinner size={12} /> Searching again; the last report stays until the new one arrives.
    </p>
  {/if}

  {#if report && shares}
    {@const total = report.counts.positive + report.counts.negative + report.counts.neutral}
    <div class="mb-4">
      <div
        class="flex h-2 w-full overflow-hidden rounded-full bg-stone-900/10 dark:bg-white/10"
        role="img"
        aria-label="{report.counts.positive} positive, {report.counts.negative} negative, {report.counts
          .neutral} neutral"
      >
        <div class="bg-emerald-500" style:width="{shares.positive}%"></div>
        <div class="bg-rose-500" style:width="{shares.negative}%"></div>
        <div class="bg-stone-400" style:width="{shares.neutral}%"></div>
      </div>
      <p class="mt-1.5 flex flex-wrap gap-x-4 text-xs">
        <span class="text-emerald-700 dark:text-emerald-300">{report.counts.positive} positive</span>
        <span class="text-rose-700 dark:text-rose-300">{report.counts.negative} negative</span>
        <span class="muted">{report.counts.neutral} neutral</span>
        <span class="muted">· {total} articles</span>
      </p>
    </div>

    {#if report.articles.length}
      <ul class="divide-y divide-stone-900/5 dark:divide-white/5">
        {#each report.articles as article, i (i)}
          <li class="py-3 first:pt-0 last:pb-0">
            <div class="flex flex-wrap items-start justify-between gap-2">
              <a
                class="link min-w-0 flex-1 text-sm font-medium"
                href={article.url}
                target="_blank"
                rel="noopener noreferrer">{article.title || article.url}</a
              >
              <Badge tone={sentimentTone[article.sentiment] ?? 'neutral'}>{article.sentiment}</Badge>
            </div>
            <p class="muted mt-0.5 text-xs">{sourceHost(article)}</p>
            {#if article.summary}<p class="mt-1 text-sm">{article.summary}</p>{/if}
            <div class="mt-2 flex flex-wrap gap-x-5 gap-y-1">
              <ScoreBar label="Confidence" value={article.confidence} />
              <ScoreBar label="Impact" value={article.impact} />
            </div>
            {#if article.snippet}
              <details class="mt-1.5 text-xs">
                <summary class="muted cursor-pointer select-none">Snippet</summary>
                <p class="mt-1 whitespace-pre-line text-stone-600 dark:text-stone-300">{article.snippet}</p>
              </details>
            {/if}
          </li>
        {/each}
      </ul>
    {:else}
      <p class="muted text-sm">The search found no articles for this topic. Try another wording or regenerate later.</p>
    {/if}
  {:else if loading}
    <p class="muted mb-3 flex items-center gap-2 text-xs">
      <Spinner size={12} /> Searching the web and classifying each article…
    </p>
    <SkeletonText lines={4} label="the {topic} news scan" />
  {/if}
</section>
