<script lang="ts">
  import { untrack } from 'svelte';
  import { NAV_PROGRESS_DELAY_MS, NAV_PROGRESS_FADE_MS } from './navProgress';

  // The top bar while a page loads (docs/ui/README.md, "Loading"): after a short delay it creeps towards the end, slowing as it goes,
  // then fills and fades once the page is there. The bar is always in the DOM so
  // each phase transitions from wherever the last one left it.
  let { active }: { active: boolean } = $props();

  let phase = $state<'idle' | 'loading' | 'done'>('idle');

  $effect(() => {
    if (active) {
      // A new navigation during the fade restarts from empty.
      if (untrack(() => phase) === 'done') phase = 'idle';
      const timer = setTimeout(() => (phase = 'loading'), NAV_PROGRESS_DELAY_MS);
      return () => clearTimeout(timer);
    }
    if (untrack(() => phase) !== 'loading') return;
    phase = 'done';
    const timer = setTimeout(() => (phase = 'idle'), NAV_PROGRESS_FADE_MS);
    return () => clearTimeout(timer);
  });
</script>

<div
  class="pointer-events-none fixed inset-x-0 top-0 z-50 h-[3px]"
  data-nav-progress
  data-phase={phase}
  role={phase === 'loading' ? 'progressbar' : undefined}
  aria-label={phase === 'loading' ? 'Loading page' : undefined}
  aria-hidden={phase === 'loading' ? undefined : 'true'}
>
  <div class="bar h-full origin-left bg-accent-600 dark:bg-accent-300"></div>
</div>

<style>
  .bar {
    opacity: 0;
    transform: scaleX(0);
  }
  [data-phase='loading'] .bar {
    opacity: 1;
    transform: scaleX(0.9);
    /* Fast at first, then barely moving: a long load still looks alive without claiming to be done. */
    transition: transform 10s cubic-bezier(0.05, 0.7, 0.1, 1);
  }
  [data-phase='done'] .bar {
    opacity: 0;
    transform: scaleX(1);
    transition:
      transform 200ms ease-out,
      opacity 200ms ease-out 200ms;
  }
</style>
