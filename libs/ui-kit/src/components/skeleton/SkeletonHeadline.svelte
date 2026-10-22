<script lang="ts">
  import Loading from './Loading.svelte';

  // Placeholder for a PageHeadline: the real question, the answer and evidence loading.
  interface Props {
    /** The page question, shown as text, e.g. "Am I within budget?". */
    question: string;
    label?: string;
    /** Adds a row of stat tiles under the answer (BudgetOverview). */
    stats?: number;
    class?: string;
  }

  let { question, label, stats = 0, class: className = '' }: Props = $props();
</script>

<section class="panel border-l-4 border-l-ink-300 p-5 dark:border-l-ink-600 {className}" aria-label={question}>
  <p class="muted text-xs font-medium uppercase">{question}</p>
  <Loading label={label ?? 'the answer'} class="mt-2">
    <div class="skeleton h-6 w-2/3 max-w-md"></div>
    <div class="skeleton mt-3 h-3.5 w-full max-w-xl"></div>
    <div class="skeleton mt-2 h-3.5 w-1/2 max-w-sm"></div>
    {#if stats}
      <div class="mt-5 grid grid-cols-2 gap-4 sm:grid-cols-4">
        {#each { length: stats }, i (i)}
          <div class="space-y-2">
            <div class="skeleton h-3 w-1/2"></div>
            <div class="skeleton h-5 w-3/4"></div>
          </div>
        {/each}
      </div>
    {/if}
  </Loading>
</section>
