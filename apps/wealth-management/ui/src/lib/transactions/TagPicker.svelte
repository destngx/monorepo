<script lang="ts">
  // Multi-select over existing tags, plus free entry for a new one. Fixed tags
  // (card and MCC tags) show first and are changed elsewhere, so they cannot be removed here.
  let {
    options,
    selected = $bindable([]),
    fixed = [],
  }: { options: string[]; selected: string[]; fixed?: { tag: string; title: string }[] } = $props();

  let query = $state('');
  const suggestions = $derived(
    options
      .filter((tag) => !selected.includes(tag) && tag.toLowerCase().includes(query.trim().toLowerCase()))
      .slice(0, 8),
  );

  // Enter adds the tag with exactly the typed name (in its saved spelling) or a new one, never a
  // longer tag that only contains the text; a suggestion is picked by clicking it.
  function addTyped() {
    const typed = query.trim().toLowerCase();
    add(options.find((tag) => tag.toLowerCase() === typed) ?? query);
  }

  function add(tag: string) {
    const value = tag.trim();
    if (value && !selected.includes(value)) selected = [...selected, value];
    query = '';
  }

  function remove(tag: string) {
    selected = selected.filter((t) => t !== tag);
  }
</script>

<div
  class="field flex flex-wrap items-center gap-1.5 py-1.5 focus-within:border-indigo-500 focus-within:ring-2 focus-within:ring-indigo-500/25"
>
  {#each fixed as { tag, title } (tag)}
    <span class="rounded-full bg-slate-900/5 px-2 py-0.5 text-xs dark:bg-white/10" {title}>{tag}</span>
  {/each}
  {#each selected as tag (tag)}
    <span
      class="inline-flex items-center gap-1 rounded-full bg-indigo-500/15 px-2 py-0.5 text-xs text-indigo-700 dark:text-indigo-300"
    >
      {tag}
      <button
        type="button"
        class="hit-area -mr-1 grid size-4 place-items-center rounded-full hover:bg-indigo-500/25 active:bg-indigo-500/35"
        aria-label="Remove {tag}"
        title="Remove {tag}"
        onclick={() => remove(tag)}>×</button
      >
    </span>
  {/each}
  <input
    class="min-w-24 flex-1 bg-transparent text-sm outline-none"
    placeholder={selected.length || fixed.length ? '' : 'Add tags'}
    aria-label="Add tag"
    bind:value={query}
    onkeydown={(event) => {
      if (event.key === 'Enter' && query.trim()) {
        event.preventDefault();
        addTyped();
      } else if (event.key === 'Backspace' && !query && selected.length) {
        remove(selected[selected.length - 1]);
      }
    }}
  />
</div>
{#if query && suggestions.length}
  <div class="mt-1 flex flex-wrap gap-1">
    {#each suggestions as tag (tag)}
      <button
        type="button"
        class="rounded-full bg-slate-900/5 px-2 py-0.5 text-xs hover:bg-slate-900/10 active:bg-slate-900/15 dark:bg-white/10 dark:hover:bg-white/15"
        onclick={() => add(tag)}>{tag}</button
      >
    {/each}
  </div>
{/if}
