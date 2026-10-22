<script lang="ts">
  import type { DocumentKind, InboxItem, Person, Provider, Visit, VisitInput } from '$lib/api/types';
  import { documentKindLabels, formatDate, options, visitTypeLabels } from '$lib/format';
  import { action } from '$lib/state/action.svelte';
  import { session } from '$lib/state/session.svelte';
  import { proposalVisit } from '$lib/reading';
  import Field from './Field.svelte';
  import FilePreview from './FilePreview.svelte';
  import FilmGuide from './FilmGuide.svelte';
  import Modal from './Modal.svelte';
  import VisitFields from './VisitFields.svelte';

  // Files an inbox item as a document under an existing or new visit. The
  // AI reading's proposal, when there is one, prefills the form (HW-READ-3).
  interface Props {
    item: InboxItem | null;
    people: Person[];
    providers: Provider[];
    onclose: () => void;
    onfiled: () => void;
  }

  let { item, people, providers, onclose, onfiled }: Props = $props();

  let kind = $state<DocumentKind>('lab_result');
  let title = $state('');
  let mode = $state<'new' | 'existing'>('new');
  let visit = $state<VisitInput>(proposalVisit(undefined, [], ''));
  let visitId = $state('');
  let visits = $state<Visit[]>([]);

  $effect(() => {
    if (!item) return;
    const proposal = item.reading?.status === 'done' ? item.reading.proposal : undefined;
    kind = proposal?.kind ?? 'lab_result';
    title = (proposal?.title || item.note).slice(0, 120);
    mode = 'new';
    visit = proposalVisit(proposal, people, item.personId ?? '');
    visitId = '';
  });

  // The existing visits of the chosen person, newest first.
  $effect(() => {
    const personId = visit.personId;
    if (mode !== 'existing' || !personId) {
      visits = [];
      return;
    }
    let cancelled = false;
    session.api
      .visits(personId)
      .then((list) => {
        if (!cancelled) visits = list;
      })
      .catch(() => {
        if (!cancelled) visits = [];
      });
    return () => {
      cancelled = true;
    };
  });

  const file = action(
    async () => {
      if (!item) return;
      await session.api.file(item.id, mode === 'existing' ? { visitId, kind, title } : { visit, kind, title });
      onfiled();
    },
    { errorContext: 'Could not file', success: 'Filed' },
  );
</script>

<Modal open={item !== null} title="File into the records" {onclose} wide>
  {#if item}
    <div class="mb-5 grid grid-cols-4 gap-2 sm:grid-cols-6">
      {#each item.files as f (f.id)}<FilePreview file={f} compact />{/each}
    </div>
    <form
      id="file-form"
      class="space-y-5"
      onsubmit={(e) => {
        e.preventDefault();
        void file.run();
      }}
    >
      <div class="grid gap-4 sm:grid-cols-2">
        <Field id="fi-kind" label="What is it">
          <select id="fi-kind" class="field" bind:value={kind}>
            {#each options(documentKindLabels) as o (o.value)}<option value={o.value}>{o.label}</option>{/each}
          </select>
        </Field>
        <Field id="fi-title" label="Title" hint="Optional, e.g. Blood test, CT chest.">
          <input id="fi-title" class="field" bind:value={title} maxlength="120" />
        </Field>
      </div>
      {#if kind === 'imaging_film'}
        <p class="rounded-xl bg-warn-500/10 px-3 py-2 text-xs text-warn-800 dark:text-warn-200">
          Film photos are stored and shown only. File the written radiology report too; it's the part that can be
          searched and read.
        </p>
        <FilmGuide />
      {/if}

      <fieldset>
        <legend class="label">Visit</legend>
        <div class="flex gap-2">
          <label class="btn-ghost {mode === 'new' ? 'bg-accent-600/10 text-accent-700 dark:text-accent-300' : ''}">
            <input class="sr-only" type="radio" bind:group={mode} value="new" />New visit
          </label>
          <label class="btn-ghost {mode === 'existing' ? 'bg-accent-600/10 text-accent-700 dark:text-accent-300' : ''}">
            <input class="sr-only" type="radio" bind:group={mode} value="existing" />Existing visit
          </label>
        </div>
      </fieldset>

      {#if mode === 'new'}
        <VisitFields bind:value={visit} {people} {providers} idPrefix="fi" />
      {:else}
        <Field id="fi-person" label="Person">
          <select id="fi-person" class="field" bind:value={visit.personId} required>
            <option value="" disabled>Choose…</option>
            {#each people as p (p.id)}<option value={p.id}>{p.displayName}</option>{/each}
          </select>
        </Field>
        <Field id="fi-visit" label="Visit">
          <select id="fi-visit" class="field" bind:value={visitId} required disabled={!visit.personId}>
            <option value="" disabled>{visits.length ? 'Choose…' : 'No visits for this person yet'}</option>
            {#each visits as v (v.id)}
              <option value={v.id}>
                {formatDate(v.date)} · {v.title || visitTypeLabels[v.type]}{v.provider ? ` · ${v.provider.name}` : ''}
              </option>
            {/each}
          </select>
        </Field>
      {/if}
    </form>
  {/if}
  {#snippet footer()}
    <button class="btn-ghost" type="button" onclick={onclose}>Cancel</button>
    <button class="btn-primary" type="submit" form="file-form" disabled={file.busy}
      >{file.busy ? 'Filing…' : 'File'}</button
    >
  {/snippet}
</Modal>
