<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import { ApiError } from '$lib/api/client';
  import { api } from '$lib/api/endpoints';
  import type { DependencyStatus, SheetInfo } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import { sync } from '$lib/state/sync.svelte';

  // Engine health comes from the root layout load.
  const health = $derived(page.data.health);

  const tone: Record<DependencyStatus, 'green' | 'amber' | 'red' | 'neutral'> = {
    ok: 'green',
    degraded: 'amber',
    down: 'red',
    unconfigured: 'neutral',
  };

  // The connected spreadsheet (WM-SET-2). Engines before /sheets/info answer
  // 404; the session's own last sync is shown then.
  let sheet = $state<SheetInfo | null>(null);
  let sheetError = $state('');
  let sheetUnsupported = $state(false);

  async function loadSheet() {
    sheetError = '';
    try {
      sheet = await api.sheetInfo();
      sheetUnsupported = false;
    } catch (failure) {
      sheet = null;
      if (failure instanceof ApiError && failure.status === 404 && !failure.code) sheetUnsupported = true;
      else sheetError = failure instanceof Error ? failure.message : String(failure);
    }
  }

  $effect(() => {
    void loadSheet();
  });

  const timestamp = (value: Date) =>
    value.toLocaleString('en-GB', {
      day: '2-digit',
      month: '2-digit',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
      timeZone: 'Asia/Ho_Chi_Minh',
    });
  const lastSync = $derived.by(() => {
    if (sheet?.lastSyncAt) return `${timestamp(new Date(sheet.lastSyncAt))} (engine)`;
    if (sync.lastSynced) return `${timestamp(sync.lastSynced)} (this session)`;
    return sheet ? 'not since the engine started' : 'not yet in this session';
  });

  async function testConnection() {
    await Promise.all([invalidateAll(), loadSheet()]);
  }
</script>

<Card title="Spreadsheet">
  <dl class="grid grid-cols-[auto_1fr] gap-x-6 gap-y-2 text-sm">
    <dt class="muted">Sheet</dt>
    <dd class="min-w-0">
      {#if sheet && !sheet.spreadsheetId}
        {sheet.title || 'Untitled'} <span class="muted">(local workbook)</span>
      {:else if sheet}
        <a
          class="text-indigo-600 hover:underline dark:text-indigo-400"
          href="https://docs.google.com/spreadsheets/d/{encodeURIComponent(sheet.spreadsheetId)}"
          target="_blank"
          rel="noopener noreferrer"
          title={sheet.spreadsheetId}>{sheet.title || 'Untitled'}</a
        >
      {:else if sheetError}
        <span class="text-rose-600 dark:text-rose-400">{sheetError}</span>
      {:else if sheetUnsupported}
        <span class="muted">This engine does not report sheet details.</span>
      {:else}
        <span class="muted">Loading…</span>
      {/if}
    </dd>
    <dt class="muted">Last sync</dt>
    <dd>{lastSync}</dd>
  </dl>
</Card>

<Card title="Services" subtitle="Live status reported by the engine">
  {#snippet actions()}
    <button type="button" class="btn-ghost text-xs" onclick={testConnection}>Test again</button>
  {/snippet}
  {#if health}
    <ul class="-my-2 divide-y divide-slate-900/5 text-sm dark:divide-white/5">
      {#each health.checks as check (check.name)}
        <li class="flex items-start justify-between gap-3 py-2">
          <div class="min-w-0">
            <p class="font-medium">{check.name}</p>
            {#if check.error}<p class="muted text-xs break-words">{check.error}</p>{/if}
          </div>
          <div class="flex items-center gap-2">
            {#if check.latencyMs !== undefined}<span class="muted tabular text-xs">{check.latencyMs} ms</span>{/if}
            <Badge tone={tone[check.status]}>{check.status}</Badge>
          </div>
        </li>
      {/each}
    </ul>
  {:else}
    <p class="text-sm text-rose-600 dark:text-rose-400">The engine is not reachable.</p>
  {/if}
</Card>
