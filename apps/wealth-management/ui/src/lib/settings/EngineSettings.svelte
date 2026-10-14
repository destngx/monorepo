<script lang="ts">
  import { goto, invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import { api } from '$lib/api/endpoints';
  import type { EngineSetting, EngineSettings, GoogleAuthPurpose } from '$lib/api/types';
  import Badge from '$lib/components/Badge.svelte';
  import Card from '$lib/components/Card.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { groupSettings, initialDrafts, pendingChanges, waitForRestart, type Drafts } from './form';
  import GoogleConnect from './GoogleConnect.svelte';
  import { googleAuthNotice, googleAuthParams } from './googleAuth';

  // Engine settings stored by the engine (they replace the env file). Saved
  // changes apply after a restart. Secrets are write-only: the engine only
  // says whether they are set.
  let view = $state<EngineSettings | null>(null);
  let drafts = $state<Drafts>({});
  let loadError = $state('');
  let saveError = $state('');
  let saving = $state(false);
  let restarting = $state(false);

  const groups = $derived(view ? groupSettings(view) : []);
  const changes = $derived(view ? pendingChanges(view.settings, drafts) : {});
  const changeCount = $derived(Object.keys(changes).length);

  function show(next: EngineSettings) {
    view = next;
    drafts = initialDrafts(next.settings);
    saveError = '';
  }

  async function load() {
    loadError = '';
    try {
      show(await api.settings());
    } catch (failure) {
      loadError = failure instanceof Error ? failure.message : String(failure);
    }
  }

  $effect(() => {
    void load();
  });

  // The engine's Google callback redirects here with the outcome; report it
  // once and drop it from the address so a reload does not repeat it.
  onMount(() => {
    const notice = googleAuthNotice(page.url.searchParams);
    if (!notice) return;
    toasts.push(notice.kind, notice.message);
    const url = new URL(page.url);
    for (const key of googleAuthParams) url.searchParams.delete(key);
    void goto(url, { replaceState: true, noScroll: true, keepFocus: true });
  });

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (changeCount === 0) return;
    saving = true;
    saveError = '';
    try {
      show(await api.updateSettings(changes));
      toasts.push('success', 'Settings saved. Restart the engine to apply them.');
    } catch (failure) {
      saveError = failure instanceof Error ? failure.message : String(failure);
    } finally {
      saving = false;
    }
  }

  async function restart() {
    restarting = true;
    try {
      await api.restartEngine();
      show(await waitForRestart(() => api.settings()));
      await invalidateAll();
      toasts.push('success', 'Engine restarted with the saved settings.');
    } catch (failure) {
      toasts.error(failure, 'Restart');
    } finally {
      restarting = false;
    }
  }

  // Sign-ins save a token, which would overwrite unsaved edits on reload.
  const connects: Record<string, { purpose: GoogleAuthPurpose; label: string }> = {
    sheets: { purpose: 'sheets', label: 'Connect Google account' },
    testing: { purpose: 'test', label: 'Connect test account' },
  };

  const placeholder = (setting: EngineSetting) => {
    if (setting.secret) return setting.isSet ? 'Stored; type to replace' : 'Not set';
    return setting.default ? `Default: ${setting.default}` : 'Not set';
  };
</script>

<section class="mt-6 space-y-4" aria-labelledby="engine-settings">
  <div class="flex flex-wrap items-start justify-between gap-3">
    <div>
      <h2 id="engine-settings" class="text-base font-semibold">Engine settings</h2>
      <p class="muted text-xs">
        Credentials and integrations, stored by the engine{#if view}
          in <code class="font-mono break-all">{view.path}</code>{/if}. Secrets are never shown again after saving.
      </p>
    </div>
    <div class="flex items-center gap-2">
      {#if view?.restartRequired}<Badge tone="amber">Restart to apply</Badge>{/if}
      <button
        type="button"
        class={view?.restartRequired ? 'btn-primary' : 'btn-ghost'}
        disabled={!view || restarting || saving || changeCount > 0}
        title={changeCount > 0 ? 'Save or discard your changes first' : undefined}
        onclick={restart}>{restarting ? 'Restarting...' : 'Restart engine'}</button
      >
    </div>
  </div>

  {#if loadError}
    <Card>
      <p class="text-sm text-rose-600 dark:text-rose-400">{loadError}</p>
      <button type="button" class="btn-ghost mt-3" onclick={load}>Try again</button>
    </Card>
  {:else if view}
    <form class="space-y-4" onsubmit={save}>
      <div class="grid gap-4 lg:grid-cols-2">
        {#each groups as { group, settings } (group.id)}
          <Card title={group.label} subtitle={group.description}>
            {#snippet actions()}
              {#if connects[group.id]}
                <GoogleConnect
                  {...connects[group.id]}
                  disabled={changeCount > 0 || saving}
                  disabledReason="Save or discard your changes first"
                  onchange={load}
                />
              {/if}
            {/snippet}
            <div class="space-y-4">
              {#each settings as setting (setting.key)}
                {@const id = `setting-${setting.key}`}
                {@const draft = drafts[setting.key]}
                <div>
                  <div class="flex items-center justify-between gap-2">
                    <label class="label" for={id}>{setting.label}</label>
                    {#if setting.secret}
                      <Badge tone={draft.clear ? 'amber' : setting.isSet ? 'green' : 'neutral'}
                        >{draft.clear ? 'will be cleared' : setting.isSet ? 'set' : 'not set'}</Badge
                      >
                    {/if}
                  </div>
                  <div class="mt-1 flex gap-2">
                    {#if setting.kind === 'choice'}
                      <select {id} class="field" bind:value={draft.value}>
                        <option value="">Default{setting.default ? ` (${setting.default})` : ''}</option>
                        {#each setting.options ?? [] as option (option)}
                          <option value={option}>{option}</option>
                        {/each}
                      </select>
                    {:else}
                      <input
                        {id}
                        class="field font-mono"
                        type={setting.secret ? 'password' : setting.kind === 'integer' ? 'number' : 'text'}
                        min={setting.kind === 'integer' ? 1 : undefined}
                        inputmode={setting.kind === 'url' ? 'url' : undefined}
                        autocomplete="off"
                        spellcheck="false"
                        disabled={draft.clear}
                        placeholder={placeholder(setting)}
                        bind:value={draft.value}
                      />
                    {/if}
                    {#if setting.secret && setting.isSet}
                      <button
                        type="button"
                        class="btn-ghost"
                        onclick={() => (drafts[setting.key] = { value: '', clear: !draft.clear })}
                        >{draft.clear ? 'Keep' : 'Clear'}</button
                      >
                    {/if}
                  </div>
                  <p class="muted mt-1 text-xs">
                    <code class="font-mono">{setting.key}</code>{#if setting.description}. {setting.description}{/if}
                  </p>
                </div>
              {/each}
            </div>
          </Card>
        {/each}
      </div>

      <div class="glass sticky bottom-4 flex flex-wrap items-center justify-end gap-3 p-3">
        {#if saveError}<p class="mr-auto text-sm text-rose-600 dark:text-rose-400" role="alert">{saveError}</p>{/if}
        <span class="muted text-xs"
          >{changeCount === 0 ? 'No changes' : `${changeCount} change${changeCount === 1 ? '' : 's'}`}</span
        >
        <button
          type="button"
          class="btn-ghost"
          disabled={changeCount === 0 || saving}
          onclick={() => view && show(view)}>Discard</button
        >
        <button type="submit" class="btn-primary" disabled={changeCount === 0 || saving}
          >{saving ? 'Saving...' : 'Save settings'}</button
        >
      </div>
    </form>
  {:else}
    <p class="muted text-sm">Loading engine settings...</p>
  {/if}
</section>
