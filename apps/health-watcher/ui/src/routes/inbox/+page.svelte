<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import type { InboxGroup, InboxItem } from '$lib/api/types';
  import { Button, EmptyState, PageTitle } from '@libs/ui-kit';
  import FileItemModal from '$lib/components/FileItemModal.svelte';
  import FilePreview from '$lib/components/FilePreview.svelte';
  import ReadingBadge from '$lib/components/ReadingBadge.svelte';
  import UploadPanel from '$lib/components/UploadPanel.svelte';
  import { formatDate, visitTypeLabels } from '$lib/format';
  import { readingState } from '$lib/reading';
  import { session } from '$lib/state/session.svelte';

  let { data } = $props();
  let filing = $state<InboxItem | null>(null);

  const states = $derived(new Map(data.items.map((item) => [item.id, readingState(item)])));
  const working = $derived([...states.values()].some((s) => s.tone === 'busy' || s.tone === 'warn'));
  const personName = $derived(new Map(data.people.map((p) => [p.id, p.displayName])));

  // While the Mac or the AI is reading, refresh every few seconds.
  $effect(() => {
    if (!working) return;
    const timer = setInterval(() => void invalidateAll(), 4000);
    return () => clearInterval(timer);
  });

  async function discard(item: InboxItem) {
    if (!confirm('Discard this item? Its files are deleted unless a record already uses them.')) return;
    await session.api.discard(item.id);
    await invalidateAll();
  }

  async function fileGroup(g: InboxGroup) {
    await session.api.fileGroup({
      visit: {
        personId: g.personId,
        date: g.date,
        type: g.visitType,
        title: '',
        note: '',
        providerName: g.providerName,
        providerKind: g.providerKind,
      },
      documents: g.items.map((item) => ({
        itemId: item.id,
        kind: item.reading?.proposal?.kind ?? 'other',
        title: item.reading?.proposal?.title ?? '',
      })),
    });
    await invalidateAll();
  }
</script>

<PageTitle
  title="Inbox"
  subtitle="Everything you upload lands here first. The Mac reads the text, then the AI suggests how to file each item."
/>

{#if session.canEdit}
  <UploadPanel people={data.people} onuploaded={() => void invalidateAll()} />
{/if}

{#if data.groups.length > 0}
  <h2 class="mt-8 mb-3 heading">
    Suggested visits ({data.groups.length})
  </h2>
  <div class="space-y-3">
    {#each data.groups as g (`${g.personId}-${g.date}-${g.providerName}`)}
      <section class="panel flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
        <div class="min-w-0 flex-1">
          <p class="font-medium">
            {personName.get(g.personId) ?? 'Someone'} · {formatDate(g.date)}
          </p>
          <p class="muted text-xs">
            {visitTypeLabels[g.visitType]}{g.providerName ? ` · ${g.providerName}` : ''} · {g.items.length}
            {g.items.length === 1 ? 'item' : 'items'}
          </p>
        </div>
        {#if session.canEdit}
          <Button variant="primary" success="Filed" errorContext="Could not file" onclick={() => fileGroup(g)}>
            File as one visit
          </Button>
        {/if}
      </section>
    {/each}
  </div>
{/if}

<h2 class="mt-8 mb-3 heading">
  Waiting to be filed ({data.items.length})
</h2>
<div class="space-y-3">
  {#each data.items as item (item.id)}
    {@const status = states.get(item.id)}
    <section class="panel flex flex-col gap-3 p-4 sm:flex-row sm:items-center">
      <div class="grid w-full grid-cols-4 gap-2 sm:w-64 sm:shrink-0">
        {#each item.files.slice(0, 4) as f (f.id)}<FilePreview file={f} compact />{/each}
      </div>
      <div class="min-w-0 flex-1 space-y-1">
        <a class="link font-medium" href="{base}/inbox/{item.id}">
          {item.reading?.proposal?.title ||
            (item.files.length === 1 ? item.files[0].originalName || '1 file' : `${item.files.length} pages`)}
        </a>
        <p class="muted text-xs">
          Added {formatDate(item.createdAt)}{item.source === 'shortcut'
            ? ' from the iPhone Shortcut'
            : ''}{item.personId ? ` · ${personName.get(item.personId) ?? ''}` : ''}
        </p>
        {#if item.note}<p class="text-sm">{item.note}</p>{/if}
        {#if status}<ReadingBadge {status} />{/if}
      </div>
      <div class="flex gap-2">
        <a class="btn-ghost" href="{base}/inbox/{item.id}">Review</a>
        {#if session.canEdit}
          <Button variant="primary" onclick={() => (filing = item)}>File</Button>
          <Button errorContext="Could not discard" onclick={() => discard(item)}>Discard</Button>
        {/if}
      </div>
    </section>
  {:else}
    <section class="panel"><EmptyState title="The inbox is empty" detail="Uploaded files show up here." /></section>
  {/each}
</div>

<FileItemModal
  item={filing}
  people={data.people}
  providers={data.providers}
  onclose={() => (filing = null)}
  onfiled={() => {
    filing = null;
    void invalidateAll();
  }}
/>
