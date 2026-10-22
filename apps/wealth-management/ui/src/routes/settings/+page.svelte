<script lang="ts">
  import { base } from '$app/paths';
  import type { GoogleAuthPurpose } from '$lib/api/types';
  import { goto, invalidateAll } from '$app/navigation';
  import { page } from '$app/state';
  import { onMount } from 'svelte';
  import AIStatusCard from '$lib/ai/AIStatusCard.svelte';
  import { Button, LoadError, Modal, SkeletonCard } from '@libs/ui-kit';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import ApiTokenCard from '$lib/settings/ApiTokenCard.svelte';
  import BrowserPreferences from '$lib/settings/BrowserPreferences.svelte';
  import ConnectionsSection from '$lib/settings/ConnectionsSection.svelte';
  import { EngineSettingsForm } from '$lib/settings/engine.svelte';
  import GoalPoolCard from '$lib/settings/GoalPoolCard.svelte';
  import GoogleConnect from '$lib/settings/GoogleConnect.svelte';
  import { googleAuthNotice, googleAuthParams } from '$lib/settings/googleAuth';
  import SaveBar from '$lib/settings/SaveBar.svelte';
  import { sectionFromUrl, sections, sectionsWithChanges } from '$lib/settings/sections';
  import SettingsGroupCard from '$lib/settings/SettingsGroupCard.svelte';
  import { sync } from '$lib/state/sync.svelte';
  import { toasts } from '$lib/state/toasts.svelte';

  // The tabs and browser preferences show at once; the engine settings, the
  // AI status, the sheet and the goal pool each load on their own with a
  // skeleton, then their values or an error with a Retry (WM-UI-16).
  const form = new EngineSettingsForm();
  const active = $derived(sectionFromUrl(page.url));
  const current = $derived(sections.find((section) => section.id === active) ?? sections[0]);
  const changed = $derived(form.view ? sectionsWithChanges(form.view.settings, form.changes) : new Set());
  const groups = $derived(form.groupsIn(current.groups));

  $effect(() => {
    void form.load();
  });

  // The model selects need the endpoint's models; ask once the AI tab opens.
  $effect(() => {
    if (active === 'ai' && form.models === null) void form.loadModels();
  });

  // The engine's Google callback redirects here with the outcome; report it
  // once and drop it from the address so a reload does not repeat it.
  onMount(() => {
    const notice = googleAuthNotice(page.url.searchParams);
    if (!notice) return;
    toasts.push(notice.kind, notice.message);
    const url = new URL(page.url);
    url.searchParams.set('tab', sectionFromUrl(url));
    for (const key of googleAuthParams) url.searchParams.delete(key);
    void goto(url, { replaceState: true, noScroll: true, keepFocus: true });
  });

  // Sign-ins save a token, which would overwrite unsaved edits on reload.
  const connects: Record<string, { purpose: GoogleAuthPurpose; label: string }> = {
    sheets: { purpose: 'sheets', label: 'Connect Google account' },
  };

  let confirmFlush = $state(false);
  async function flush() {
    confirmFlush = false;
    if (await sync.run(invalidateAll)) toasts.push('success', 'Caches flushed; data reloaded from the sheet.');
  }
</script>

<PageHeader title="Settings" />

<div class="grid grid-cols-1 gap-6 md:grid-cols-[11rem_minmax(0,1fr)]">
  <nav
    aria-label="Settings sections"
    class="-mx-1 flex gap-1 overflow-x-auto px-1 md:sticky md:top-4 md:flex-col md:self-start"
  >
    {#each sections as section (section.id)}
      <a
        href="{base}/settings?tab={section.id}"
        aria-current={section.id === active ? 'page' : undefined}
        class="flex shrink-0 items-center justify-between gap-2 rounded-xl px-3 py-2 text-sm font-medium whitespace-nowrap transition active:scale-[0.98] motion-reduce:active:scale-100 {section.id ===
        active
          ? 'bg-accent-600/10 text-accent-700 dark:bg-accent-400/15 dark:text-accent-300'
          : 'text-stone-600 hover:bg-stone-900/5 dark:text-stone-300 dark:hover:bg-white/5'}"
      >
        {section.label}
        {#if changed.has(section.id)}
          <span class="size-1.5 rounded-full bg-amber-500" title="Unsaved changes"></span>
          <span class="sr-only">(unsaved changes)</span>
        {/if}
      </a>
    {/each}
  </nav>

  <div class="min-w-0 max-w-3xl space-y-4">
    {#if active === 'general'}
      <BrowserPreferences />
      <ApiTokenCard />
    {:else if active === 'connections'}
      <ConnectionsSection />
    {:else if active === 'ai'}
      {#key form.generation}<AIStatusCard />{/key}
    {:else if active === 'goals'}
      <GoalPoolCard />
    {/if}

    {#if current.groups.length}
      {#if form.loadFailure && !form.view}
        <LoadError failure={form.loadFailure} label="the engine settings" retry={form.load} />
      {:else if !form.view}
        {#each current.groups as id (id)}
          <SkeletonCard lines={3} label="the engine settings" />
        {/each}
      {:else}
        {#each groups as grouped (grouped.group.id)}
          {@const connect = connects[grouped.group.id]}
          <SettingsGroupCard {grouped} {form}>
            {#snippet actions()}
              {#if connect}
                <GoogleConnect
                  {...connect}
                  disabled={form.changeCount > 0 || form.saving}
                  disabledReason="Save or discard your changes first"
                  onchange={form.load}
                />
              {:else if grouped.group.id === 'general'}
                <Button
                  size="sm"
                  icon="refresh"
                  busy={form.restarting}
                  busyText="Restarting…"
                  disabled={form.saving || form.changeCount > 0}
                  title={form.changeCount > 0 ? 'Save or discard your changes first' : undefined}
                  onclick={form.restart}>Restart engine</Button
                >
              {:else if grouped.group.id === 'cache'}
                <Button size="sm" busy={sync.syncing} busyText="Flushing…" onclick={() => (confirmFlush = true)}
                  >Flush caches</Button
                >
              {/if}
            {/snippet}
            {#snippet intro()}
              {#if grouped.group.id === 'ai' && form.modelsError}
                <p class="mb-3 rounded-xl bg-amber-500/10 px-3 py-2 text-xs text-amber-800 dark:text-amber-200">
                  Could not list the endpoint's models ({form.modelsError}); type model ids instead.
                  <button type="button" class="link" onclick={form.loadModels}>Retry</button>
                </p>
              {/if}
            {/snippet}
            {#if grouped.group.id === 'general'}
              <p class="muted mt-3 text-xs">Stored in <code class="font-mono break-all">{form.view.path}</code></p>
            {/if}
          </SettingsGroupCard>
        {/each}
      {/if}
    {/if}

    <SaveBar {form} />
  </div>
</div>

<Modal open={confirmFlush} title="Flush caches?" onclose={() => (confirmFlush = false)}>
  <p class="text-sm">
    Every page reloads its data from Google Sheets. This is safe but slower for the next few requests.
  </p>
  {#snippet footer()}
    <Button onclick={() => (confirmFlush = false)}>Cancel</Button>
    <Button variant="danger" busy={sync.syncing} busyText="Flushing…" onclick={flush}>Flush</Button>
  {/snippet}
</Modal>
