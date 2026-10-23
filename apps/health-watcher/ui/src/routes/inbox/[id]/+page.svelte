<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import type { Finding, FindingInput, Observation, ObservationInput, Page, Quad, RecordStatus } from '$lib/api/types';
  import {
    Button,
    Card,
    EmptyState,
    FoldToggle,
    Icon,
    MenuButton,
    type MenuItem,
    PageTitle,
    action,
    toasts,
  } from '@libs/ui-kit';
  import FileItemModal from '$lib/components/FileItemModal.svelte';
  import { Folds } from '$lib/folds.svelte';
  import FindingList from '$lib/components/FindingList.svelte';
  import FindingModal from '$lib/components/FindingModal.svelte';
  import ItemCamera from '$lib/components/ItemCamera.svelte';
  import ObservationModal from '$lib/components/ObservationModal.svelte';
  import ObservationTable from '$lib/components/ObservationTable.svelte';
  import PageImage from '$lib/components/PageImage.svelte';
  import PhoneCaptureModal from '$lib/components/PhoneCaptureModal.svelte';
  import ReadingBadge from '$lib/components/ReadingBadge.svelte';
  import { toggleOff } from '$lib/boxes';
  import { documentKindLabels, formatDate } from '$lib/format';
  import { pageLookup, pageName } from '$lib/pages';
  import { findingInput, observationInput, readingState } from '$lib/reading';
  import { session } from '$lib/state/session.svelte';
  import { ACCEPT, addedMessage, pickFiles } from '$lib/upload';

  // Reviews one inbox item before filing (HW-READ-4, HW-AI-7): the pages
  // and their masked copies, the reading's status and its drafts.
  let { data } = $props();

  const item = $derived(data.detail.item);
  const reading = $derived(item.reading);
  const status = $derived(readingState(item));
  const pages = $derived(data.detail.pages);
  const pageOf = $derived(pageLookup(pages));
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
  // Pages start folded, except one waiting for its mask to be checked.
  const folds = new Folds({ startOpen: (id) => pages.find((p) => p.id === id)?.mask?.status === 'needs_check' });
  const pageIds = $derived(pages.map((p) => p.id));
  let view = $state<Record<string, 'masked' | 'original'>>({});
  let drawing = $state<string | null>(null);
  let toggling = $state<string | null>(null);
  let editingObs = $state<Observation | null>(null);
  let editingFinding = $state<Finding | null>(null);
  let addInput = $state<HTMLInputElement>();
  let camera = $state<ItemCamera>();
  let phoneOpen = $state(false);

  // More material for the same paper, such as its X-ray: a phone opens its
  // own camera, a computer shows a QR code for the iPhone.
  const addItems: MenuItem[] = [
    { label: 'Add files', icon: 'file', onselect: () => addInput?.click() },
    { label: 'Take photos', icon: 'camera', class: 'hidden pointer-coarse:flex', onselect: () => camera?.take() },
    {
      label: 'Take photos with iPhone',
      icon: 'camera',
      class: 'flex pointer-coarse:hidden',
      onselect: () => (phoneOpen = true),
    },
  ];

  // More files for the same paper, such as its X-ray image or a page that
  // came later. The whole item is read again with them.
  const addFiles = action(
    async (files: File[]) => {
      const { accepted, skipped } = pickFiles(files);
      if (skipped.length) toasts.push('warning', `Skipped: ${skipped.join(', ')}`);
      if (!accepted.length) return;
      const resp = await session.api.addFiles(item.id, accepted);
      toasts.success(addedMessage(resp.files));
      await invalidateAll();
    },
    { errorContext: 'Could not add the files' },
  );

  // Photos the phone camera already sent ahead (HW-IMP-11), added by id.
  const addPhotos = action(
    async (fileIds: string[]) => {
      await session.api.addFiles(item.id, [], fileIds);
      toasts.success(
        `${fileIds.length} ${fileIds.length === 1 ? 'photo' : 'photos'} added; the item is read again with them`,
      );
      await invalidateAll();
    },
    { errorContext: 'Could not add the photos' },
  );

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

  const canMask = (p: Page) => editable && !!p.mask && p.hasImage && data.detail.aiEnabled;

  async function toggleBox(p: Page, key: string) {
    if (toggling) return;
    toggling = p.id;
    try {
      await session.api.setPageOff(item.id, p.id, toggleOff(p.mask?.auto ?? [], key));
      await invalidateAll();
    } finally {
      toggling = null;
    }
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
      <MenuButton label="Add more" icon="plus" busy={addFiles.busy} busyText="Adding…" items={addItems} />
      <Button icon="trash" errorContext="Could not discard" onclick={discard}>Discard</Button>
    {/if}
  {/snippet}
</PageTitle>

{#if editable}
  <ItemCamera bind:this={camera} itemId={item.id} busy={addPhotos.busy} onsend={(ids) => addPhotos.run(ids)} />
{/if}

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
      Look at each page as the AI would see it. Black boxes hide names, ids, phone numbers and barcodes. Click a
      numbered box to let a line through when it hides a result, use Hide more to black out anything else, then send.
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
      {pageOf}
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
      {pageOf}
      onedit={(f) => (editingFinding = f)}
      onstatus={setFindingStatus}
    />
  </Card>
{/if}

<div class="mb-3 flex items-center gap-3">
  <h2 class="heading">Pages</h2>
  {#if pages.length > 1}
    <button type="button" class="btn-ghost px-2.5 py-1 text-xs" onclick={() => folds.toggleAll(pageIds)}
      >{folds.allFolded(pageIds) ? 'Expand all' : 'Fold all'}</button
    >
  {/if}
</div>
<div class="space-y-5">
  {#each pages as p, i (p.id)}
    <section class="panel p-4">
      <header class="flex flex-wrap items-center gap-2 {folds.isOpen(p.id) ? 'mb-3' : ''}">
        <FoldToggle
          open={folds.isOpen(p.id)}
          label="page {i + 1}"
          controls="page-body-{p.id}"
          ontoggle={() => folds.toggle(p.id)}
        />
        <span class="font-medium">{pageName(i + 1, p.title)}</span>
        {#if !p.mask && reading && ['done', 'needs_check'].includes(reading.status)}
          <span
            class="tag bg-ink-900/5 dark:bg-ink-50/10"
            title="Film photos and blank pages are kept but never read by AI">Not sent: too little text</span
          >
        {:else if p.mask?.status === 'needs_check'}
          <span class="tag bg-warn-500/15 text-warn-800 dark:text-warn-200">Needs a check</span>
        {:else if p.mask?.status === 'checked'}
          <span class="tag bg-accent-600/10 text-accent-700 dark:text-accent-300">Checked</span>
        {/if}
        {#if p.mask}
          {@const through = (p.mask.auto ?? []).filter((b) => b.off).length}
          <span class="muted text-xs"
            >{p.mask.maskedLines} lines hidden{through
              ? `, ${through} ${through === 1 ? 'box' : 'boxes'} let through`
              : ''}</span
          >
        {/if}
        <span class="ml-auto flex gap-1 {folds.isOpen(p.id) ? '' : 'hidden'}">
          {#if p.mask}
            <button
              type="button"
              class="btn-ghost px-2.5 py-1 text-xs {viewOf(p) === 'masked'
                ? 'bg-accent-600/10 text-accent-700 dark:text-accent-300'
                : ''}"
              onclick={() => (view[p.id] = 'masked')}>What the AI sees</button
            >
            <button
              type="button"
              class="btn-ghost px-2.5 py-1 text-xs {viewOf(p) === 'original'
                ? 'bg-accent-600/10 text-accent-700 dark:text-accent-300'
                : ''}"
              onclick={() => {
                view[p.id] = 'original';
                if (drawing === p.id) drawing = null;
              }}>Original</button
            >
          {/if}
          {#if canMask(p)}
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
      {#if folds.isOpen(p.id)}
        <div id="page-body-{p.id}">
          {#if p.mask?.reasons?.length}
            <ul class="mb-3 list-disc pl-5 text-xs text-warn-800 dark:text-warn-200">
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
            auto={p.mask?.auto ?? []}
            ontoggle={canMask(p) && toggling === null ? (key) => void toggleBox(p, key) : undefined}
          />
          {#if viewOf(p) === 'masked' && p.mask?.auto?.length}
            <details class="mt-3" open={canMask(p)}>
              <summary class="muted cursor-pointer text-xs">Numbered boxes ({p.mask.auto.length})</summary>
              {#if canMask(p)}
                <p class="muted mt-2 text-xs">
                  Untick a box that hides a result or another line the AI needs. That line is then sent in the image and
                  the text.
                </p>
              {/if}
              <ul class="mt-2 max-h-72 space-y-1 overflow-auto text-sm">
                {#each p.mask.auto as box, n (box.key)}
                  <li>
                    <label class="flex items-start gap-2 {canMask(p) ? 'cursor-pointer' : ''}">
                      <input
                        type="checkbox"
                        class="mt-1"
                        checked={!box.off}
                        disabled={!canMask(p) || toggling !== null}
                        onchange={() => void toggleBox(p, box.key)}
                      />
                      <span
                        class="mt-0.5 min-w-6 rounded px-1 text-center text-xs font-semibold text-ink-50 {box.off
                          ? 'bg-warn-500'
                          : 'bg-accent-600'}">{n + 1}</span
                      >
                      <span class={box.off ? 'muted line-through' : ''}
                        >{box.kind === 'barcode' ? 'Barcode or QR code' : box.text}</span
                      >
                    </label>
                  </li>
                {/each}
              </ul>
            </details>
          {/if}
          {#if p.text}
            {@const masked = viewOf(p) === 'masked' && p.mask}
            <details class="mt-3">
              <summary class="muted cursor-pointer text-xs"
                >{masked ? 'Text the AI sees (personal lines replaced with [ẩn])' : 'Text read on the Mac'}</summary
              >
              <pre
                class="mt-2 max-h-80 overflow-auto rounded-xl bg-ink-900/5 p-3 text-xs whitespace-pre-wrap dark:bg-ink-50/5">{masked
                  ? p.mask?.text
                  : p.text}</pre>
            </details>
          {/if}
        </div>
      {/if}
    </section>
  {:else}
    <section class="panel">
      <EmptyState title="No pages yet" detail="The Mac is still reading the files. This page refreshes on its own." />
    </section>
  {/each}
</div>

<input
  bind:this={addInput}
  class="hidden"
  type="file"
  multiple
  accept={ACCEPT}
  onchange={(e) => {
    const files = [...(e.currentTarget.files ?? [])];
    if (files.length) void addFiles.run(files);
    e.currentTarget.value = '';
  }}
/>
{#if phoneOpen}
  <PhoneCaptureModal
    open
    itemId={item.id}
    onclose={() => {
      phoneOpen = false;
      void invalidateAll();
    }}
  />
{/if}
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
