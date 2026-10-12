<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import Icon from '$lib/components/Icon.svelte';
  import SheetsAuthSteps from '$lib/components/SheetsAuthSteps.svelte';

  // Errors from older loaders carry only authProblem; treat them as expired.
  const sheetsAuth = $derived(page.error?.sheetsAuth ?? (page.error?.authProblem ? 'expired' : null));
</script>

<svelte:head><title>Error · Wealth</title></svelte:head>

<div class="glass mx-auto mt-10 max-w-xl p-6 text-center">
  <Icon name="alert" size={28} class="mx-auto text-orange-500" />
  <h1 class="mt-3 text-lg font-semibold">{page.status === 404 ? 'Page not found' : 'Something went wrong'}</h1>
  <p class="muted mt-2 text-sm break-words">{page.error?.message}</p>
  {#if sheetsAuth}
    <SheetsAuthSteps kind={sheetsAuth} class="mx-auto mt-3 max-w-md" />
  {/if}
  {#if page.status !== 404}
    <button type="button" class="btn-primary mt-5" onclick={() => invalidateAll()}>Try again</button>
  {/if}
</div>
