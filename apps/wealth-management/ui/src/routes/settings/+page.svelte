<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import AISettings from '$lib/ai/AISettings.svelte';
  import { ApiError } from '$lib/api/client';
  import { api } from '$lib/api/endpoints';
  import type { DependencyStatus, SheetInfo } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { apiToken, entryReminder, theme, type Theme } from '$lib/state/settings.svelte';
  import { sync } from '$lib/state/sync.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { parseReminderHour } from '$lib/transactions/reminder';

  // Engine health comes from the root layout load.
  const health = $derived(page.data.health);

  let tokenDraft = $state(apiToken.value);
  let showToken = $state(false);
  let confirmFlush = $state(false);

  const tone: Record<DependencyStatus, 'green' | 'amber' | 'red' | 'neutral'> = {
    ok: 'green',
    degraded: 'amber',
    down: 'red',
    unconfigured: 'neutral',
  };

  function saveToken(event: SubmitEvent) {
    event.preventDefault();
    apiToken.value = tokenDraft.trim();
    toasts.push('success', apiToken.value ? 'API token saved in this browser.' : 'API token cleared.');
  }

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

  function setReminder(enabled: boolean) {
    entryReminder.enabled.value = enabled ? 'true' : 'false';
    toasts.info(enabled ? 'Daily entry reminder on.' : 'Daily entry reminder off.');
  }

  function setReminderHour(value: string) {
    entryReminder.hour.value = String(parseReminderHour(value));
    toasts.info(`Reminder shows after ${entryReminder.hour.value}:00.`);
  }

  async function flush() {
    confirmFlush = false;
    if (await sync.run(invalidateAll)) toasts.push('success', 'Caches flushed; data reloaded from the sheet.');
  }
</script>

<PageHeader title="Settings" />

<div class="grid gap-4 lg:grid-cols-2">
  <Card title="API token" subtitle="Required for changes when the engine has API_TOKEN set">
    <form class="space-y-3" onsubmit={saveToken}>
      <div class="flex gap-2">
        <label class="sr-only" for="api-token">API token</label>
        <input
          id="api-token"
          class="field font-mono"
          type={showToken ? 'text' : 'password'}
          autocomplete="off"
          spellcheck="false"
          bind:value={tokenDraft}
          placeholder="Paste the engine's API_TOKEN"
        />
        <button type="button" class="btn-ghost" onclick={() => (showToken = !showToken)}
          >{showToken ? 'Hide' : 'Show'}</button
        >
      </div>
      <p class="muted text-xs">
        Stored only in this browser and sent as a Bearer token on writes. Without a token, the engine accepts writes
        only when it listens on localhost.
      </p>
      <button type="submit" class="btn-primary">Save token</button>
    </form>
  </Card>

  <Card title="Appearance">
    <fieldset class="space-y-2 text-sm">
      <legend class="label">Theme</legend>
      {#each [['system', 'Follow the system'], ['light', 'Light'], ['dark', 'Dark']] as [value, label] (value)}
        <label class="flex items-center gap-2">
          <input
            type="radio"
            name="theme"
            class="accent-indigo-600"
            checked={theme.value === value}
            onchange={() => (theme.value = value as Theme)}
          />
          {label}
        </label>
      {/each}
    </fieldset>
    <p class="muted mt-4 text-xs">
      Amounts are always hidden on load and after navigation; click any amount or the eye icon to reveal.
    </p>
  </Card>

  <Card title="Connections" subtitle="Live status reported by the engine">
    {#snippet actions()}
      <button type="button" class="btn-ghost text-xs" onclick={testConnection}>Test again</button>
    {/snippet}
    <dl class="mb-3 grid grid-cols-[auto_1fr] gap-x-4 gap-y-1.5 text-sm">
      {#if sheet}
        <dt class="muted">Spreadsheet</dt>
        <dd class="min-w-0">
          <a
            class="text-indigo-600 hover:underline dark:text-indigo-400"
            href="https://docs.google.com/spreadsheets/d/{encodeURIComponent(sheet.spreadsheetId)}"
            target="_blank"
            rel="noopener noreferrer">{sheet.title || 'Untitled'}</a
          >
          <p class="muted font-mono text-xs break-all">{sheet.spreadsheetId}</p>
        </dd>
      {:else if sheetError}
        <dt class="muted">Spreadsheet</dt>
        <dd class="text-rose-600 dark:text-rose-400">{sheetError}</dd>
      {:else if sheetUnsupported}
        <dt class="muted">Spreadsheet</dt>
        <dd class="muted">This engine does not report sheet details.</dd>
      {/if}
      <dt class="muted">Last sync</dt>
      <dd>{lastSync}</dd>
    </dl>
    {#if health}
      <ul class="divide-y divide-slate-900/5 text-sm dark:divide-white/5">
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

  <AISettings />

  <Card title="Daily entry reminder" subtitle="A banner when nothing is recorded today">
    <label class="flex items-center gap-2 text-sm">
      <input
        type="checkbox"
        class="size-4 accent-indigo-600"
        checked={entryReminder.enabled.value === 'true'}
        onchange={(event) => setReminder(event.currentTarget.checked)}
      />
      Remind me to record today's transactions
    </label>
    <div class="mt-3 flex items-center gap-2 text-sm">
      <label for="reminder-hour">Show after</label>
      <input
        id="reminder-hour"
        type="number"
        min="0"
        max="23"
        class="field w-20"
        value={entryReminder.hour.value}
        disabled={entryReminder.enabled.value !== 'true'}
        onchange={(event) => setReminderHour(event.currentTarget.value)}
      />
      <span class="muted">:00 (Vietnam time)</span>
    </div>
    <p class="muted mt-3 text-xs">
      Checked when the app opens. Dismissing hides it until tomorrow. Stored in this browser; no system notifications.
    </p>
  </Card>

  <Card title="Cache" subtitle="Sheet data is cached on the engine">
    <p class="text-sm">Flushing drops every cached sheet read, so the next requests read Google Sheets directly.</p>
    <button type="button" class="btn-danger mt-4" disabled={sync.syncing} onclick={() => (confirmFlush = true)}
      >Flush caches</button
    >
  </Card>
</div>

<Modal open={confirmFlush} title="Flush caches?" onclose={() => (confirmFlush = false)}>
  <p class="text-sm">
    Every page reloads its data from Google Sheets. This is safe but slower for the next few requests.
  </p>
  {#snippet footer()}
    <button type="button" class="btn-ghost" onclick={() => (confirmFlush = false)}>Cancel</button>
    <button type="button" class="btn-danger" onclick={flush}>Flush</button>
  {/snippet}
</Modal>
