<script lang="ts">
  import { onDestroy } from 'svelte';
  import { base } from '$app/paths';
  import { invalidateAll } from '$app/navigation';
  import { track, Button, Card, EmptyState, Icon } from '@libs/ui-kit';
  import { api } from '$lib/api/endpoints';
  import { ApiError } from '$lib/api/client';
  import { DEFAULT_NEWS_TOPICS, type NewsReport } from '$lib/api/types';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import AIRequirement from '$lib/investments/ai/AIRequirement.svelte';
  import { aiUnavailable } from '$lib/ai/aiStatus';
  import NewsReportCard from '$lib/investments/news/NewsReportCard.svelte';
  import { normalizeTopic, topicKey } from '$lib/investments/news';
  import { newsStore } from '$lib/investments/newsStore.svelte';
  import { toasts } from '$lib/state/toasts.svelte';

  // News Scanner (WM-INV-17): search a topic, classify each article, derive
  // the bias. Reports stay in this tab so the Think Tank can include them.
  let { data } = $props();

  const NEWS_PROMPTS = ['market/news-sentiment'];
  // The AI status streams in; until it answers the scan controls stay usable.
  const aiStatus = track(() => data.aiStatus);
  const unavailable = $derived(aiUnavailable(aiStatus.value, NEWS_PROMPTS));

  interface Scan {
    topic: string;
    loading: boolean;
    error: string;
  }

  // Per-topic request state; the reports themselves live in the store.
  let scans = $state<Record<string, Scan>>({});
  const controllers = new Map<string, AbortController>();
  let custom = $state('');

  const anyLoading = $derived(Object.values(scans).some((s) => s.loading));

  // Scanned reports first, then topics still loading or failed without a report.
  const cards = $derived.by(() => {
    const keys = new Set(newsStore.reports.map((r) => topicKey(r.topic)));
    const pending = Object.entries(scans)
      .filter(([key]) => !keys.has(key))
      .map(([key, scan]) => ({ key, topic: scan.topic, report: undefined as NewsReport | undefined }));
    return [...newsStore.reports.map((r) => ({ key: topicKey(r.topic), topic: r.topic, report: r })), ...pending];
  });

  async function scan(rawTopic: string, force = false) {
    const topic = normalizeTopic(rawTopic);
    if (!topic) return;
    const key = topicKey(topic);
    controllers.get(key)?.abort();
    const controller = new AbortController();
    controllers.set(key, controller);
    scans[key] = { topic, loading: true, error: '' };
    try {
      const report = await api.newsSentiment(topic, { force, signal: controller.signal });
      if (controller.signal.aborted) return;
      newsStore.upsert(report);
      delete scans[key];
    } catch (error) {
      if (controller.signal.aborted) return;
      if (error instanceof ApiError && error.authProblem) toasts.error(error);
      scans[key] = { topic, loading: false, error: error instanceof Error ? error.message : String(error) };
    } finally {
      if (controllers.get(key) === controller) controllers.delete(key);
    }
  }

  // Defaults run in parallel; each topic keeps its own error.
  const scanAll = () => Promise.all(DEFAULT_NEWS_TOPICS.map((topic) => scan(topic)));

  function scanCustom(event: SubmitEvent) {
    event.preventDefault();
    if (!normalizeTopic(custom)) return;
    void scan(custom);
    custom = '';
  }

  function remove(key: string, topic: string) {
    controllers.get(key)?.abort();
    delete scans[key];
    newsStore.remove(topic);
  }

  const isLoading = (topic: string) => scans[topicKey(topic)]?.loading ?? false;

  onDestroy(() => {
    for (const controller of controllers.values()) controller.abort();
  });
</script>

<PageHeader title="News Scanner" subtitle="Sentiment of the latest news per topic, classified article by article">
  {#snippet actions()}
    {#if newsStore.reports.length}
      <a class="btn-ghost" href="{base}/investments/think-tank"><Icon name="sparkles" size={16} /> Use in Think Tank</a>
      <Button icon="trash" disabled={anyLoading} onclick={() => newsStore.clear()}>Clear</Button>
    {/if}
  {/snippet}
</PageHeader>

<AIRequirement status={aiStatus} {unavailable} retry={() => invalidateAll()} />

<Card title="Topics" subtitle="Results are cached for 30 minutes; Regenerate searches again.">
  <div class="flex flex-wrap items-center gap-2">
    {#each DEFAULT_NEWS_TOPICS as topic (topic)}
      <Button
        icon="newspaper"
        class="border border-stone-900/10 dark:border-white/10"
        busy={isLoading(topic)}
        disabled={!!unavailable}
        title={isLoading(topic) ? `Scanning ${topic}…` : `Scan ${topic}`}
        onclick={() => void scan(topic)}>{topic}</Button
      >
    {/each}
    <Button
      variant="primary"
      icon="search"
      busy={anyLoading}
      busyText="Scanning…"
      disabled={!!unavailable}
      onclick={() => void scanAll()}>Scan all defaults</Button
    >
  </div>
  {#if anyLoading}
    <p class="muted mt-3 text-xs" role="status">
      Searching the web and classifying each article; a topic can take a minute.
    </p>
  {/if}

  <form class="mt-4 flex flex-col gap-2 sm:flex-row" onsubmit={scanCustom}>
    <label class="sr-only" for="news-topic">Custom topic</label>
    <input
      id="news-topic"
      class="field flex-1"
      placeholder="Custom topic, e.g. Gold prices or VN-Index banks"
      maxlength={200}
      bind:value={custom}
      disabled={!!unavailable}
    />
    <Button type="submit" variant="primary" icon="search" disabled={!!unavailable || !normalizeTopic(custom)}
      >Scan</Button
    >
  </form>
</Card>

{#if cards.length}
  <div class="mt-4 grid grid-cols-1 gap-4 xl:grid-cols-2">
    {#each cards as card (card.key)}
      <NewsReportCard
        topic={card.topic}
        report={card.report}
        loading={scans[card.key]?.loading ?? false}
        error={scans[card.key]?.error ?? ''}
        onregenerate={() => scan(card.topic, !!card.report)}
        onremove={() => remove(card.key, card.topic)}
      />
    {/each}
  </div>
{:else}
  <Card class="mt-4">
    <EmptyState
      title="No topics scanned yet"
      detail="Scan a default topic or your own. Scanned reports stay in this tab and can be sent to the Think Tank."
    />
  </Card>
{/if}
