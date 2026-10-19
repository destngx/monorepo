<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { Category } from '$lib/api/types';
  import Modal from '$lib/components/Modal.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { parseSplitTags } from './natures';

  // Definition and split tags editor (WM-CAT-1, WM-CAT-2): writes Categories columns D and E.
  interface Props {
    category: Category | null;
    onclose: () => void;
  }

  let { category, onclose }: Props = $props();
  let definition = $state('');
  let tags = $state('');
  let saving = $state(false);

  // Seeded each time a category opens.
  $effect(() => {
    definition = category?.definition ?? '';
    tags = (category?.splitTags ?? []).join(', ');
  });

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (!category) return;
    saving = true;
    try {
      await api.setCategoryDefinition({
        category: category.name,
        definition: definition.trim(),
        splitTags: parseSplitTags(tags),
      });
      toasts.push('success', `${category.name} definition saved.`);
      onclose();
      await invalidateAll();
    } catch (error) {
      toasts.error(error, `Could not save ${category.name}`);
    } finally {
      saving = false;
    }
  }
</script>

<Modal open={category !== null} title="Define {category?.name ?? ''}" {onclose}>
  <form id="category-definition-form" class="space-y-4" onsubmit={save}>
    <div>
      <label class="label" for="category-definition">What belongs here</label>
      <textarea
        id="category-definition"
        class="field min-h-24"
        maxlength="500"
        bind:value={definition}
        placeholder="What goes in this category, and what goes elsewhere. The AI files transactions by it."
      ></textarea>
    </div>
    <div>
      <label class="label" for="category-split-tags">Split tags</label>
      <input
        id="category-split-tags"
        class="field"
        bind:value={tags}
        placeholder="Comma separated, e.g. a tag for one kind of spend inside this category"
      />
      <p class="muted mt-1 text-xs">
        Transactions with a split tag show as their own share of this category. Say in the definition when each tag
        applies, so the AI adds it.
      </p>
    </div>
  </form>
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={onclose}>Cancel</button>
    <button type="submit" form="category-definition-form" class="btn-primary" disabled={saving}
      >{saving ? 'Saving…' : 'Save'}</button
    >
  {/snippet}
</Modal>
