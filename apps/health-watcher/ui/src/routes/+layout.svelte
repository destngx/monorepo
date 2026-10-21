<script lang="ts">
  import './layout.css';
  import { goto, invalidateAll } from '$app/navigation';
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { setUnauthorizedHandler } from '$lib/api/client';
  import { auth } from '$lib/api/endpoints';
  import Icon, { type IconName } from '$lib/components/Icon.svelte';
  import Toasts from '$lib/components/Toasts.svelte';
  import { session } from '$lib/state/session.svelte';
  import { toasts } from '$lib/state/toasts.svelte';
  import { isPublicPath } from '$lib/paths';

  let { children } = $props();

  const path = $derived(page.url.pathname.slice(base.length) || '/');
  const shell = $derived(session.me !== null && !isPublicPath(path));

  const nav: { href: string; label: string; icon: IconName }[] = [
    { href: '/', label: 'Home', icon: 'home' },
    { href: '/inbox', label: 'Inbox', icon: 'inbox' },
    { href: '/people', label: 'People', icon: 'users' },
    { href: '/settings', label: 'Settings', icon: 'settings' },
  ];

  function active(href: string): boolean {
    return href === '/' ? path === '/' : path === href || path.startsWith(`${href}/`);
  }

  // A request that finds the session gone sends the user back to login.
  $effect(() => {
    setUnauthorizedHandler(() => {
      if (!session.me) return;
      session.clear();
      toasts.info('Your session ended. Please log in again.');
      void goto(`${base}/login?next=${encodeURIComponent(path)}`);
    });
    return () => setUnauthorizedHandler(null);
  });

  async function switchTenant(id: string) {
    session.selectTenant(id);
    await goto(`${base}/`);
    await invalidateAll();
  }

  async function logout() {
    try {
      await auth.logout();
    } catch (error) {
      toasts.error(error, 'Could not log out');
      return;
    }
    session.clear();
    await goto(`${base}/login`);
  }
</script>

<svelte:head><title>Health Watcher</title></svelte:head>

{#if shell}
  <div class="mx-auto flex min-h-screen max-w-6xl flex-col px-3 pb-24 sm:px-6 lg:pb-8">
    <header class="glass sticky top-3 z-30 mt-3 flex items-center gap-3 px-3 py-2">
      <a href="{base}/" class="flex items-center gap-2 font-semibold tracking-tight">
        <span class="grid size-8 place-items-center rounded-xl bg-teal-600 text-white"
          ><Icon name="activity" size={18} /></span
        >
        <span class="hidden sm:inline">Health Watcher</span>
      </a>
      <nav class="ml-4 hidden items-center gap-1 lg:flex" aria-label="Main">
        {#each nav as item (item.href)}
          <a
            href="{base}{item.href}"
            class="btn-ghost px-3 py-1.5 {active(item.href) ? 'bg-teal-600/10 text-teal-700 dark:text-teal-300' : ''}"
            aria-current={active(item.href) ? 'page' : undefined}
          >
            <Icon name={item.icon} size={16} />{item.label}
          </a>
        {/each}
      </nav>
      <div class="ml-auto flex items-center gap-2">
        <form
          role="search"
          class="hidden md:block"
          onsubmit={(e) => {
            e.preventDefault();
            const q = new FormData(e.currentTarget).get('q')?.toString().trim() ?? '';
            if (q) void goto(`${base}/search?q=${encodeURIComponent(q)}`);
          }}
        >
          <label class="sr-only" for="header-search">Search</label>
          <input
            id="header-search"
            name="q"
            class="field w-48 py-1.5"
            type="search"
            placeholder="Search"
            maxlength="200"
          />
        </form>
        <a href="{base}/search" class="icon-btn size-9 md:hidden" aria-label="Search" title="Search">
          <Icon name="search" size={18} />
        </a>
        {#if (session.me?.memberships.length ?? 0) > 1}
          <label class="sr-only" for="tenant">Family or group</label>
          <select
            id="tenant"
            class="field w-auto py-1.5"
            value={session.tenantId}
            onchange={(e) => void switchTenant(e.currentTarget.value)}
          >
            {#each session.me?.memberships ?? [] as m (m.tenant.id)}
              <option value={m.tenant.id}>{m.tenant.name}</option>
            {/each}
          </select>
        {:else if session.membership}
          <span class="muted hidden text-sm sm:inline">{session.membership.tenant.name}</span>
        {/if}
        <button type="button" class="icon-btn size-9" aria-label="Log out" title="Log out" onclick={logout}>
          <Icon name="logout" size={18} />
        </button>
      </div>
    </header>

    <main class="mt-5 flex-1">
      {#if session.tenantId}
        {@render children()}
      {:else}
        <div class="glass mx-auto max-w-lg p-6 text-center">
          <p class="font-medium">You are not in any family or group yet.</p>
          <p class="muted mt-1 text-sm">Open an invite link, or create a group in Settings.</p>
          <a class="btn-primary mt-4" href="{base}/settings">Settings</a>
        </div>
      {/if}
    </main>

    <nav
      class="glass fixed inset-x-2 bottom-2 z-30 grid grid-cols-4 px-1 py-1.5 lg:hidden"
      style="padding-bottom: max(0.375rem, env(safe-area-inset-bottom))"
      aria-label="Main"
    >
      {#each nav as item (item.href)}
        <a
          href="{base}{item.href}"
          class="flex flex-col items-center gap-0.5 rounded-xl py-1 text-xs {active(item.href)
            ? 'text-teal-700 dark:text-teal-300'
            : 'muted'}"
          aria-current={active(item.href) ? 'page' : undefined}
        >
          <Icon name={item.icon} size={20} />{item.label}
        </a>
      {/each}
    </nav>
  </div>
{:else}
  <main class="flex min-h-screen items-start justify-center px-4 py-12 sm:items-center">
    {@render children()}
  </main>
{/if}

<Toasts />
