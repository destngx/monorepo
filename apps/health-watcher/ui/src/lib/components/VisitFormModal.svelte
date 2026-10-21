<script lang="ts">
  import type { Person, Provider, Visit, VisitInput } from '$lib/api/types';
  import { blankVisit } from '$lib/visits';
  import { action } from '$lib/state/action.svelte';
  import { session } from '$lib/state/session.svelte';
  import Modal from './Modal.svelte';
  import VisitFields from './VisitFields.svelte';

  interface Props {
    open: boolean;
    /** The visit to edit; null adds one for personId. */
    visit: Visit | null;
    personId: string;
    people: Person[];
    providers: Provider[];
    onclose: () => void;
    onsaved: (visit: Visit) => void;
  }

  let { open, visit, personId, people, providers, onclose, onsaved }: Props = $props();

  let value = $state<VisitInput>(blankVisit(''));

  $effect(() => {
    if (!open) return;
    value = visit
      ? {
          personId: visit.personId,
          date: visit.date,
          type: visit.type,
          title: visit.title,
          note: visit.note,
          providerName: visit.provider?.name ?? '',
          providerKind: visit.provider?.kind ?? '',
        }
      : blankVisit(personId);
  });

  const save = action(
    async () => {
      const saved = visit ? await session.api.updateVisit(visit.id, value) : await session.api.createVisit(value);
      onsaved(saved);
    },
    { errorContext: 'Could not save the visit', success: 'Saved' },
  );
</script>

<Modal {open} title={visit ? 'Edit visit' : 'Add a visit'} {onclose} wide>
  <form
    id="visit-form"
    onsubmit={(e) => {
      e.preventDefault();
      void save.run();
    }}
  >
    <VisitFields bind:value {people} {providers} idPrefix="vf" />
  </form>
  {#snippet footer()}
    <button class="btn-ghost" type="button" onclick={onclose}>Cancel</button>
    <button class="btn-primary" type="submit" form="visit-form" disabled={save.busy}
      >{save.busy ? 'Saving…' : 'Save'}</button
    >
  {/snippet}
</Modal>
