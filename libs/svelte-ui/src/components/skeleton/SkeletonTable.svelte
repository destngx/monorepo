<script lang="ts">
  import Loading from './Loading.svelte';

  // Placeholder for a table: a header row and `rows` rows of `columns` cells.
  interface Props {
    rows?: number;
    columns?: number;
    label?: string;
    class?: string;
  }

  let { rows = 5, columns = 4, label, class: className = '' }: Props = $props();
</script>

<Loading {label} class={className}>
  <div class="divide-y divide-ink-900/5 dark:divide-ink-50/5">
    <div class="flex gap-4 pb-2">
      {#each { length: columns }, c (c)}<div class="skeleton h-3 flex-1 opacity-60"></div>{/each}
    </div>
    {#each { length: rows }, r (r)}
      <div class="flex gap-4 py-2.5">
        {#each { length: columns }, c (c)}
          <div class="skeleton h-3.5 flex-1 {c === 0 ? '' : 'max-w-24'}"></div>
        {/each}
      </div>
    {/each}
  </div>
</Loading>
