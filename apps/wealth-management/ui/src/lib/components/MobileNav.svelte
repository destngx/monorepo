<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { pendingDrafts } from '$lib/state/drafts.svelte';
  import Icon from './Icon.svelte';
  import { PRIMARY_AREAS, areaFor, pageFor } from './nav';

  // Phone area bar (ROADMAP P1, WM-UI-12): the five areas plus Assistant,
  // each an icon with its label. The area's own pages are AreaTabs.
  const path = $derived(page.url.pathname.slice(base.length) || '/');
  const currentArea = $derived(areaFor(path));
  const current = $derived(pageFor(path));
</script>

<nav
  aria-label="Areas"
  class="glass fixed inset-x-2 bottom-2 z-30 lg:hidden"
  style="padding-bottom: env(safe-area-inset-bottom)"
>
  <ul class="grid grid-cols-6">
    {#each PRIMARY_AREAS as area (area.id)}
      {@const active = currentArea?.id === area.id}
      {@const count = area.badge === 'pendingDrafts' ? (pendingDrafts.count ?? 0) : 0}
      <li>
        <a
          href="{base}{area.href === '/' ? '/' : area.href}"
          aria-current={active ? (current?.href === area.href ? 'page' : 'true') : undefined}
          class="group flex min-h-14 flex-col items-center justify-center gap-0.5 rounded-xl px-1 py-1.5 text-[0.68rem] font-medium transition focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:outline-none {active
            ? 'text-indigo-700 dark:text-indigo-300'
            : 'text-slate-600 hover:text-slate-900 dark:text-slate-300 dark:hover:text-white'}"
        >
          <span
            class="relative grid h-7 w-11 place-items-center rounded-full transition duration-150 group-active:scale-90 motion-reduce:group-active:scale-100 {active
              ? 'bg-indigo-600/10 dark:bg-indigo-400/15'
              : 'group-hover:bg-slate-900/[0.06] group-active:bg-slate-900/10 dark:group-hover:bg-white/10 dark:group-active:bg-white/15'}"
          >
            <Icon name={area.icon} size={20} />
            {#if count}
              <span class="absolute top-0.5 right-2 h-2 w-2 rounded-full bg-indigo-600 dark:bg-indigo-400"></span>
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
