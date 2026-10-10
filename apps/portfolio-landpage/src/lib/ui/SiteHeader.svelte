<script lang="ts">
  import { page } from '$app/state';
  import { nav, site } from '$lib/data/site';
  import ThemeToggle from './ThemeToggle.svelte';

  let isMenuOpen = $state(false);

  const isActive = (href: string) => !href.includes('#') && page.url.pathname.startsWith(href);

  $effect(() => {
    // Close the mobile menu after any navigation.
    void page.url.pathname;
    isMenuOpen = false;
  });
</script>

<header class="site-header sticky top-0 z-40 px-4 pt-4 sm:px-6">
  <div class="panel mx-auto flex max-w-6xl items-center justify-between gap-4 px-4 py-2.5 !shadow-ink-sm">
    <a href="/" class="group flex items-center gap-2" aria-label="{site.name}, home" data-cursor="home">
      <span
        class="grid h-9 w-9 place-items-center border-[3px] border-rule bg-accent font-display text-lg text-accent-ink transition-transform group-hover:rotate-[-8deg]"
        aria-hidden="true">D</span
      >
      <span class="font-display text-lg tracking-tight">{site.name}</span>
    </a>

    <nav aria-label="Main" class="hidden md:block">
      <ul class="flex items-center gap-1">
        {#each nav as item (item.href)}
          <li>
            <a
              href={item.href}
              class="px-3 py-1.5 font-mono text-sm font-semibold uppercase tracking-wider transition-colors hover:bg-ink hover:text-paper"
              class:bg-ink={isActive(item.href)}
              class:text-paper={isActive(item.href)}
              aria-current={isActive(item.href) ? 'page' : undefined}>{item.label}</a
            >
          </li>
        {/each}
      </ul>
    </nav>

    <div class="flex items-center gap-2">
      <ThemeToggle />
      <button
        type="button"
        class="panel grid h-10 w-10 place-items-center !shadow-ink-sm md:hidden"
        aria-label={isMenuOpen ? 'Close menu' : 'Open menu'}
        aria-expanded={isMenuOpen}
        aria-controls="mobile-menu"
        onclick={() => (isMenuOpen = !isMenuOpen)}
      >
        <svg viewBox="0 0 24 24" class="h-5 w-5" fill="none" stroke="currentColor" stroke-width="3" aria-hidden="true">
          {#if isMenuOpen}
            <path d="M6 6l12 12M18 6 6 18" />
          {:else}
            <path d="M4 7h16M4 12h16M4 17h10" />
          {/if}
        </svg>
      </button>
    </div>
  </div>

  {#if isMenuOpen}
    <nav id="mobile-menu" aria-label="Mobile" class="panel mx-auto mt-3 max-w-6xl p-2 md:hidden">
      <ul>
        {#each nav as item (item.href)}
          <li>
            <a
              href={item.href}
              class="block px-3 py-3 font-display text-xl hover:bg-ink hover:text-paper"
              onclick={() => (isMenuOpen = false)}>{item.label}</a
            >
          </li>
        {/each}
      </ul>
    </nav>
  {/if}
</header>

<style>
  .site-header {
    view-transition-name: site-header;
  }
</style>
