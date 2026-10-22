<script lang="ts">
  import type { Member, Person, PersonInput, Sex } from '$lib/api/types';
  import { todayISO } from '$lib/format';
  import { action, Field, Modal } from '@libs/ui-kit';
  import { session } from '$lib/state/session.svelte';

  interface Props {
    open: boolean;
    /** The person to edit; null adds a new one. */
    person: Person | null;
    members: Member[];
    onclose: () => void;
    onsaved: (person: Person) => void;
  }

  let { open, person, members, onclose, onsaved }: Props = $props();

  let displayName = $state('');
  let birthDate = $state('');
  let sex = $state<Sex>('');
  let userId = $state('');

  // Refill the form each time it opens.
  $effect(() => {
    if (!open) return;
    displayName = person?.displayName ?? '';
    birthDate = person?.birthDate ?? '';
    sex = person?.sex ?? '';
    userId = person?.userId ?? '';
  });

  const save = action(
    async () => {
      const input: PersonInput = { displayName, birthDate, sex, userId };
      const saved = person ? await session.api.updatePerson(person.id, input) : await session.api.createPerson(input);
      onsaved(saved);
    },
    { errorContext: 'Could not save', success: 'Saved' },
  );
</script>

<Modal {open} title={person ? `Edit ${person.displayName}` : 'Add a person'} {onclose}>
  <form
    id="person-form"
    class="space-y-4"
    onsubmit={(e) => {
      e.preventDefault();
      void save.run();
    }}
  >
    <Field id="p-name" label="Name">
      <input id="p-name" class="field" bind:value={displayName} required maxlength="80" />
    </Field>
    <div class="grid gap-4 sm:grid-cols-2">
      <Field id="p-birth" label="Date of birth" hint="Reference ranges depend on age.">
        <input id="p-birth" class="field" type="date" bind:value={birthDate} max={todayISO()} />
      </Field>
      <Field id="p-sex" label="Sex at birth" hint="Some ranges differ by sex.">
        <select id="p-sex" class="field" bind:value={sex}>
          <option value="">Not set</option>
          <option value="female">Female</option>
          <option value="male">Male</option>
        </select>
      </Field>
    </div>
    <Field id="p-user" label="Linked account" hint="Only for a person who logs in themselves.">
      <select id="p-user" class="field" bind:value={userId}>
        <option value="">None</option>
        {#each members as m (m.userId)}
          <option value={m.userId}>{m.displayName} ({m.email})</option>
        {/each}
      </select>
    </Field>
  </form>
  {#snippet footer()}
    <button class="btn-ghost" type="button" onclick={onclose}>Cancel</button>
    <button class="btn-primary" type="submit" form="person-form" disabled={save.busy}
      >{save.busy ? 'Saving…' : 'Save'}</button
    >
  {/snippet}
</Modal>
