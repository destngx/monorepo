<script lang="ts">
  import { onMount } from 'svelte';
  import { api } from '$lib/api/endpoints';
  import type { ApiTokenStatus } from '$lib/api/types';
  import { Button, Card } from '@libs/ui-kit';
  import { apiToken } from '$lib/state/settings.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { apiTokenProblem, generateApiToken } from './apiToken';

  // The engine keeps the token (WM-SEC-1) and never returns it; this browser
  // keeps its own copy to send with every change.
  let status = $state<ApiTokenStatus | null>(null);
  let statusError = $state('');
  let draft = $state(apiToken.value);
  let show = $state(false);

  const token = $derived(draft.trim());
  const problem = $derived(token ? apiTokenProblem(token) : '');
  // With no token on a non-loopback bind the engine refuses every write,
  // this one included.
  const locked = $derived(status !== null && !status.set && !status.loopback);

  async function loadStatus() {
    statusError = '';
    try {
      status = await api.apiTokenStatus();
    } catch (failure) {
      statusError = failure instanceof Error ? failure.message : String(failure);
    }
  }

  onMount(() => {
    void loadStatus();
  });

  function generate() {
    draft = generateApiToken();
    show = true;
  }

  async function setOnEngine() {
    status = await api.setApiToken(token);
    // Only now: the request itself had to carry the old token.
    apiToken.value = token;
  }

  function saveInBrowser() {
    apiToken.value = token;
    toasts.success(token ? 'API token saved in this browser.' : 'API token cleared from this browser.');
  }

  async function copy() {
    if (!navigator.clipboard) {
      // Plain http on another device is not a secure context: no clipboard.
      show = true;
      toasts.info('Select the token and copy it.');
      return;
    }
    await navigator.clipboard.writeText(token);
    toasts.success('API token copied.');
  }
</script>

<Card title="API token" subtitle="Every change to the engine needs it, from this browser and from the iPhone Shortcut">
  <p class="mb-3 text-sm" role="status">
    {#if statusError}
      <span class="text-rose-700 dark:text-rose-300">Could not check the engine: {statusError}</span>
      <button type="button" class="link" onclick={loadStatus}>Retry</button>
    {:else if !status}
      <span class="muted">Checking the engine…</span>
    {:else if status.set}
      The engine requires a token for changes. Setting a new one replaces it at once; other devices then need the new
      one.
    {:else if status.loopback}
      No token yet: changes only work from this Mac. Set one before opening the engine on the tailnet.
    {:else}
      No token, so the engine refuses every change. Start it on localhost (without <code>HOST</code>), open
      <code>http://localhost:9090/ui</code> and set a token there.
    {/if}
  </p>

  <div class="flex flex-wrap items-center gap-2">
    <label class="sr-only" for="api-token">API token</label>
    <input
      id="api-token"
      class="field min-w-0 flex-1 font-mono"
      type={show ? 'text' : 'password'}
      autocomplete="off"
      spellcheck="false"
      placeholder="Not set in this browser"
      aria-invalid={problem ? 'true' : undefined}
      aria-describedby={problem ? 'api-token-problem' : undefined}
      bind:value={draft}
    />
    <button
      type="button"
      class="btn-ghost px-2"
      aria-pressed={show}
      title={show ? 'Hide the token' : 'Show the token'}
      onclick={() => (show = !show)}>{show ? 'Hide' : 'Show'}</button
    >
    <Button size="sm" onclick={generate} title="Fill in a new random token">Generate</Button>
    <Button size="sm" onclick={copy} disabled={!token} title="Copy the token, e.g. for the iPhone Shortcut">Copy</Button
    >
  </div>
  {#if problem}<p id="api-token-problem" class="mt-1.5 text-xs text-rose-700 dark:text-rose-300">{problem}</p>{/if}

  <div class="mt-3 flex flex-wrap justify-end gap-2">
    <Button
      size="sm"
      disabled={token === apiToken.value}
      title={token === apiToken.value ? 'No changes to save' : 'Use this token from this browser only'}
      onclick={saveInBrowser}>Save in this browser</Button
    >
    <Button
      size="sm"
      variant="primary"
      disabled={!status || locked || !token || !!problem}
      title={locked ? 'Open the engine on localhost to set the first token' : undefined}
      success="API token set on the engine and saved in this browser."
      errorContext="Could not set the API token"
      onclick={setOnEngine}>Set on engine</Button
    >
  </div>
</Card>
