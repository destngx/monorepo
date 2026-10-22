<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { page as appPage } from '$app/state';
  import type { AuditEntry, DocumentKind, Finding, FindingInput, Observation, ObservationInput } from '$lib/api/types';
  import { Button, Card, FoldToggle, Icon, Modal, PageTitle, action } from '@libs/ui-kit';
  import FilePreview from '$lib/components/FilePreview.svelte';
  import { Folds } from '$lib/folds.svelte';
  import FindingList from '$lib/components/FindingList.svelte';
  import FindingModal from '$lib/components/FindingModal.svelte';
  import HistoryModal from '$lib/components/HistoryModal.svelte';
  import ObservationModal from '$lib/components/ObservationModal.svelte';
  import ObservationTable from '$lib/components/ObservationTable.svelte';
  import PageImage from '$lib/components/PageImage.svelte';
  import { documentKindLabels, formatDate, options, visitTypeLabels } from '$lib/format';
  import { findingInput, observationInput } from '$lib/reading';
  import { session } from '$lib/state/session.svelte';

  // The document viewer (HW-FIND-3): page images with zoom and their text,
  // the approved values and findings, editing with history (HW-REC-6) and
  // manual entry (HW-READ-7).
  let { data } = $props();

  const doc = $derived(data.detail.document);
  const visit = $derived(data.detail.visit);
  const pages = $derived(data.detail.pages);
  const pageIndex = $derived(new Map(pages.map((p, i) => [p.id, i + 1])));
  const pageNumber = (id: string | undefined) => (id ? pageIndex.get(id) : undefined);
  const editable = $derived(session.canEdit);
  // Only a document without values goes back to the inbox (HW-READ-8);
  // films are never read with AI (D5).
  const unfilable = $derived(
    editable &&
      doc.kind !== 'imaging_film' &&
      data.detail.observations.length === 0 &&
      data.detail.findings.length === 0,
  );
  // Search results link here with ?page= to open that page.
  const focusPage = $derived(appPage.url.searchParams.get('page'));
  const filesWithoutPages = $derived(doc.files.filter((f) => !pages.some((p) => p.fileId === f.id)));

  let zoom = $state(1);
  const folds = new Folds();
  const pageIds = $derived(pages.map((p) => p.id));
  let showText = $state<Record<string, boolean>>({});
  // Per page: the original, or the masked copy the AI was sent (HW-AI-7).
  let masked = $state<Record<string, boolean>>({});
  let obsForm = $state<{ id: string | null; value: ObservationInput } | null>(null);
  let findingForm = $state<{ id: string | null; value: FindingInput } | null>(null);
  let history = $state<AuditEntry[] | null>(null);
  let editingDoc = $state(false);
  let docKind = $state<DocumentKind>('other');
  let docTitle = $state('');

  $effect(() => {
    if (!focusPage) return;
    folds.open(focusPage);
    document.getElementById(`page-${focusPage}`)?.scrollIntoView({ block: 'start' });
  });

  const blankObs = (): ObservationInput => ({
    analyte: '',
    metricCode: '',
    value: '',
    unit: '',
    reference: '',
    flag: '',
    sampleDate: '',
    pageId: '',
  });
  const blankFinding = (): FindingInput => ({
    kind: 'diagnosis',
    section: '',
    text: '',
    icd10: '',
    drug: '',
    dose: '',
    frequency: '',
    duration: '',
    pageId: '',
  });

  async function saveObs(input: ObservationInput) {
    if (!obsForm) return;
    if (obsForm.id) await session.api.updateObservation(obsForm.id, input);
    else await session.api.addObservation(doc.id, input);
    await invalidateAll();
  }

  async function saveFinding(input: FindingInput) {
    if (!findingForm) return;
    if (findingForm.id) await session.api.updateFinding(findingForm.id, input);
    else await session.api.addFinding(doc.id, input);
    await invalidateAll();
  }

  async function deleteObs(o: Observation) {
    if (!confirm(`Delete "${o.analyte}"? It stays in the history.`)) return;
    await session.api.deleteObservation(o.id);
    await invalidateAll();
  }

  async function deleteFinding(f: Finding) {
    if (!confirm('Delete this finding? It stays in the history.')) return;
    await session.api.deleteFinding(f.id);
    await invalidateAll();
  }

  const saveDoc = action(
    async () => {
      await session.api.updateDocument(doc.id, { kind: docKind, title: docTitle });
      editingDoc = false;
      await invalidateAll();
    },
    { errorContext: 'Could not save', success: 'Saved' },
  );

  async function deleteDoc() {
    const label = doc.title || documentKindLabels[doc.kind];
    if (!confirm(`Delete "${label}", its values and its files? The values stay in the history.`)) return;
    await session.api.deleteDocument(doc.id);
    await goto(`${base}/visits/${visit.id}`);
  }

  async function unfile() {
    const question = 'Move this document back to the inbox to read it with AI? Its visit stays for filing it again.';
    if (!confirm(question)) return;
    const { itemId } = await session.api.unfileDocument(doc.id);
    await goto(`${base}/inbox/${itemId}`);
  }
</script>

<a class="link mb-3 inline-flex items-center gap-1 text-sm" href="{base}/visits/{visit.id}">
  <Icon name="chevronLeft" size={16} />{visit.title || visitTypeLabels[visit.type]}, {formatDate(visit.date)}
</a>

<PageTitle
  title={doc.title || documentKindLabels[doc.kind]}
  subtitle="{data.person.displayName} · {documentKindLabels[doc.kind]}{visit.provider
    ? ` · ${visit.provider.name}`
    : ''}"
>
  {#snippet actions()}
    {#if editable}
      <Button
        icon="pencil"
        onclick={() => {
          docKind = doc.kind;
          docTitle = doc.title;
          editingDoc = true;
        }}>Edit</Button
      >
      {#if unfilable}
        <Button icon="inbox" errorContext="Could not move it back" onclick={unfile}>Back to inbox</Button>
      {/if}
      <Button icon="trash" errorContext="Could not delete" onclick={deleteDoc}>Delete</Button>
    {/if}
  {/snippet}
</PageTitle>

<div class="grid gap-5 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
  <div class="space-y-5">
    <Card title="Values" subtitle={data.detail.observations.length ? undefined : 'No values yet.'}>
      {#snippet actions()}
        {#if editable}
          <Button size="sm" icon="plus" onclick={() => (obsForm = { id: null, value: blankObs() })}>Add</Button>
        {/if}
      {/snippet}
      {#if data.detail.observations.length}
        <ObservationTable
          observations={data.detail.observations}
          {editable}
          {pageNumber}
          onedit={(o) => (obsForm = { id: o.id, value: observationInput(o) })}
          ondelete={deleteObs}
          onhistory={async (o) => (history = await session.api.observationHistory(o.id))}
        />
      {/if}
    </Card>
    <Card title="Findings" subtitle={data.detail.findings.length ? undefined : 'No findings yet.'}>
      {#snippet actions()}
        {#if editable}
          <Button size="sm" icon="plus" onclick={() => (findingForm = { id: null, value: blankFinding() })}>Add</Button>
        {/if}
      {/snippet}
      {#if data.detail.findings.length}
        <FindingList
          findings={data.detail.findings}
          {editable}
          {pageNumber}
          onedit={(f) => (findingForm = { id: f.id, value: findingInput(f) })}
          ondelete={deleteFinding}
          onhistory={async (f) => (history = await session.api.findingHistory(f.id))}
        />
      {/if}
    </Card>
  </div>

  <div class="space-y-4">
    <div class="flex items-center gap-3">
      <h2 class="heading">Pages</h2>
      {#if pages.length > 1}
        <button type="button" class="btn-ghost px-2.5 py-1 text-xs" onclick={() => folds.toggleAll(pageIds)}
          >{folds.allFolded(pageIds) ? 'Expand all' : 'Fold all'}</button
        >
      {/if}
      {#if pages.some((p) => p.hasImage)}
        <label class="muted ml-auto flex items-center gap-2 text-xs">
          Zoom
          <input type="range" min="1" max="3" step="0.25" bind:value={zoom} aria-label="Zoom" />
        </label>
      {/if}
    </div>
    {#each pages as p, i (p.id)}
      <section id="page-{p.id}" class="panel scroll-mt-24 p-3">
        <header class="flex items-center gap-2 {folds.isOpen(p.id) ? 'mb-2' : ''}">
          <FoldToggle
            open={folds.isOpen(p.id)}
            label="page {i + 1}"
            controls="page-body-{p.id}"
            ontoggle={() => folds.toggle(p.id)}
          />
          <span class="text-sm font-medium">Page {i + 1}</span>
          <span class="ml-auto flex gap-1 {folds.isOpen(p.id) ? '' : 'hidden'}">
            {#if p.mask}
              <button
                type="button"
                class="btn-ghost px-2.5 py-1 text-xs {masked[p.id]
                  ? 'bg-accent-600/10 text-accent-700 dark:text-accent-300'
                  : ''}"
                aria-pressed={!!masked[p.id]}
                onclick={() => (masked[p.id] = !masked[p.id])}>Masked</button
              >
            {/if}
            {#if p.text}
              <button
                type="button"
                class="btn-ghost px-2.5 py-1 text-xs"
                onclick={() => (showText[p.id] = !showText[p.id])}>{showText[p.id] ? 'Image' : 'Text'}</button
              >
            {/if}
          </span>
        </header>
        {#if !folds.isOpen(p.id)}
          <!-- Folded: the header alone. -->
        {:else if showText[p.id]}
          <pre
            id="page-body-{p.id}"
            class="max-h-[70vh] overflow-auto rounded-xl bg-ink-900/5 p-3 text-xs whitespace-pre-wrap dark:bg-ink-50/5">{masked[
              p.id
            ] && p.mask
              ? p.mask.text
              : p.text}</pre>
        {:else}
          <div id="page-body-{p.id}"><PageImage page={p} {zoom} masked={!!masked[p.id] && !!p.mask} /></div>
        {/if}
      </section>
    {/each}
    {#each filesWithoutPages as f (f.id)}
      <FilePreview file={f} />
    {/each}
    {#if doc.kind === 'imaging_film'}
      <p class="muted text-xs">
        Film photos are kept for reference only. The app never reads them with AI; the written radiology report is the
        part to rely on.
      </p>
    {/if}
  </div>
</div>

<ObservationModal
  value={obsForm?.value ?? null}
  title={obsForm?.id ? 'Edit value' : 'Add a value'}
  {pages}
  onsave={saveObs}
  onclose={() => (obsForm = null)}
/>
<FindingModal
  value={findingForm?.value ?? null}
  title={findingForm?.id ? 'Edit finding' : 'Add a finding'}
  {pages}
  onsave={saveFinding}
  onclose={() => (findingForm = null)}
/>
<HistoryModal entries={history} onclose={() => (history = null)} />

<Modal open={editingDoc} title="Edit document" onclose={() => (editingDoc = false)}>
  <form
    id="doc-form"
    class="space-y-4"
    onsubmit={(e) => {
      e.preventDefault();
      void saveDoc.run();
    }}
  >
    <div>
      <label class="label" for="doc-kind">What is it</label>
      <select id="doc-kind" class="field" bind:value={docKind}>
        {#each options(documentKindLabels) as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
      </select>
    </div>
    <div>
      <label class="label" for="doc-title">Title</label>
      <input id="doc-title" class="field" bind:value={docTitle} maxlength="120" />
    </div>
  </form>
  {#snippet footer()}
    <button class="btn-ghost" type="button" onclick={() => (editingDoc = false)}>Cancel</button>
    <button class="btn-primary" type="submit" form="doc-form" disabled={saveDoc.busy}>
      {saveDoc.busy ? 'Saving…' : 'Save'}
    </button>
  {/snippet}
</Modal>
