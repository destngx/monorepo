<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { pendingDrafts } from '$lib/state/drafts.svelte';
  import Icon from './Icon.svelte';
  import { AREAS, areaFor, isPageActive, pageFor, type NavArea, type NavPage } from './nav';

  // Grouped navigation (ROADMAP P1, WM-UI-11): Home, the four areas with
  // their pages, Assistant, then Settings. Collapsed, each area is one icon.
  interface Props {
    collapsed?: boolean;
    /** Called after a link is followed (closes the mobile sheet). */
    onnavigate?: () => void;
  }

  let { collapsed = false, onnavigate }: Props = $props();

  const path = $derived(page.url.pathname.slice(base.length) || '/');
  const current = $derived(pageFor(path));
  const currentArea = $derived(areaFor(path));

  function badgeCount(badge: NavPage['badge']): number {
    return badge === 'pendingDrafts' ? (pendingDrafts.count ?? 0) : 0;
  }

  function href(target: string): string {
    return `${base}${target === '/' ? '/' : target}`;
  }

  const linkBase =
    'flex items-center gap-3 rounded-xl px-3 py-2 text-sm font-medium transition focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:outline-none';
  const activeClass = 'bg-indigo-600/10 text-indigo-700 dark:bg-indigo-400/15 dark:text-indigo-300';
  const idleClass = 'text-slate-600 hover:bg-slate-900/5 dark:text-slate-300 dark:hover:bg-white/5';
</script>

{#snippet badge(count: number)}
  <span
    class="ml-auto rounded-full bg-indigo-600 px-1.5 text-[0.7rem] leading-5 font-semibold text-white tabular-nums dark:bg-indigo-400 dark:text-slate-900"
    >{count}<span class="sr-only"> waiting</span></span
  >
{/snippet}

{#snippet single(area: NavArea)}
  {@const item = area.pages[0]}
  {@const active = isPageActive(item.href, path)}
  <a
    href={href(item.href)}
    onclick={onnavigate}
    aria-current={current?.href === item.href ? 'page' : undefined}
    class="{linkBase} {active ? activeClass : idleClass}"
  >
    <Icon name={item.icon} />
    <span>{item.label}</span>
  </a>
{/snippet}

{#snippet group(area: NavArea)}
  {@const headingId = `nav-area-${area.id}`}
  <div class="mt-3">
    <p
      id={headingId}
      class="mb-1 flex items-center gap-2 px-3 text-[0.7rem] font-semibold tracking-wider uppercase {currentArea?.id ===
      area.id
        ? 'text-indigo-700 dark:text-indigo-300'
        : 'text-slate-500 dark:text-slate-400'}"
    >
      <Icon name={area.icon} size={14} />
      {area.label}
    </p>
    <ul aria-labelledby={headingId} class="flex flex-col gap-0.5">
      {#each area.pages as item (item.href)}
        {@const active = isPageActive(item.href, path)}
        <li>
          <a
            href={href(item.href)}
            onclick={onnavigate}
            aria-current={current?.href === item.href ? 'page' : undefined}
            class="{linkBase} {active ? activeClass : idleClass}"
          >
            <Icon name={item.icon} size={16} />
            <span>{item.label}</span>
            {#if badgeCount(item.badge) && !(active && item.children)}{@render badge(badgeCount(item.badge))}{/if}
          </a>
          {#if item.children && active}
            <ul
              aria-label="{item.label} pages"
              class="mt-0.5 ml-6 flex flex-col gap-0.5 border-l border-slate-900/10 pl-3 dark:border-white/10"
            >
              {#each item.children as child (child.href)}
                {@const childCurrent = current?.href === child.href}
                <li>
                  <a
                    href={href(child.href)}
                    onclick={onnavigate}
                    aria-current={childCurrent ? 'page' : undefined}
                    class="flex items-center rounded-lg px-2 py-1.5 text-sm transition focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:outline-none {childCurrent
                      ? 'font-medium text-indigo-700 dark:text-indigo-300'
                      : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200'}"
                  >
                    {child.label}
                    {#if badgeCount(child.badge)}{@render badge(badgeCount(child.badge))}{/if}
                  </a>
                </li>
              {/each}
            </ul>
          {/if}
        </li>
      {/each}
    </ul>
  </div>
{/snippet}

<nav aria-label="Main" class="flex flex-col">
  {#if collapsed}
    <ul class="flex flex-col gap-1">
      {#each AREAS as area (area.id)}
        {@const active = currentArea?.id === area.id}
        {@const count = badgeCount(area.badge)}
        <li class={area.utility ? 'mt-3 border-t border-slate-900/10 pt-3 dark:border-white/10' : ''}>
          <a
            href={href(area.href)}
            onclick={onnavigate}
            title={area.label}
            aria-label={count ? `${area.label}, ${count} drafts waiting` : area.label}
            aria-current={active ? (current?.href === area.href ? 'page' : 'true') : undefined}
            class="{linkBase} justify-center px-2 {active ? activeClass : idleClass}"
          >
            <span class="relative">
              <Icon name={area.icon} />
              {#if count}
                <span class="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-indigo-600 dark:bg-indigo-400"></span>
              {/if}
            </span>
          </a>
        </li>
      {/each}
    </ul>
  {:else}
    {#each AREAS as area (area.id)}
      {#if area.utility}
        <div class="mt-3 border-t border-slate-900/10 pt-3 dark:border-white/10">{@render single(area)}</div>
      {:else if area.pages.length === 1}
        <div class={area.id === 'home' ? '' : 'mt-3'}>{@render single(area)}</div>
      {:else}
        {@render group(area)}
      {/if}
    {/each}
  {/if}
</nav>
