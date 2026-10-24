<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { pendingDrafts } from '$lib/state/drafts.svelte';
  import { Icon } from '@libs/ui-kit';
  import { PRIMARY_AREAS, areaFor, pageFor } from './nav';

  // Phone area bar (ROADMAP P1, WM-UI-12): the five areas plus Assistant,
  // each an icon with its label. The area's own pages are AreaTabs. It floats
  // just above the home indicator, not over it, so it needs no inner padding.
  const path = $derived(page.url.pathname.slice(base.length) || '/');
  const currentArea = $derived(areaFor(path));
  const current = $derived(pageFor(path));
</script>

<nav
  aria-label="Areas"
  class="panel fixed right-[max(0.5rem,env(safe-area-inset-right))] bottom-[max(0.5rem,calc(env(safe-area-inset-bottom)-0.75rem))] left-[max(0.5rem,env(safe-area-inset-left))] z-30 lg:hidden"
>
  <ul class="grid grid-cols-6">
    {#each PRIMARY_AREAS as area (area.id)}
      {@const active = currentArea?.id === area.id}
      {@const count = area.badge === 'pendingDrafts' ? (pendingDrafts.count ?? 0) : 0}
      <li>
        <a
          href="{base}{area.href === '/' ? '/' : area.href}"
          aria-current={active ? (current?.href === area.href ? 'page' : 'true') : undefined}
          class="group flex min-h-14 flex-col items-center justify-center gap-0.5 rounded-xl px-1 py-1.5 text-[0.68rem] font-medium transition focus-visible:ring-2 focus-visible:ring-accent-600 focus-visible:outline-none {active
            ? 'text-stone-900 dark:text-stone-50'
            : 'text-stone-600 hover:text-stone-900 dark:text-stone-300 dark:hover:text-white'}"
        >
          <span
            class="relative grid h-7 w-11 place-items-center rounded-md border-[1.5px] transition duration-150 group-active:scale-90 motion-reduce:group-active:scale-100 {active
              ? 'border-edge bg-accent-200 dark:bg-accent-800'
              : 'border-transparent group-hover:bg-stone-900/[0.06] group-active:bg-stone-900/10 dark:group-hover:bg-white/10 dark:group-active:bg-white/15'}"
          >
            <Icon name={area.icon} size={20} />
            {#if count}
              <span class="absolute top-0.5 right-2 h-2 w-2 rounded-full bg-accent-600 dark:bg-accent-400"></span>
            {/if}
          </span>
          <span class="max-w-full truncate"
            >{area.label}{#if count}<span class="sr-only">, {count} drafts waiting</span>{/if}</span
          >
        </a>
      </li>
    {/each}
  </ul>
</nav>
