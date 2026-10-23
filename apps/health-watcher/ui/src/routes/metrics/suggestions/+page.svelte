<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { EmptyState, Icon, PageTitle } from '@libs/ui-kit';
  import SuggestionCard from '$lib/components/SuggestionCard.svelte';
  import { catalog } from '$lib/state/catalog.svelte';

  // Suggestions for printed names no metric knows (HW-MET-8): after each
  // filing the app asks the AI, with the names and units alone, whether each
  // one is a known metric under another name or a test the catalog lacks.
  // Nothing changes until a person approves.
  let { data } = $props();

  async function decided(added: boolean) {
    if (added) await catalog.reload();
    await invalidateAll();
  }
</script>

<svelte:head><title>Metric suggestions · Health Watcher</title></svelte:head>

<PageTitle
  title="Metric suggestions"
  subtitle={data.suggestions.length
    ? `${data.suggestions.length} ${data.suggestions.length === 1 ? 'name' : 'names'} no metric knows yet`
    : 'Nothing to decide'}
/>

<p class="panel mb-5 flex items-start gap-2 p-3 text-sm" role="note">
  <Icon name="info" size={16} />
  <span
    >Only the printed names and units went to the AI, no values, people or dates. Until you decide, the values stay
    under Not in the catalog, by the name printed on the paper. Keep as printed leaves them there and stops asking.</span
  >
</p>

<ul class="space-y-4">
  {#each data.suggestions as s (s.id)}
    <li><SuggestionCard suggestion={s} ondecided={decided} /></li>
  {:else}
    <li class="panel">
      <EmptyState
        title="No suggestions"
        detail="When a filed paper prints a test no metric knows, a suggestion for it shows here."
      />
    </li>
  {/each}
</ul>
