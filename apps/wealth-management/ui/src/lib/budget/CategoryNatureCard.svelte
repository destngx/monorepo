<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { api } from '$lib/api/endpoints';
  import type { Category } from '$lib/api/types';
  import Card from '$lib/components/Card.svelte';
  import EmptyState from '$lib/components/EmptyState.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import CategoryDefinitionModal from './CategoryDefinitionModal.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { groupCategories, NATURE_OPTIONS, natureChoice, type NatureChoice } from './natures';

  // Category nature editor (WM-LEND-5): writes Categories column B. Each row
  // also opens the definition and split tags editor (WM-CAT-1, WM-CAT-2).
  let { categories }: { categories: Category[] } = $props();

  const groups = $derived(groupCategories(categories));
  // Choices saved or being saved, shown until the reloaded categories catch up.
  let pending = $state<Record<string, NatureChoice>>({});
  let saving = $state<string | null>(null);
  let defining = $state<Category | null>(null);

  async function save(category: Category, nature: NatureChoice) {
    const previous = natureChoice(category);
    if (nature === previous) return;
    pending[category.name] = nature;
    saving = category.name;
    try {
      await api.setCategoryNature({ category: category.name, nature });
      const label = NATURE_OPTIONS.find((option) => option.value === nature)?.label ?? nature;
      toasts.push('success', `${category.name} is now ${label.toLowerCase()}.`);
      await invalidateAll();
    } catch (error) {
      toasts.error(error, `Could not change ${category.name}`);
    } finally {
      delete pending[category.name];
      saving = null;
    }
  }
</script>

<Card
  title="Categories"
  subtitle="Fixed and essential are needs, flexible is wants. Lending categories hold money lent and repaid: they leave income and expense and feed Money lent. Define a category to say what belongs in it; split tags show a share of its spend."
>
  {#if groups.length}
    <div class="grid gap-6 md:grid-cols-2 xl:grid-cols-3">
      {#each groups as group (group.label)}
        <div>
          <h3 class="label">{group.label}</h3>
          <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
            {#each group.categories as category (category.name)}
              {@const id = `nature-${group.label}-${category.name}`}
              <li class="flex items-center justify-between gap-3 py-1.5">
                <div class="min-w-0">
                  <label for={id} class="block truncate">{category.name}</label>
                  {#if category.definition || category.splitTags?.length}
                    <p class="muted truncate text-xs" title={category.definition}>
                      {#each category.splitTags ?? [] as tag (tag)}<span class="mr-1 font-medium">#{tag}</span
                        >{/each}{category.definition ?? ''}
                    </p>
                  {/if}
                </div>
                <button
                  type="button"
                  class="btn-ghost shrink-0 px-1.5"
                  aria-label="Define {category.name}"
                  title="Definition and split tags"
                  onclick={() => (defining = category)}><Icon name="pencil" size={14} /></button
                >
                <select
                  {id}
                  class="field w-36 py-1 text-xs"
                  value={pending[category.name] ?? natureChoice(category)}
                  disabled={saving === category.name}
                  onchange={(event) => save(category, event.currentTarget.value as NatureChoice)}
                >
                  {#each NATURE_OPTIONS as option (option.value)}
                    <option value={option.value} title={option.detail}>{option.label}</option>
                  {/each}
                </select>
              </li>
            {/each}
          </ul>
        </div>
      {/each}
    </div>
  {:else}
    <EmptyState title="No categories" detail="Categories come from the Categories tab." />
  {/if}
</Card>

<CategoryDefinitionModal category={defining} onclose={() => (defining = null)} />
