<script lang="ts">
  import './layout.css';
  import favicon from '$lib/assets/favicon.svg';
  import { untrack } from 'svelte';
  import { afterNavigate, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { navigating, page } from '$app/state';
  import AIUsageGauge from '$lib/ai/AIUsageGauge.svelte';
  import { track } from '$lib/api/deferred.svelte';
  import { endBootScreen } from '$lib/boot';
  import ChatWidget from '$lib/chat/ChatWidget.svelte';
  import DependencyBanner from '$lib/components/DependencyBanner.svelte';
  import Icon from '$lib/components/Icon.svelte';
  import MaskToggle from '$lib/components/MaskToggle.svelte';
  import Modal from '$lib/components/Modal.svelte';
  import SheetsAuthAlert from '$lib/components/SheetsAuthAlert.svelte';
  import AreaTabs from '$lib/components/AreaTabs.svelte';
  import MobileNav from '$lib/components/MobileNav.svelte';
  import NavProgress from '$lib/components/NavProgress.svelte';
  import Sidebar from '$lib/components/Sidebar.svelte';
  import SyncButton from '$lib/components/SyncButton.svelte';
  import ThemeToggle from '$lib/components/ThemeToggle.svelte';
  import Toasts from '$lib/components/Toasts.svelte';
  import { titleFor } from '$lib/components/nav';
  import { pendingDrafts } from '$lib/state/drafts.svelte';
  import { engineHealth } from '$lib/state/health.svelte';
  import { mask } from '$lib/state/mask.svelte';
  import { network } from '$lib/state/network.svelte';
  import { applyTheme, sidebarCollapsed, theme } from '$lib/state/settings.svelte';
  import { AUTO_SYNC_MS, sync } from '$lib/state/sync.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import EntryReminder from '$lib/transactions/EntryReminder.svelte';

  let { data, children } = $props();

  let mobileNavOpen = $state(false);
  let mobileNav = $state<HTMLDialogElement>();
  // A modal dialog makes the page inert, keeps focus inside and gives focus back to the menu button on close.
  $effect(() => {
    if (!mobileNav) return;
    if (mobileNavOpen && !mobileNav.open) mobileNav.showModal();
    if (!mobileNavOpen && mobileNav.open) mobileNav.close();
  });
  // The sheet is hidden from the desktop breakpoint up; an open one would leave the page inert behind nothing.
  $effect(() => {
    const desktop = globalThis.matchMedia?.('(min-width: 64rem)');
    if (!desktop) return;
    const onChange = () => {
      if (desktop.matches) mobileNavOpen = false;
    };
    desktop.addEventListener('change', onChange);
    return () => desktop.removeEventListener('change', onChange);
  });
  const collapsed = $derived(sidebarCollapsed.value === 'true');
  const path = $derived(page.url.pathname.slice(base.length) || '/');
  const title = $derived(titleFor(path));
  // Health arrives in the background; the last answer stays while a sync re-checks it.
  const engine = track(() => data.engine);
  $effect(() => {
    const settled = engine.settled;
    untrack(() => engineHealth.report(settled));
  });
  const aiReady = $derived(engineHealth.ready('ai'));
  const sheetsReady = $derived(engineHealth.ready('sheets'));

  // Stealth mode re-masks on every navigation (WM-UI-2).
  afterNavigate(() => {
    mask.hide();
    mobileNavOpen = false;
  });

  // Drafts arrive from the phone in the background: recount on navigation and focus.
  $effect(() => {
    if (!sheetsReady) return;
    const refresh = () => void pendingDrafts.refresh();
    refresh();
    window.addEventListener('focus', refresh);
    return () => window.removeEventListener('focus', refresh);
  });
  afterNavigate(() => {
    if (untrack(() => sheetsReady)) void pendingDrafts.refresh();
  });

  // The first render replaces app.html's loading screen.
  $effect(() => endBootScreen());

  // Theme follows the setting, and the OS when set to system.
  $effect(() => {
    const query = window.matchMedia('(prefers-color-scheme: dark)');
    const update = () => applyTheme(theme.value, query.matches);
    update();
    query.addEventListener('change', update);
    return () => query.removeEventListener('change', update);
  });

  // Auto-sync on load and every 15 minutes (WM-UI-3), once the sheet is reachable.
  let autoSyncStarted = false;
  $effect(() => {
    if (!sheetsReady || autoSyncStarted) return;
    autoSyncStarted = true;
    void sync.run(invalidateAll);
  });
  $effect(() => {
    const timer = setInterval(() => {
      if (sheetsReady && network.online) void sync.run(invalidateAll);
    }, AUTO_SYNC_MS);
    return () => clearInterval(timer);
  });

  // Offline blocks the app (WM-UI-6); on reconnect the data is re-synced.
  $effect(() => {
    untrack(() => network.update(navigator.onLine));
    const onChange = () => {
      if (network.update(navigator.onLine) !== 'restored') return;
      toasts.info('Back online. Refreshing your data.');
      void (sheetsReady ? sync.run(invalidateAll) : invalidateAll());
    };
    window.addEventListener('online', onChange);
    window.addEventListener('offline', onChange);
    return () => {
      window.removeEventListener('online', onChange);
      window.removeEventListener('offline', onChange);
    };
  });
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<NavProgress active={!!navigating.to} />

<div class="flex min-h-screen">
  <!-- Desktop sidebar -->
  <aside
    class="glass sticky top-3 m-3 mr-0 hidden h-[calc(100vh-1.5rem)] shrink-0 flex-col p-3 transition-[width] lg:flex {collapsed
      ? 'w-[4.25rem]'
      : 'w-60'}"
  >
    <a
      href="{base}/"
      class="mb-4 flex items-center gap-2 rounded-xl px-2 py-1 font-semibold tracking-tight transition hover:bg-slate-900/[0.04] dark:hover:bg-white/5"
    >
      <span class="grid size-8 place-items-center rounded-xl bg-indigo-600 text-sm text-white">₫</span>
      {#if !collapsed}<span>Wealth</span>{/if}
    </a>
    <div class="flex-1 overflow-y-auto"><Sidebar {collapsed} /></div>
    <button
      type="button"
      class="btn-ghost mt-2 {collapsed ? 'px-2' : 'justify-start'}"
      aria-label={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      title={collapsed ? 'Expand sidebar' : 'Collapse sidebar'}
      onclick={() => (sidebarCollapsed.value = collapsed ? 'false' : 'true')}
    >
      <Icon name={collapsed ? 'chevronRight' : 'chevronLeft'} />
      {#if !collapsed}<span>Collapse</span>{/if}
    </button>
  </aside>

  <!-- Mobile navigation sheet -->
  <!-- The dialog covers the screen; a click on it outside the sheet is a click on the backdrop. -->
  <dialog
    bind:this={mobileNav}
    class="m-0 h-full max-h-none w-full max-w-none bg-transparent p-0 text-inherit backdrop:bg-slate-950/40 backdrop:backdrop-blur-sm lg:hidden"
    aria-label="All pages"
    onclose={() => (mobileNavOpen = false)}
    onclick={(event) => {
      if (event.target === mobileNav) mobileNavOpen = false;
    }}
  >
    {#if mobileNavOpen}
      <aside class="glass absolute inset-y-3 left-3 w-64 overflow-y-auto p-3">
        <div class="mb-2 flex items-center justify-between px-2 py-1">
          <p class="font-semibold tracking-tight">Wealth</p>
          <button
            type="button"
            class="btn-ghost px-2"
            aria-label="Close menu"
            title="Close menu"
            onclick={() => (mobileNavOpen = false)}
          >
            <Icon name="x" />
          </button>
        </div>
        <Sidebar onnavigate={() => (mobileNavOpen = false)} />
      </aside>
    {/if}
  </dialog>

  <div class="flex min-w-0 flex-1 flex-col">
    <header class="glass sticky top-3 z-30 m-3 flex items-center gap-2 px-3 py-2">
      <button
        type="button"
        class="btn-ghost px-2 lg:hidden"
        aria-label="Open menu"
        title="Open menu"
        aria-expanded={mobileNavOpen}
        onclick={() => (mobileNavOpen = true)}
      >
        <Icon name="menu" />
      </button>
      <p class="flex-1 truncate font-semibold">{title}</p>
      {#if aiReady}<AIUsageGauge />{/if}
      <MaskToggle />
      <ThemeToggle />
      {#if sheetsReady}<SyncButton />{/if}
    </header>

    <main class="mx-auto w-full max-w-7xl flex-1 px-3 pt-3 pb-28 sm:px-6 lg:pb-12">
      <AreaTabs />
      {#if engineHealth.checked}<DependencyBanner health={engineHealth.health} />{/if}
      <SheetsAuthAlert />
      <EntryReminder ready={sheetsReady} />
      {@render children()}
    </main>
  </div>
</div>

<MobileNav />

{#if aiReady && path !== '/chat'}<ChatWidget page={{ path, title }} />{/if}
<Toasts />

<Modal open={!network.online} title="You are offline" onclose={() => {}} dismissible={false}>
  <p class="text-sm">
    The ledger lives in Google Sheets, so nothing can be read or saved without a connection. This closes and your data
    refreshes as soon as you are back online.
  </p>
</Modal>
