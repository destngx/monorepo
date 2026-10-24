<script lang="ts">
  import '@fontsource-variable/inter';
  import '@fontsource-variable/playfair-display';
  import '@fontsource-variable/jetbrains-mono';
  import './layout.css';
  import { navigating } from '$app/state';
  import { Aurora, NavProgress, Toasts } from '@libs/ui-kit';
  import { live } from '$lib/state/live.svelte';
  import { theme } from '$lib/state/theme.svelte';

  let { children } = $props();

  // With the system theme, the page follows a later switch between light and dark.
  $effect(() => theme.watchSystem());
  // Every page shows live agent states.
  $effect(() => live.watch());
</script>

<svelte:head><title>Agents</title></svelte:head>

<Aurora />
<NavProgress active={!!navigating.to} />

{@render children()}

<Toasts />
