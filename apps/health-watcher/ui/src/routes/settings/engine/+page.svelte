<script lang="ts">
  import { untrack } from 'svelte';
  import { invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { engine } from '$lib/api/endpoints';
  import type { AICheck, EngineSettingsView, User } from '$lib/api/types';
  import { Button, Card, Field, PageTitle, action, toasts } from '@libs/ui-kit';
  import { engineForm, isDirty, rejectedNote, restartNote, settingsInput } from '$lib/engine';
  import { formatDate } from '$lib/format';
  import { session } from '$lib/state/session.svelte';

  let { data } = $props();

  // Follow each load, and take the server's answer after a save.
  let view = $derived<EngineSettingsView>(data.view);
  let users = $derived<User[]>(data.users);
  // The form is deep state for bind:value. It follows a newer saved view
  // only while it has no unsaved edits, so a reload never drops them.
  // svelte-ignore state_referenced_locally
  let form = $state(engineForm(data.view));
  // svelte-ignore state_referenced_locally
  let shown = data.view;
  $effect.pre(() => {
    const next = view;
    untrack(() => {
      if (!isDirty(form, shown)) form = engineForm(next);
      shown = next;
    });
  });
  let check = $state<AICheck | null>(null);

  const dirty = $derived(isDirty(form, view));
  const restart = $derived(restartNote(view.restartFields));
  const rejected = $derived(rejectedNote(view.rejected));
  const me = $derived(session.me?.user.id);

  const save = action(
    async () => {
      const saved = await engine.save(settingsInput(form));
      form = engineForm(saved);
      view = saved;
      check = null;
    },
    { errorContext: 'Could not save', success: 'Saved' },
  );

  async function checkAI() {
    check = await engine.checkAI();
  }

  async function setAdmin(u: User, input: HTMLInputElement) {
    try {
      const updated = await engine.setAdmin(u.id, input.checked);
      users = users.map((x) => (x.id === updated.id ? updated : x));
      if (u.id === me && !updated.admin) await invalidateAll();
    } catch (error) {
      input.checked = u.admin;
      toasts.error(error, 'Could not change admin');
    }
  }

  function submit(e: SubmitEvent) {
    e.preventDefault();
    void save.run();
  }
</script>

<PageTitle title="Engine settings" subtitle="For every family on this Mac. Only engine admins see this page." />

<p class="mb-4 text-sm"><a class="link" href="{base}/settings">← Settings</a></p>

{#if rejected}
  <p class="panel mb-5 border-danger-500/40 p-4 text-sm" role="alert">{rejected}</p>
{/if}

{#if restart}
  <p class="panel mb-5 border-warn-500/40 p-4 text-sm" role="status">{restart} Listening on {view.listening} now.</p>
{/if}

<form onsubmit={submit} class="grid gap-5 lg:grid-cols-2">
  <Card
    title="AI reading"
    subtitle="Masked pages go here after the Mac reads them. Leave the address empty to turn AI reading off."
  >
    <div class="space-y-3">
      <Field
        id="ai-url"
        label="Address"
        hint="An OpenAI Responses API base, e.g. http://localhost:8080/v1 for ai-gateway."
      >
        <input
          id="ai-url"
          class="field"
          type="url"
          bind:value={form.aiBaseUrl}
          placeholder="http://localhost:8080/v1"
          autocomplete="off"
        />
      </Field>
      <Field
        id="ai-key"
        label="API key"
        hint={view.aiApiKeySet ? 'A key is saved. Type a new one to replace it.' : 'Only if the provider needs one.'}
      >
        <input
          id="ai-key"
          class="field"
          type="password"
          bind:value={form.aiApiKey}
          placeholder={view.aiApiKeySet ? 'Saved' : 'None'}
          autocomplete="off"
          disabled={form.shouldClearKey}
        />
      </Field>
      {#if view.aiApiKeySet}
        <label class="flex items-center gap-2 text-sm">
          <input type="checkbox" bind:checked={form.shouldClearKey} /> Remove the saved key
        </label>
      {/if}
      <Field id="ai-timeout" label="Time limit per reading (seconds)">
        <input
          id="ai-timeout"
          class="field w-32"
          type="number"
          min="5"
          max="1800"
          bind:value={form.aiTimeoutSeconds}
          required
        />
      </Field>
      <p class="muted text-xs">Model: {view.aiModel} at {view.aiEffort} effort. It is fixed, not a setting.</p>
      <div class="flex flex-wrap items-center gap-2">
        <Button
          size="sm"
          onclick={checkAI}
          errorContext="Could not check"
          disabled={!view.settings.aiBaseUrl || dirty}
          title={dirty ? 'Save first; the check uses the saved settings' : undefined}>Check connection</Button
        >
        {#if check}
          <span
            class="text-xs {check.ok ? 'text-accent-700 dark:text-accent-300' : 'text-danger-600 dark:text-danger-400'}"
            >{check.message}</span
          >
        {/if}
      </div>
    </div>
  </Card>

  <Card title="Reading on this Mac">
    <div class="space-y-3">
      <Field
        id="mac-ocr"
        label="mac-ocr program"
        hint={form.macOcrBin ? 'A full path ending in /mac-ocr.' : `Empty uses ${view.defaultMacOcrBin}.`}
      >
        <input
          id="mac-ocr"
          class="field font-mono text-xs"
          bind:value={form.macOcrBin}
          placeholder={view.defaultMacOcrBin}
          autocomplete="off"
          spellcheck="false"
        />
      </Field>
      {#if !view.macOcrFound}
        <p class="text-xs text-danger-600 dark:text-danger-400">
          The mac-ocr program is missing. Build it with <code>pnpm exec nx build mac-ocr</code>.
        </p>
      {/if}
      <Field id="workers" label="Background jobs at once" hint="1 to 8. OCR and AI readings share them.">
        <input id="workers" class="field w-24" type="number" min="1" max="8" bind:value={form.workers} required />
      </Field>
    </div>
  </Card>

  <Card title="Uploads">
    <div class="grid gap-3 sm:grid-cols-2">
      <Field id="max-file" label="Largest file (MB)">
        <input id="max-file" class="field" type="number" min="1" max="1024" bind:value={form.maxFileMb} required />
      </Field>
      <Field id="max-batch" label="Largest upload (MB)" hint="Raising it applies after a restart.">
        <input id="max-batch" class="field" type="number" min="1" max="4096" bind:value={form.maxBatchMb} required />
      </Field>
    </div>
  </Card>

  <Card title="Network" subtitle="Listening on {view.listening}. Changes apply after a restart.">
    <div class="grid gap-3 sm:grid-cols-[1fr_8rem]">
      <Field
        id="host"
        label="Host"
        hint="127.0.0.1, or the Mac's Tailscale address to reach it from a phone. The engine always answers on 127.0.0.1 too."
      >
        <input id="host" class="field font-mono text-xs" bind:value={form.host} autocomplete="off" required />
      </Field>
      <Field id="port" label="Port">
        <input id="port" class="field" type="number" min="1024" max="65535" bind:value={form.port} required />
      </Field>
    </div>
    <p class="muted mt-2 text-xs">
      If the engine cannot listen on the saved address for 30 seconds, it starts on 127.0.0.1:9191 so this page stays
      reachable.
    </p>
  </Card>

  <div class="flex flex-wrap items-center gap-3 lg:col-span-2">
    <button class="btn-primary" type="submit" disabled={!dirty || save.busy}>Save</button>
    {#if dirty}
      <button class="btn-ghost" type="button" onclick={() => (form = engineForm(view))}>Undo changes</button>
    {/if}
    {#if view.stamp.updatedAt}
      <span class="muted text-xs"
        >Last changed {formatDate(view.stamp.updatedAt)} by {view.stamp.updatedByName || 'a removed account'}</span
      >
    {/if}
  </div>
</form>

<Card
  class="mt-5"
  title="Engine admins"
  subtitle="Admins change these settings and choose other admins. One must remain."
>
  <ul class="divide-y divide-ink-900/5 dark:divide-ink-50/5">
    {#each users as u (u.id)}
      <li class="flex items-center gap-3 py-2">
        <div class="min-w-0 flex-1">
          <p class="truncate text-sm font-medium">{u.displayName}{u.id === me ? ' (you)' : ''}</p>
          <p class="muted truncate text-xs">{u.email}</p>
        </div>
        <label class="flex items-center gap-2 text-sm">
          <input type="checkbox" checked={u.admin} onchange={(e) => void setAdmin(u, e.currentTarget)} /> Admin
        </label>
      </li>
    {/each}
  </ul>
</Card>
