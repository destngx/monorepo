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
    <ul class="flex gap-1.5 px-1 pt-0.5 pr-2 pb-1.5">
      {#each area.pages as item (item.href)}
        {@const active = isPageActive(item.href, path)}
        <li class="shrink-0">
          <a
            href="{base}{item.href}"
            aria-current={current?.href === item.href ? 'page' : active ? 'true' : undefined}
            class="block rounded-lg border-[1.5px] px-3 py-1.5 text-sm font-medium whitespace-nowrap transition focus-visible:ring-2 focus-visible:ring-accent-600 focus-visible:outline-none {active
              ? 'border-edge bg-stone-900 text-stone-50 shadow-hard-sm dark:bg-stone-100 dark:text-stone-900'
              : 'border-edge/30 bg-surface text-stone-700 hover:border-edge active:bg-stone-900/10 dark:text-stone-200 dark:active:bg-white/15'}"
            >{item.label}</a
          >
        </li>
      {/each}
    </ul>
  </nav>
{/if}
