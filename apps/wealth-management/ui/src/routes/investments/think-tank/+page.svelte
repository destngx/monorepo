<script lang="ts">
  import { base } from '$app/paths';
  import Markdown from '$lib/ai/Markdown.svelte';
  import type { ChatPageContext } from '$lib/api/types';
  import ChatPanel from '$lib/chat/ChatPanel.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import CommandCards from '$lib/investments/thinktank/CommandCards.svelte';
  import PhaseStepper from '$lib/investments/thinktank/PhaseStepper.svelte';
  import RawContext from '$lib/investments/thinktank/RawContext.svelte';
  import { biasTone } from '$lib/investments/news';
  import { newsStore } from '$lib/investments/newsStore.svelte';
  import { chatSummary, focusLabel, parseSymbols } from '$lib/investments/thinktank';
  import { recordNote } from '$lib/recommendations/recommendations';
  import { thinkTank } from '$lib/investments/thinktank.svelte';

  // Think Tank (WM-AI-9, WM-INV-18): the engine gathers the portfolio, market
  // pulse and web searches, then streams a debate, a synthesis and commands.
  // Without symbols it analyses the user's holdings; what is worth doing goes
  // to the recommendations list (ROADMAP M5, WM-INV-29, WM-REC-8).
  let { data } = $props();

  const REQUIRED_PROMPTS = ['market/think-tank', 'market/synthesis', 'market/action'];
  const unavailable = $derived.by(() => {
    if (!data.aiStatus) return 'AI is not available on the engine.';
    if (!data.aiStatus.configured) return 'AI is not configured: add the AI endpoint and model.';
    const missing = REQUIRED_PROMPTS.filter((key) => data.aiStatus?.missingPrompts.includes(key));
    if (missing.length) return `The Prompts sheet is missing: ${missing.join(', ')}.`;
    return null;
  });

  let includeNews = $state(true);
  let symbolsText = $state('');
  const symbols = $derived(parseSymbols(symbolsText));
  const run = $derived(thinkTank.state);
  const started = $derived(run.status !== 'idle');
  const finished = $derived(run.status === 'done');

  function start() {
    void thinkTank.start(includeNews ? newsStore.reports : [], symbols);
  }

  const chatPage = $derived<ChatPageContext>({
    path: '/investments/think-tank',
    title: 'Think Tank',
    summary: chatSummary(run),
  });
</script>

<PageHeader title="Think Tank" subtitle="An AI debate over your holdings, the market pulse and the news">
  {#snippet actions()}
    <a class="btn-ghost" href="{base}/investments/news"><Icon name="newspaper" size={16} /> News Scanner</a>
  {/snippet}
</PageHeader>

{#if unavailable}
  <Card class="mb-4">
    <p class="flex items-center gap-2 text-sm text-amber-700 dark:text-amber-300">
      <Icon name="alert" size={16} />
      {unavailable} See <a class="underline" href="{base}/settings">Settings</a>.
    </p>
  </Card>
{/if}

<Card
  title="Analysis"
  subtitle="Figures are computed by the app; the models only interpret them. Commands are suggestions, nothing is executed."
>
  <div class="mb-4">
    <label for="think-tank-symbols" class="text-sm font-medium">Symbols <span class="muted">(optional)</span></label>
    <input
      id="think-tank-symbols"
      class="field mt-1 w-full max-w-md"
      placeholder="e.g. FPT, BTC"
      autocomplete="off"
      aria-describedby="think-tank-symbols-hint"
      bind:value={symbolsText}
      disabled={thinkTank.running}
    />
    <p id="think-tank-symbols-hint" class="muted mt-1 text-xs">
      {symbols.length
        ? `Analyses ${symbols.join(', ')}.`
        : 'Leave empty to analyse your current holdings (symbols only are sent, never account names).'}
    </p>
  </div>
  <div class="flex flex-wrap items-center justify-between gap-3">
    {#if newsStore.reports.length}
      <label class="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          class="size-4 accent-indigo-600"
          bind:checked={includeNews}
          disabled={thinkTank.running}
        />
        Include {newsStore.reports.length} scanned news {newsStore.reports.length === 1 ? 'report' : 'reports'}
        <span class="flex flex-wrap gap-1">
          {#each newsStore.reports as report (report.topic)}
            <Badge tone={biasTone[report.bias]}>{report.topic}</Badge>
          {/each}
        </span>
      </label>
    {:else}
      <p class="muted text-sm">
        Tip: <a class="underline" href="{base}/investments/news">scan news topics</a> first to include their sentiment.
      </p>
    {/if}
    <div class="flex gap-2">
      {#if thinkTank.running}
        <button type="button" class="btn-ghost" onclick={() => thinkTank.stop()}>
          <Icon name="stop" size={16} /> Stop
        </button>
      {:else}
        <button type="button" class="btn-primary" disabled={!!unavailable} onclick={start}>
          <Icon name={started ? 'refresh' : 'sparkles'} size={16} />
          {started ? 'Run again' : 'Run analysis'}
        </button>
      {/if}
    </div>
  </div>

  {#if started}
    <div class="mt-4" aria-live="polite">
      <PhaseStepper phases={run.phases} />
    </div>
  {/if}
  {#if run.context?.focus}
    <p class="mt-3 text-sm" role="status"><span class="font-medium">Focus:</span> {focusLabel(run.context.focus)}</p>
  {/if}

  {#if run.status === 'error'}
    <div class="mt-4 flex flex-wrap items-start justify-between gap-2 rounded-xl bg-rose-500/10 px-4 py-3" role="alert">
      <p class="flex items-start gap-2 text-sm text-rose-700 dark:text-rose-300">
        <Icon name="alert" size={16} class="mt-0.5" />
        {run.error}
      </p>
      <button type="button" class="btn-ghost text-sm" disabled={!!unavailable} onclick={start}>
        <Icon name="refresh" size={16} /> Retry
      </button>
    </div>
  {:else if run.status === 'stopped'}
    <p class="muted mt-4 text-sm" role="status">Stopped. Finished phases are kept below.</p>
  {/if}
</Card>

{#if !started}
  <Card class="mt-4">
    <EmptyState
      title="No analysis yet"
      detail="Run the analysis to get a think tank debate, a synthesis and concrete commands. It can take a few minutes."
    />
  </Card>
{/if}

{#if run.thinkTank}
  <Card
    title={run.thinkTank.name}
    subtitle={run.thinkTank.model ? `Model: ${run.thinkTank.model}` : undefined}
    class="mt-4"
  >
    <Markdown text={run.thinkTank.content} />
  </Card>
{:else if run.phases['think-tank'].status === 'running'}
  <Card title="Think tank" subtitle="The experts are debating…" class="mt-4">
    <div class="space-y-2" role="status" aria-label="Generating">
      <div class="h-3 w-3/4 animate-pulse rounded bg-slate-900/10 dark:bg-white/10"></div>
      <div class="h-3 w-full animate-pulse rounded bg-slate-900/10 dark:bg-white/10"></div>
      <div class="h-3 w-2/3 animate-pulse rounded bg-slate-900/10 dark:bg-white/10"></div>
    </div>
  </Card>
{/if}

{#if run.synthesis}
  <Card
    title={run.synthesis.name}
    subtitle={run.synthesis.model ? `Model: ${run.synthesis.model}` : undefined}
    class="mt-4"
  >
    <Markdown text={run.synthesis.content} />
  </Card>
{:else if run.phases.synthesis.status === 'running'}
  <Card title="Synthesis" subtitle="Weighing the debate…" class="mt-4">
    <div
      class="h-3 w-2/3 animate-pulse rounded bg-slate-900/10 dark:bg-white/10"
      role="status"
      aria-label="Generating"
    ></div>
  </Card>
{/if}

{#if run.actions}
  <Card title={run.actions.name} subtitle={run.actions.model ? `Model: ${run.actions.model}` : undefined} class="mt-4">
    {#if run.actions.commands.length}
      <CommandCards commands={run.actions.commands} />
    {:else}
      <p class="muted text-sm">The models proposed no commands.</p>
    {/if}
    {#if run.actions.recorded}
      <p class="mt-3 text-sm {run.actions.recorded.error ? 'text-rose-700 dark:text-rose-300' : 'muted'}" role="status">
        {recordNote(run.actions.recorded)}
        <a class="underline" href="{base}/recommendations?page={encodeURIComponent('/investments/think-tank')}"
          >See the recommendations</a
        >
      </p>
    {/if}
  </Card>
{/if}

{#if run.context}
  <details class="glass group mt-4 p-5">
    <summary
      class="flex cursor-pointer items-center gap-2 text-sm font-semibold text-slate-700 select-none dark:text-slate-200"
    >
      <Icon name="chevronRight" size={16} class="transition group-open:rotate-90" />
      Raw context
      <span class="muted text-xs font-normal">What the models saw</span>
    </summary>
    <div class="mt-4"><RawContext context={run.context} /></div>
  </details>
{/if}

{#if finished}
  <Card
    title="Follow-up"
    subtitle="Ask about this analysis; the synthesis and commands are shared as page context."
    class="mt-4"
  >
    <ChatPanel page={chatPage} class="h-[28rem]" />
  </Card>
{/if}
