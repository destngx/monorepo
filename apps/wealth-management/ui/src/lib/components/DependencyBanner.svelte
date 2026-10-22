<script lang="ts">
  import { base } from '$app/paths';
  import { sheetsAuthKind } from '$lib/api/client';
  import type { EngineHealth } from '$lib/api/types';
  import SheetsAuthSteps from './SheetsAuthSteps.svelte';

  // Warns when the engine reports a dependency that is down or not configured
  // (WM-OPS-6); without the sheet, most pages have no data.
  let { health }: { health: EngineHealth | null } = $props();

  // Only the ledger dependencies earn a banner: the sheet, and a cache that is
  // configured but failing. An unconfigured cache is a supported setup, and
  // market providers are optional per page (their status is in Settings).
  const problems = $derived(
    health?.checks.filter(
      (check) =>
        (check.name === 'sheets' && check.status !== 'ok') || (check.name === 'cache' && check.status === 'down'),
    ) ?? [],
  );
  const sheetsCheck = $derived(problems.find((check) => check.name === 'sheets'));
  const sheetsMissing = $derived(sheetsCheck !== undefined);
  // Setup steps when unconfigured, re-auth steps when Google rejects the token (WM-UI-4).
  const sheetsAuth = $derived(
    sheetsCheck?.status === 'unconfigured' ? 'missing' : sheetsAuthKind(undefined, sheetsCheck?.error ?? ''),
  );
</script>

{#if health === null}
  <div role="alert" class="panel mb-6 border border-rose-500/40 p-4 text-sm">
    <p class="font-semibold text-rose-700 dark:text-rose-300">The engine is not reachable</p>
    <p class="muted mt-1">Start it with <code>pnpx nx run wealth-management:serve</code> and reload.</p>
  </div>
{:else if problems.length}
  <div role="status" class="panel mb-6 border border-amber-500/40 p-4 text-sm">
    <p class="font-semibold text-amber-700 dark:text-amber-300">
      {sheetsMissing ? 'Google Sheets is unavailable, so ledger data cannot load.' : 'Some services are degraded.'}
    </p>
    <ul class="muted mt-1 space-y-0.5">
      {#each problems as check (check.name)}
        <li><span class="font-medium">{check.name}</span>: {check.status}{check.error ? ` - ${check.error}` : ''}</li>
      {/each}
    </ul>
    {#if sheetsAuth}<SheetsAuthSteps kind={sheetsAuth} class="mt-2" />{/if}
    <a class="mt-2 inline-block text-accent-600 hover:underline dark:text-accent-400" href="{base}/settings"
      >Open settings</a
    >
  </div>
{/if}
