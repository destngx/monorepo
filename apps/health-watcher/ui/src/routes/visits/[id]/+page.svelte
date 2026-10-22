<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import type { HealthDocument } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import FilePreview from '$lib/components/FilePreview.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import PageTitle from '$lib/components/PageTitle.svelte';
  import VisitFormModal from '$lib/components/VisitFormModal.svelte';
  import { documentKindLabels, formatDate, providerKindLabels, visitTypeLabels } from '$lib/format';
  import { session } from '$lib/state/session.svelte';

  let { data } = $props();
  let editing = $state(false);
  let openDoc = $state<string | null>(null);
  const documents = $derived(data.visit.documents ?? []);

  async function deleteDocument(doc: HealthDocument) {
    const label = doc.title || documentKindLabels[doc.kind];
    if (!confirm(`Delete "${label}" and its files? This can't be undone.`)) return;
    await session.api.deleteDocument(doc.id);
    await invalidateAll();
  }

  async function deleteVisit() {
    if (!confirm('Delete this visit?')) return;
    await session.api.deleteVisit(data.visit.id);
    await goto(`${base}/people/${data.visit.personId}`);
  }
</script>

<a class="link mb-3 inline-flex items-center gap-1 text-sm" href="{base}/people/{data.person.id}">
  <Icon name="chevronLeft" size={16} />{data.person.displayName}
</a>

<PageTitle
  title={data.visit.title || visitTypeLabels[data.visit.type]}
  subtitle="{formatDate(data.visit.date)} · {visitTypeLabels[data.visit.type]}{data.visit.provider
    ? ` · ${data.visit.provider.name} (${providerKindLabels[data.visit.provider.kind].toLowerCase()})`
    : ''}"
>
  {#snippet actions()}
    {#if session.canEdit}
      <Button icon="pencil" onclick={() => (editing = true)}>Edit</Button>
      {#if documents.length === 0}
        <Button icon="trash" errorContext="Could not delete" onclick={deleteVisit}>Delete</Button>
      {/if}
    {/if}
  {/snippet}
</PageTitle>

{#if data.visit.note}
  <Card title="Note" class="mb-5"><p class="text-sm whitespace-pre-wrap">{data.visit.note}</p></Card>
{/if}

<h2 class="mb-3 heading">Documents</h2>
<div class="space-y-4">
  {#each documents as doc (doc.id)}
    <section class="panel p-4">
      <header class="flex flex-wrap items-center gap-3">
        <span class="grid size-9 place-items-center rounded-xl bg-accent-600/10 text-accent-700 dark:text-accent-300"
          ><Icon name="file" size={18} /></span
        >
        <div class="min-w-0 flex-1">
          <p class="truncate font-medium">{doc.title || documentKindLabels[doc.kind]}</p>
          <p class="muted text-xs">
            {documentKindLabels[doc.kind]} · {doc.files.length}
            {doc.files.length === 1 ? 'file' : 'pages'}
          </p>
        </div>
        <a class="btn-ghost px-2.5 py-1 text-xs" href="{base}/documents/{doc.id}">Open</a>
        <Button size="sm" onclick={() => (openDoc = openDoc === doc.id ? null : doc.id)}>
          {openDoc === doc.id ? 'Hide files' : 'Files'}
        </Button>
        {#if session.canEdit}
          <Button
            size="icon"
            icon="trash"
            label="Delete document"
            errorContext="Could not delete"
            onclick={() => deleteDocument(doc)}
          />
        {/if}
      </header>
      {#if openDoc === doc.id}
        <div class="mt-4 space-y-4">
          {#each doc.files as file (file.id)}<FilePreview {file} />{/each}
        </div>
      {:else}
        <div class="mt-3 grid grid-cols-4 gap-2 sm:grid-cols-6 lg:grid-cols-8">
          {#each doc.files as file (file.id)}<FilePreview {file} compact />{/each}
        </div>
      {/if}
      {#if doc.kind === 'imaging_film'}
        <p class="muted mt-3 text-xs">
          Film photos are kept for reference only. The app never reads them with AI; the written radiology report is the
          part to rely on.
        </p>
      {/if}
    </section>
  {:else}
    <section class="panel">
      <EmptyState title="No documents" detail="Upload files in the inbox and file them under this visit." />
    </section>
  {/each}
</div>

<VisitFormModal
  open={editing}
  visit={data.visit}
  personId={data.visit.personId}
  people={data.people}
  providers={data.providers}
  onclose={() => (editing = false)}
  onsaved={() => {
    editing = false;
    void invalidateAll();
  }}
/>
