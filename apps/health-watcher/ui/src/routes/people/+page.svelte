<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import type { Person } from '$lib/api/types';
  import Button from '$lib/components/Button.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import PageTitle from '$lib/components/PageTitle.svelte';
  import PersonFormModal from '$lib/components/PersonFormModal.svelte';
  import { formatDate } from '$lib/format';
  import { session } from '$lib/state/session.svelte';

  let { data } = $props();
  let editing = $state<Person | null>(null);
  let open = $state(false);

  function edit(p: Person | null) {
    editing = p;
    open = true;
  }

  async function remove(p: Person) {
    if (!confirm(`Delete ${p.displayName}? Only a person with no visits can be deleted.`)) return;
    await session.api.deletePerson(p.id);
    await invalidateAll();
  }

  const memberName = (userId?: string) => data.members.find((m) => m.userId === userId)?.displayName;
</script>

<PageTitle title="People" subtitle="Everyone whose records you keep. A person doesn't need an account.">
  {#snippet actions()}
    {#if session.canEdit}<Button variant="primary" icon="plus" onclick={() => edit(null)}>Add a person</Button>{/if}
  {/snippet}
</PageTitle>

<section class="glass divide-y divide-slate-900/5 dark:divide-white/5">
  {#each data.people as p (p.id)}
    <div class="flex items-center gap-3 px-4 py-3">
      <a href="{base}/people/{p.id}" class="min-w-0 flex-1">
        <span class="block font-medium">{p.displayName}</span>
        <span class="muted block text-xs">
          {p.birthDate ? `Born ${formatDate(p.birthDate)} · ` : ''}{p.visitCount} visits{memberName(p.userId)
            ? ` · account: ${memberName(p.userId)}`
            : ''}
        </span>
      </a>
      {#if session.canEdit}
        <Button size="icon" icon="pencil" label="Edit {p.displayName}" onclick={() => edit(p)} />
        <Button
          size="icon"
          icon="trash"
          label="Delete {p.displayName}"
          errorContext="Could not delete"
          onclick={() => remove(p)}
        />
      {/if}
    </div>
  {:else}
    <EmptyState title="No people yet" detail="Add yourself, your children or your parents." />
  {/each}
</section>

<PersonFormModal
  {open}
  person={editing}
  members={data.members}
  onclose={() => (open = false)}
  onsaved={() => {
    open = false;
    void invalidateAll();
  }}
/>
