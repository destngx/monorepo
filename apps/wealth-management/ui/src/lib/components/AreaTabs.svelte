<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { areaFor, isPageActive, pageFor } from './nav';

  // The current area's pages as a scrollable tab strip on phones (ROADMAP
  // P1, WM-UI-12). Sub-pages (drafts, capture) are linked from their page.
  const path = $derived(page.url.pathname.slice(base.length) || '/');
  const area = $derived(areaFor(path));
  const current = $derived(pageFor(path));
</script>

{#if area && area.pages.length > 1}
  <nav aria-label="{area.label} pages" class="-mx-1 mb-4 overflow-x-auto lg:hidden">
    <ul class="flex gap-1 px-1 pb-1">
      {#each area.pages as item (item.href)}
        {@const active = isPageActive(item.href, path)}
        <li class="shrink-0">
          <a
            href="{base}{item.href}"
            aria-current={current?.href === item.href ? 'page' : active ? 'true' : undefined}
            class="block rounded-full px-3 py-1.5 text-sm font-medium whitespace-nowrap transition focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:outline-none active:scale-95 motion-reduce:active:scale-100 {active
              ? 'bg-indigo-600 text-white dark:bg-indigo-400 dark:text-slate-900'
              : 'bg-slate-900/5 text-slate-700 hover:bg-slate-900/10 active:bg-slate-900/15 dark:bg-white/5 dark:text-slate-200 dark:hover:bg-white/10 dark:active:bg-white/15'}"
            >{item.label}</a
          >
        </li>
      {/each}
    </ul>
  </nav>
{/if}
