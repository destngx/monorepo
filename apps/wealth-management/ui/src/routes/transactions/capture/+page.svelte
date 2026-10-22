<script lang="ts">
  import { onDestroy } from 'svelte';
  import { invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { track, Button, Card, Deferred, Icon, LoadError, SkeletonCard } from '@libs/ui-kit';
  import { api } from '$lib/api/endpoints';
  import type { CaptureProgress, CaptureResult, ExchangeRate, TransactionInput } from '$lib/api/types';
  import CaptureInbox from '$lib/capture/CaptureInbox.svelte';
  import CaptureReview from '$lib/capture/CaptureReview.svelte';
  import { batchUsage, releaseItem, toPayload, type InboxItem } from '$lib/capture/materials';
  import { toReviewRows, writtenRows, type ReviewRow } from '$lib/capture/review';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import SourceBadge from '$lib/components/SourceBadge.svelte';
  import { formatVND } from '$lib/format';
  import { rateDetail } from '$lib/rates';
  import { toasts } from '$lib/state/toasts.svelte';
  import { STATUS } from '$lib/status';

  // Smart Capture (§6.2): materials → streamed analysis → review → commit.
  // The materials inbox renders at once; the AI status and the review options
  // stream in. Analyze waits for the AI status, the review for the options.
  let { data } = $props();

  // Written rows go back to the engine so later captures fill in the payee
  // used for each counterparty (WM-CAP-23). The rows are in the sheet either
  // way, so a failure here only warns.
  async function write(batchId: string, batchRows: ReviewRow[], inputs: TransactionInput[]) {
    const outcome = await api.createTransactions(batchId, inputs);
    const written = writtenRows(batchRows, inputs, outcome);
    if (written.length) {
      api
        .learnCapture(written)
        .catch((error) => toasts.error(error, 'Rows written, but their payees were not remembered'));
    }
    return outcome;
  }
  const ai = track(() => data.aiStatus);
  const options = track(() => data.options);
  // Null once loaded means the engine has no AI routes.
  const aiStatus = $derived(ai.latest ? ai.latest.value : undefined);

  const EXTRACT_PROMPT = 'tasks/capture-extract';
  const limits = $derived(
    aiStatus?.capture ?? { maxItems: 20, maxItemBytes: 5 * 1024 * 1024, maxBatchBytes: 20 * 1024 * 1024 },
  );
  const unavailable = $derived.by(() => {
    if (aiStatus === undefined) return null;
    if (!aiStatus) return 'AI is not available on the engine.';
    if (!aiStatus.configured) return 'AI is not configured: add the AI endpoint and model.';
    if (aiStatus.missingPrompts.includes(EXTRACT_PROMPT)) return `The Prompts sheet has no "${EXTRACT_PROMPT}" prompt.`;
    return null;
  });
  const analyzeHint = $derived.by(() => {
    if (ai.loading) return 'Checking that AI is available…';
    if (aiStatus === undefined) return 'The AI status did not load; retry it above.';
    if (unavailable) return unavailable;
    if (!items.length) return 'Add a screenshot or text first.';
    if (preparing) return 'Preparing images…';
    if (usage.problems.length) return usage.problems[0];
    return undefined;
  });

  let stage = $state<'inbox' | 'analyzing' | 'review'>('inbox');
  let items = $state<InboxItem[]>([]);
  let preparing = $state(0);
  let progress = $state<Record<string, CaptureProgress>>({});
  let step = $state('');
  let result = $state<CaptureResult | null>(null);
  let rows = $state<ReviewRow[]>([]);
  // The USD/VND quote for previews, or why it failed; null until a foreign row needs it.
  let usd = $state<{ quote: ExchangeRate } | { error: string } | null>(null);
  const usdQuote = $derived(usd && 'quote' in usd ? usd.quote : null);
  let controller: AbortController | null = null;

  const usage = $derived(batchUsage(items, limits));
  const canAnalyze = $derived(
    aiStatus !== undefined && !unavailable && items.length > 0 && usage.problems.length === 0 && preparing === 0,
  );
  const providerHost = $derived(aiStatus?.endpoint || 'the configured AI provider');

  const stepLabels: Record<CaptureProgress['stage'], string> = {
    extract: 'Reading materials',
    match: 'Matching accounts, transfers and duplicates',
    categorize: 'Categorising',
  };

  async function analyze() {
    controller?.abort();
    const current = new AbortController();
    controller = current;
    stage = 'analyzing';
    progress = {};
    step = stepLabels.extract;
    result = null;
    try {
      const materials = await Promise.all(items.map(toPayload));
      for await (const event of api.capture(materials, current.signal)) {
        if (event.type === 'progress') {
          step = stepLabels[event.stage];
          if (event.materialId) progress[event.materialId] = event;
        } else if (event.type === 'result') {
          result = event.result;
        } else {
          throw new Error(event.message);
        }
      }
      if (!result) throw new Error('The engine ended the analysis without a result.');
      rows = toReviewRows(result);
      stage = 'review';
    } catch (error) {
      stage = 'inbox';
      if (current.signal.aborted) toasts.push('info', 'Analysis cancelled.');
      else toasts.error(error, 'Smart Capture failed');
    } finally {
      if (controller === current) controller = null;
    }
  }

  function startOver() {
    controller?.abort();
    for (const item of items) releaseItem(item);
    items = [];
    rows = [];
    result = null;
    progress = {};
    stage = 'inbox';
  }

  // Preview Binance (USD/USDT) rows in VND; the sheet formula does the real conversion.
  $effect(() => {
    if (usd !== null || !rows.length || !options.value) return;
    const foreign = new Set(options.value.accounts.filter((a) => a.currency !== 'VND').map((a) => a.name));
    if (!rows.some((row) => foreign.has(row.tx.accountName))) return;
    api
      .exchangeRate('USD', 'VND')
      .then((quote) => (usd = quote.rate > 0 ? { quote } : { error: 'the engine returned no rate' }))
      .catch((cause) => (usd = { error: cause instanceof Error ? cause.message : String(cause) }));
  });

  // Materials live only in this tab: free the image previews on leave (WM-CAP-10).
  onDestroy(() => {
    controller?.abort();
    for (const item of items) releaseItem(item);
  });

  function materialStatus(id: string): { label: string; tone: string } {
    const entry = progress[id];
    if (!entry) return { label: 'Queued', tone: 'muted' };
    switch (entry.status) {
      case 'running':
        return { label: 'Reading…', tone: STATUS.info.ink };
      case 'ok':
        return { label: `${entry.events ?? 0} found`, tone: STATUS.good.ink };
      case 'unreadable':
        return { label: `Unreadable: ${entry.message ?? ''}`, tone: STATUS.warning.ink };
      default:
        return { label: `Failed: ${entry.message ?? ''}`, tone: STATUS.critical.ink };
    }
  }
</script>

<PageHeader title="Smart Capture" subtitle="Turn bank and wallet screenshots or texts into reviewed transactions">
  {#snippet actions()}
    <a class="btn-ghost" href="{base}/transactions"><Icon name="chevronLeft" size={16} /> Transactions</a>
    {#if stage === 'review'}
      <Button onclick={() => (stage = 'inbox')}>Back to materials</Button>
      <Button icon="refresh" onclick={startOver}>Start over</Button>
    {/if}
  {/snippet}
</PageHeader>

<p
  class="mb-4 flex items-start gap-2 rounded-xl bg-sky-500/10 px-4 py-3 text-sm text-sky-800 dark:text-sky-200"
  role="note"
>
  <Icon name="shield" size={18} class="mt-0.5" />
  <span>
    Materials are sent to <strong>{providerHost}</strong> to be read. They stay in this browser tab only: never stored, logged
    or written to the sheet, and discarded when you start over or leave. Account numbers, phone numbers and configured names
    are masked in text; images are sent as they are, so crop anything you do not want to share.
  </span>
</p>

{#if unavailable}
  <Card class="mb-4">
    <p class="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-300">
      <Icon name="alert" size={16} />
      {unavailable} See <a class="link underline" href="{base}/settings">Settings</a>.
    </p>
  </Card>
{:else if ai.failure && !ai.latest}
  <LoadError failure={ai.failure} label="the AI status" retry={() => invalidateAll()} compact class="mb-4" />
{/if}

{#if stage === 'review' && result}
  {#if usd}
    <p class="muted mb-3 flex flex-wrap items-center gap-2 text-xs">
      {#if 'quote' in usd}
        USD and USDT rows are previewed at {formatVND(usd.quote.rate)} per USD; the sheet does the real conversion.
        <SourceBadge source={usd.quote.source ?? { provider: usd.quote.provider }} detail={rateDetail(usd.quote)} />
      {:else}
        USD and USDT rows have no VND preview: {usd.error}
      {/if}
    </p>
  {/if}
  <!-- The analysis needs no options; editing and committing its rows does. -->
  <Deferred data={data.options} label="accounts, categories and tags">
    {#snippet children(value)}
      <div>
        {#if result}
          <CaptureReview
            bind:rows
            {result}
            {items}
            accounts={value.accounts}
            categories={value.categories}
            tags={value.tags}
            cardTags={value.cardTags}
            cardNames={value.cardNames}
            creditLines={value.creditLines}
            usdRate={usdQuote?.rate ?? null}
            {write}
          />
        {/if}
      </div>
    {/snippet}
    {#snippet skeleton()}
      <SkeletonCard title="Suggested transactions" lines={1} rows={4} label="accounts, categories and tags" />
    {/snippet}
  </Deferred>
{:else}
  <Card title="Materials" subtitle="Add everything about the same period at once: matching messages are merged.">
    <CaptureInbox bind:items bind:preparing {limits} disabled={stage === 'analyzing'} />

    {#if stage === 'analyzing'}
      <div class="mt-4 rounded-xl bg-stone-900/5 p-4 dark:bg-white/5" aria-live="polite">
        <p class="flex items-center gap-2 text-sm font-medium">
          <Icon name="sparkles" size={16} class="animate-pulse" />
          {step}…
        </p>
        <ul class="mt-2 space-y-0.5 text-xs">
          {#each items as item (item.id)}
            {@const status = materialStatus(item.id)}
            <li>
              <span class="font-medium">{item.id}</span>
              {item.name}: <span class={status.tone}>{status.label}</span>
            </li>
          {/each}
        </ul>
      </div>
    {/if}

    <div class="mt-4 flex justify-end gap-2">
      {#if stage === 'analyzing'}
        <Button icon="stop" onclick={() => controller?.abort()}>Cancel</Button>
      {:else}
        <Button
          variant="primary"
          icon="sparkles"
          busy={ai.loading}
          disabled={!canAnalyze}
          title={analyzeHint}
          onclick={analyze}>Analyze {items.length || ''}</Button
        >
      {/if}
    </div>
  </Card>
{/if}
