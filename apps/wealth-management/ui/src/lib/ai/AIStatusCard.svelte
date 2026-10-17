<script lang="ts">
  import { track } from '$lib/api/deferred.svelte';
  import Badge from '$lib/components/Badge.svelte';
  import Button from '$lib/components/Button.svelte';
  import Card from '$lib/components/Card.svelte';
  import Deferred from '$lib/components/Deferred.svelte';
  import SkeletonText from '$lib/components/skeleton/SkeletonText.svelte';
  import AIUsage from './AIUsage.svelte';
  import { loadAIStatus } from './aiStatus';

  // What the running engine uses for AI (WM-SET-1) and the provider quota
  // (WM-SET-4). Models are edited with the other AI settings below. The
  // status streams in; Refresh keeps the shown one until the new one answers.
  let request = $state.raw(loadAIStatus());
  const status = track(() => request);

  async function refresh() {
    request = loadAIStatus();
    const result = await request;
    if (!result.ok) throw new Error(result.error);
  }
</script>

<Card title="Status" subtitle="As the running engine sees it">
  {#snippet actions()}
    <Button
      size="sm"
      icon="refresh"
      busy={status.refreshing}
      busyText="Refreshing…"
      errorContext="Could not refresh the AI status"
      onclick={refresh}>Refresh</Button
    >
  {/snippet}
  <Deferred data={request} label="the AI status" retry={() => (request = loadAIStatus())}>
    {#snippet children(status)}
      {#if !status}
        <p class="muted text-sm">AI is not available on this engine.</p>
      {:else}
        <dl class="grid grid-cols-[auto_1fr] items-center gap-x-6 gap-y-2 text-sm">
          <dt class="muted">AI</dt>
          <dd>
            <Badge status={status.configured ? 'good' : 'neutral'}
              >{status.configured ? 'configured' : 'not configured'}</Badge
            >
          </dd>
          {#if status.endpoint}
            <dt class="muted">Endpoint</dt>
            <dd class="font-mono text-xs break-all">
              {status.endpoint}{status.provider ? ` (${status.provider})` : ''}
            </dd>
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
      {/if}
    {/snippet}
    {#snippet skeleton()}<SkeletonText lines={3} label="the AI status" />{/snippet}
  </Deferred>

  <AIUsage />
</Card>
