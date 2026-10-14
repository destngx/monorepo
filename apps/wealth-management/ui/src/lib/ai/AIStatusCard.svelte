<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import type { AIStatus } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import AIUsage from './AIUsage.svelte';

  // What the running engine uses for AI (WM-SET-1) and the provider quota
  // (WM-SET-4). Models are edited with the other AI settings below.
  let status = $state<AIStatus | null>(null);
  let error = $state('');

  async function load() {
    error = '';
    try {
      status = await api.aiStatus();
    } catch (failure) {
      status = null;
      error = failure instanceof Error ? failure.message : String(failure);
    }
  }

  $effect(() => {
    void load();
  });
</script>

<Card title="Status" subtitle="As the running engine sees it">
  {#snippet actions()}
    <button type="button" class="btn-ghost text-xs" onclick={load}>Refresh</button>
  {/snippet}
  {#if status}
    <dl class="grid grid-cols-[auto_1fr] items-center gap-x-6 gap-y-2 text-sm">
      <dt class="muted">AI</dt>
      <dd>
        <Badge tone={status.configured ? 'green' : 'neutral'}
          >{status.configured ? 'configured' : 'not configured'}</Badge
        >
      </dd>
      {#if status.endpoint}
        <dt class="muted">Endpoint</dt>
        <dd class="font-mono text-xs break-all">{status.endpoint}{status.provider ? ` (${status.provider})` : ''}</dd>
      {/if}
      <dt class="muted">Prompts</dt>
      <dd>
        {#if status.promptsError}
          <span class="text-rose-600 dark:text-rose-400">{status.promptsError}</span>
        {:else if status.missingPrompts.length}
          <details class="text-xs">
            <summary class="cursor-pointer text-amber-700 dark:text-amber-300"
              >{status.missingPrompts.length} missing from the Prompts sheet</summary
            >
            <ul class="mt-1 font-mono">
              {#each status.missingPrompts as key (key)}<li>{key}</li>{/each}
            </ul>
          </details>
        {:else}
          {status.prompts} loaded
        {/if}
      </dd>
    </dl>
  {:else if error}
    <p class="text-sm text-rose-600 dark:text-rose-400">AI status unavailable: {error}</p>
  {:else}
    <p class="muted text-sm">Loading…</p>
  {/if}

  <AIUsage />
</Card>
