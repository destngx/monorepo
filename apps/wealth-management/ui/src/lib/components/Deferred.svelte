<script lang="ts" generics="T">
  import type { Snippet } from 'svelte';
  import { invalidateAll } from '$app/navigation';
  import { track } from '$lib/api/deferred.svelte';
  import type { LoadFailure, Streamed } from '$lib/api/load';
  import LoadError from './LoadError.svelte';
  import Spinner from './Spinner.svelte';
  import SkeletonCard from './skeleton/SkeletonCard.svelte';

  // A page section that streams in (docs/wealth-management/AGENTS.md "Loading
  // and feedback"): the skeleton until `data` settles, then the children with
  // the value, or the error with a Retry in the section's place. A newer
  // promise (sync, invalidate, retry) keeps the value on screen with a small
  // "Refreshing" badge; a failed refresh keeps it too, under an error strip.
  interface Props {
    /** A promise from `defer()`, usually straight from the page data. */
    data: Streamed<T> | null | undefined;
    /** What loads, for screen readers and errors: "net worth" → "Loading net worth…". */
    label: string;
    /** When this changes (a month, a code), the old value is dropped instead of kept. */
    scope?: unknown;
    /** Re-runs the request; defaults to invalidateAll (re-runs the page load). */
    retry?: () => unknown;
    children: Snippet<[T]>;
    /** Defaults to a SkeletonCard. */
    skeleton?: Snippet;
    /** Replaces the default LoadError. */
    error?: Snippet<[LoadFailure, () => unknown]>;
    class?: string;
  }

  let { data, label, scope, retry, children, skeleton, error, class: className = '' }: Props = $props();

  const section = track(() => data, { scope: () => scope });
  // The wrapper is a one-column grid so the section fills its slot in the page's grid.
  const retryLoad = () => (retry ? retry() : invalidateAll());
</script>

<div
  class="relative grid grid-cols-[minmax(0,1fr)] {section.latest && section.failure
    ? 'grid-rows-[auto_1fr]'
    : ''} {className}"
  aria-busy={section.pending}
  data-state={section.loading ? 'loading' : section.latest ? 'ready' : 'error'}
>
  {#if section.latest}
    {#if section.refreshing}
      <span
        class="pointer-events-none absolute -top-2 right-3 z-10 inline-flex items-center gap-1 rounded-full border border-slate-900/10 bg-white/90 px-2 py-0.5 text-[0.7rem] font-medium text-slate-600 shadow-sm dark:border-white/10 dark:bg-slate-800/90 dark:text-slate-300"
        data-refreshing
      >
        <Spinner size={11} />Refreshing<span class="sr-only"> {label}</span>
      </span>
    {/if}
    {#if section.failure}
      <LoadError failure={section.failure} {label} retry={retryLoad} compact class="mb-2" />
    {/if}
    {@render children(section.latest.value)}
  {:else if section.loading || !section.failure}
    {#if skeleton}{@render skeleton()}{:else}<SkeletonCard {label} />{/if}
  {:else if error}
    {@render error(section.failure, retryLoad)}
  {:else}
    <LoadError failure={section.failure} {label} retry={retryLoad} />
  {/if}
</div>
