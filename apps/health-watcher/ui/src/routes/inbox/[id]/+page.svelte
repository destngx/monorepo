<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import type { Finding, FindingInput, Observation, ObservationInput, Page, Quad, RecordStatus } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import FileItemModal from '$lib/components/FileItemModal.svelte';
  import FindingList from '$lib/components/FindingList.svelte';
  import FindingModal from '$lib/components/FindingModal.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import ObservationModal from '$lib/components/ObservationModal.svelte';
  import ObservationTable from '$lib/components/ObservationTable.svelte';
  import PageImage from '$lib/components/PageImage.svelte';
  import PageTitle from '$lib/components/PageTitle.svelte';
  import ReadingBadge from '$lib/components/ReadingBadge.svelte';
  import { documentKindLabels, formatDate } from '$lib/format';
  import { findingInput, observationInput, readingState } from '$lib/reading';
  import { session } from '$lib/state/session.svelte';

  // Reviews one inbox item before filing (HW-READ-4, HW-AI-7): the pages
  // and their masked copies, the reading's status and its drafts.
  let { data } = $props();

  const item = $derived(data.detail.item);
  const reading = $derived(item.reading);
  const status = $derived(readingState(item));
  const pages = $derived(data.detail.pages);
  const pageIndex = $derived(new Map(pages.map((p, i) => [p.id, i + 1])));
  const pageNumber = (id: string | undefined) => (id ? pageIndex.get(id) : undefined);
  const editable = $derived(session.canEdit && item.status === 'new');
  const ocrFailed = $derived((item.ocr?.failed ?? 0) > 0);
  const needsCheck = $derived(reading?.status === 'needs_check');
  const canRetry = $derived(
    ocrFailed || (data.detail.aiEnabled && ['failed', 'done', 'skipped', 'queued'].includes(reading?.status ?? '')),
  );
  const canSkip = $derived(
    data.detail.aiEnabled && reading !== undefined && !['skipped', 'off', 'waiting'].includes(reading.status),
  );
  const missingImage = $derived(pages.some((p) => !p.hasImage));

  let filing = $state(false);
  let view = $state<Record<string, 'masked' | 'original'>>({});
  let drawing = $state<string | null>(null);
  let editingObs = $state<Observation | null>(null);
  let editingFinding = $state<Finding | null>(null);

  // While the Mac or the AI is reading, refresh every few seconds.
  $effect(() => {
    if (status.tone !== 'busy' && !(status.tone === 'warn' && !needsCheck)) return;
    const timer = setInterval(() => void invalidateAll(), 4000);
    return () => clearInterval(timer);
  });

  function viewOf(p: Page): 'masked' | 'original' {
    return view[p.id] ?? (p.mask ? 'masked' : 'original');
  }

  async function act(fn: () => Promise<void>) {
    await fn();
    await invalidateAll();
  }

  async function setBoxes(p: Page, boxes: Quad[]) {
    await session.api.setPageBoxes(item.id, p.id, boxes);
    await invalidateAll();
  }

  async function setObsStatus(o: Observation, status: RecordStatus) {
    await session.api.updateObservation(o.id, { ...observationInput(o), status });
    await invalidateAll();
  }

  async function setFindingStatus(f: Finding, status: RecordStatus) {
    await session.api.updateFinding(f.id, { ...findingInput(f), status });
    await invalidateAll();
  }

  async function saveObs(input: ObservationInput) {
    if (!editingObs) return;
    await session.api.updateObservation(editingObs.id, input);
    await invalidateAll();
  }

  async function saveFinding(input: FindingInput) {
    if (!editingFinding) return;
    await session.api.updateFinding(editingFinding.id, input);
    await invalidateAll();
  }

  async function discard() {
    if (!confirm('Discard this item? Its files are deleted unless a record already uses them.')) return;
    await session.api.discard(item.id);
    await goto(`${base}/inbox`);
  }
</script>

<a class="link mb-3 inline-flex items-center gap-1 text-sm" href="{base}/inbox">
  <Icon name="chevronLeft" size={16} />Inbox
</a>

<PageTitle
  title={reading?.proposal?.title ||
    (item.files.length === 1 ? item.files[0].originalName || 'Unfiled item' : `${item.files.length} files`)}
  subtitle="Added {formatDate(item.createdAt)}{reading?.proposal?.kind
    ? ` · looks like a ${documentKindLabels[reading.proposal.kind].toLowerCase()}`
    : ''}"
>
  {#snippet actions()}
    {#if item.status !== 'new'}
      {#if item.documentId}<a class="btn-primary" href="{base}/documents/{item.documentId}">Open the document</a>{/if}
    {:else if session.canEdit}
      <Button variant="primary" icon="check" onclick={() => (filing = true)}>File</Button>
      <Button icon="trash" errorContext="Could not discard" onclick={discard}>Discard</Button>
    {/if}
  {/snippet}
</PageTitle>

<Card title="Reading" class="mb-5">
  <div class="flex flex-col gap-3 sm:flex-row sm:items-start">
    <div class="min-w-0 flex-1"><ReadingBadge {status} showDetail /></div>
    {#if editable}
      <div class="flex flex-wrap gap-2">
        {#if needsCheck && data.detail.aiEnabled}
          <Button
            variant="primary"
            icon="send"
            disabled={missingImage}
            title={missingImage ? 'Some pages have no image, so they cannot be sent.' : undefined}
            success="Sent to the AI"
            errorContext="Could not send"
            onclick={() => act(() => session.api.sendReading(item.id))}>Pages look safe, send</Button
          >
        {/if}
        {#if canRetry}
          <Button
            icon="refresh"
            errorContext="Could not retry"
            onclick={() => act(() => session.api.retryReading(item.id))}
            >{reading?.status === 'done' ? 'Read again' : 'Retry'}</Button
          >
        {/if}
        {#if canSkip}
          <Button errorContext="Could not skip" onclick={() => act(() => session.api.skipReading(item.id))}
            >Don't read with AI</Button
          >
        {/if}
      </div>
    {/if}
  </div>
  {#if needsCheck}
    <p class="muted mt-3 text-sm">
      Look at each page as the AI would see it. Black boxes hide names, ids, phone numbers and barcodes. Drag on a page
      to hide anything else, then send.
    </p>
  {:else if reading?.status === 'done' && reading.model}
    <p class="muted mt-3 text-xs">
      Read by {reading.model} from {reading.pagesSent}
      {reading.pagesSent === 1 ? 'page' : 'pages'}, with personal details blacked out. Check the values before filing.
    </p>
  {/if}
</Card>

{#if data.detail.observations.length > 0}
  <Card title="Values" subtitle="Filing approves these. Reject the ones that are wrong." class="mb-5">
    <ObservationTable
      observations={data.detail.observations}
      {editable}
      {pageNumber}
      onedit={(o) => (editingObs = o)}
      onstatus={setObsStatus}
    />
  </Card>
{/if}
{#if data.detail.findings.length > 0}
  <Card title="Findings" class="mb-5">
    <FindingList
      findings={data.detail.findings}
      {editable}
      {pageNumber}
      onedit={(f) => (editingFinding = f)}
      onstatus={setFindingStatus}
    />
  </Card>
{/if}

<h2 class="mb-3 text-sm font-semibold tracking-wide text-slate-700 dark:text-slate-200">Pages</h2>
<div class="space-y-5">
  {#each pages as p, i (p.id)}
    <section class="glass p-4">
      <header class="mb-3 flex flex-wrap items-center gap-2">
        <span class="font-medium">Page {i + 1}</span>
        {#if p.mask?.status === 'needs_check'}
          <span class="rounded-full bg-amber-500/15 px-2 py-0.5 text-xs text-amber-800 dark:text-amber-200"
            >Needs a check</span
          >
        {:else if p.mask?.status === 'checked'}
          <span class="rounded-full bg-teal-600/10 px-2 py-0.5 text-xs text-teal-700 dark:text-teal-300">Checked</span>
        {/if}
        {#if p.mask}
          <span class="muted text-xs">{p.mask.maskedLines} lines hidden</span>
        {/if}
        <span class="ml-auto flex gap-1">
          {#if p.mask}
            <button
              type="button"
              class="btn-ghost px-2.5 py-1 text-xs {viewOf(p) === 'masked'
                ? 'bg-teal-600/10 text-teal-700 dark:text-teal-300'
                : ''}"
              onclick={() => (view[p.id] = 'masked')}>What the AI sees</button
            >
            <button
              type="button"
              class="btn-ghost px-2.5 py-1 text-xs {viewOf(p) === 'original'
                ? 'bg-teal-600/10 text-teal-700 dark:text-teal-300'
                : ''}"
              onclick={() => {
                view[p.id] = 'original';
                if (drawing === p.id) drawing = null;
              }}>Original</button
            >
          {/if}
          {#if editable && p.mask && p.hasImage && data.detail.aiEnabled}
            <Button
              size="sm"
              icon="pencil"
              onclick={() => {
                drawing = drawing === p.id ? null : p.id;
                view[p.id] = 'masked';
              }}>{drawing === p.id ? 'Done' : 'Hide more'}</Button
            >
          {/if}
        </span>
      </header>
      {#if p.mask?.reasons?.length}
        <ul class="mb-3 list-disc pl-5 text-xs text-amber-800 dark:text-amber-200">
          {#each p.mask.reasons as r (r)}<li>{r}</li>{/each}
        </ul>
      {/if}
      {#if drawing === p.id}
        <p class="muted mb-2 text-xs">Drag to add a black box. Click a red box to remove it.</p>
      {/if}
      <PageImage
        page={p}
        masked={viewOf(p) === 'masked'}
        drawing={drawing === p.id}
        boxes={p.mask?.manual ?? []}
        onboxes={(boxes) => void setBoxes(p, boxes)}
      />
      {#if p.text}
        {@const masked = viewOf(p) === 'masked' && p.mask}
        <details class="mt-3">
          <summary class="muted cursor-pointer text-xs"
            >{masked ? 'Text the AI sees (personal lines replaced with [ẩn])' : 'Text read on the Mac'}</summary
          >
          <pre
            class="mt-2 max-h-80 overflow-auto rounded-xl bg-slate-900/5 p-3 text-xs whitespace-pre-wrap dark:bg-white/5">{masked
              ? p.mask?.text
              : p.text}</pre>
        </details>
      {/if}
    </section>
  {:else}
    <section class="glass">
      <EmptyState title="No pages yet" detail="The Mac is still reading the files. This page refreshes on its own." />
    </section>
  {/each}
</div>

<FileItemModal
  item={filing ? item : null}
  people={data.people}
  providers={data.providers}
  onclose={() => (filing = false)}
  onfiled={() => {
    filing = false;
    void goto(`${base}/inbox`);
  }}
/>
<ObservationModal
  value={editingObs ? observationInput(editingObs) : null}
  title="Edit value"
  {pages}
  onsave={saveObs}
  onclose={() => (editingObs = null)}
/>
<FindingModal
  value={editingFinding ? findingInput(editingFinding) : null}
  title="Edit finding"
  {pages}
  onsave={saveFinding}
  onclose={() => (editingFinding = null)}
/>
