<script lang="ts">
  import Loading from './Loading.svelte';

  // Placeholder for a Card: the title (real when given, so the page reads
  // right while loading), then text lines, a chart block or table rows.
  interface Props {
    /** The real card title; shown as text, not a placeholder. */
    title?: string;
    label?: string;
    lines?: number;
    /** Chart height in px; adds a chart block. */
    chart?: number;
    /** Adds that many table rows. */
    rows?: number;
    class?: string;
  }

  let { title, label, lines = 3, chart, rows, class: className = '' }: Props = $props();
  // "Gold and USD" reads "Loading gold and USD…"; acronyms keep their case.
  const spoken = $derived(label ?? (title && !/^.[A-Z]/.test(title) ? title[0].toLowerCase() + title.slice(1) : title));
</script>

<section class="glass p-5 {className}" aria-label={title}>
  {#if title}
    <h2 class="mb-4 text-sm font-semibold tracking-wide text-slate-700 dark:text-slate-200">{title}</h2>
  {/if}
  <Loading label={spoken}>
    {#if !title}<div class="skeleton mb-5 h-4 w-1/3"></div>{/if}
    <div class="space-y-2">
      {#each { length: lines }, i (i)}
        <div class="skeleton h-3.5 {i === 0 ? 'h-7 w-1/2' : i === lines - 1 ? 'w-3/5' : 'w-full'}"></div>
      {/each}
    </div>
    {#if chart}
      <div class="mt-4 flex items-end gap-2" style="height: {chart}px">
        {#each [45, 70, 55, 85, 60, 75, 50, 90] as bar, i (i)}
          <div class="skeleton flex-1 rounded-b-none" style="height: {bar}%"></div>
        {/each}
      </div>
    {/if}
    {#if rows}
      <div class="mt-4 divide-y divide-slate-900/5 dark:divide-white/5">
        {#each { length: rows }, r (r)}
          <div class="flex gap-4 py-2.5">
            <div class="skeleton h-3.5 flex-1"></div>
            <div class="skeleton h-3.5 w-20"></div>
          </div>
        {/each}
      </div>
    {/if}
  </Loading>
</section>
