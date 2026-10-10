<script lang="ts">
  import { onMount } from 'svelte';
  import { createThemeController } from './theme.svelte';

  let theme = $state<ReturnType<typeof createThemeController>>();

  onMount(() => {
    theme = createThemeController();
  });

  const isDark = $derived(theme?.current === 'dark');
</script>

<button
  type="button"
  class="panel grid h-10 w-10 place-items-center !shadow-ink-sm transition-transform hover:-rotate-12 active:scale-90"
  aria-label={isDark ? 'Switch to light theme' : 'Switch to dark theme'}
  data-cursor="flip"
  onclick={() => theme?.toggle()}
>
  {#if isDark}
    <!-- moon -->
    <svg viewBox="0 0 24 24" class="h-5 w-5" fill="currentColor" aria-hidden="true">
      <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z" />
    </svg>
  {:else}
    <!-- sun -->
    <svg viewBox="0 0 24 24" class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="2.5" aria-hidden="true">
      <circle cx="12" cy="12" r="4.5" fill="currentColor" />
      <path d="M12 2v2.5M12 19.5V22M4.2 4.2l1.8 1.8M18 18l1.8 1.8M2 12h2.5M19.5 12H22M4.2 19.8 6 18M18 6l1.8-1.8" />
    </svg>
  {/if}
</button>
