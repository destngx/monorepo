<script lang="ts">
  import { invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import AISettings from '$lib/ai/AISettings.svelte';
  import type { DependencyStatus } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { apiToken, theme, type Theme } from '$lib/state/settings.svelte';
  import { sync } from '$lib/state/sync.svelte';
  import { toasts } from '$lib/state/toasts.svelte';

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
      <button type="button" class="btn-ghost text-xs" onclick={() => invalidateAll()}>Test again</button>
    {/snippet}
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
    <p class="muted mt-3 text-xs">
      Last sync: {sync.lastSynced
        ? sync.lastSynced.toLocaleString('en-GB', { timeZone: 'Asia/Ho_Chi_Minh' })
        : 'not yet in this session'}
    </p>
  </Card>

  <AISettings />

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
