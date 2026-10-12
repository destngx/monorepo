<script lang="ts">
  import { onMount } from 'svelte';

  let healthStatus = $state<string | null>(null);
  let isLoading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      const response = await fetch('/api/health');
      if (!response.ok) {
        throw new Error(`Health check failed: ${response.status}`);
      }
      const data = await response.json();
      healthStatus = JSON.stringify(data, null, 2);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Unknown error';
    } finally {
      isLoading = false;
    }
  });
</script>

<div class="min-h-screen bg-gradient-to-br from-blue-500/30 to-purple-500/30 p-8 backdrop-blur-lg">
  <div class="mx-auto max-w-2xl">
    <h1 class="mb-8 text-4xl font-bold text-white">Wealth Management Dashboard</h1>

    <div class="mb-6 rounded-lg border border-white/20 bg-white/10 p-6 backdrop-blur-md">
      <h2 class="mb-4 text-xl font-semibold text-white">Go Backend Health Check</h2>

      {#if isLoading}
        <p class="text-gray-200">Checking backend health...</p>
      {:else if error}
        <p class="text-red-400">Error: {error}</p>
      {:else if healthStatus}
        <pre class="overflow-auto rounded bg-black/30 p-4 text-sm text-green-400">
{healthStatus}
				</pre>
      {/if}
    </div>

    <p class="text-gray-200">
      Visit <a href="https://svelte.dev/docs/kit" class="text-blue-300 hover:text-blue-100">svelte.dev/docs/kit</a> to read
      the documentation
    </p>
  </div>
</div>

<style global>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family:
      -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen-Sans, Ubuntu, Cantarell, 'Helvetica Neue',
      sans-serif;
  }
</style>
