<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import { pendingDrafts } from '$lib/state/drafts.svelte';
  import { Icon } from '@libs/ui-kit';
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

  // Hover tints, a press sinks the item, the current page is an outlined bronze tab on its shadow.
  const linkBase =
    'relative flex items-center gap-3 rounded-lg border-[1.5px] border-transparent px-3 py-2 text-sm font-medium transition duration-150 focus-visible:ring-2 focus-visible:ring-accent-600 focus-visible:outline-none';
  const activeClass = 'border-edge! bg-accent-100 text-stone-900 shadow-hard-sm dark:bg-accent-900 dark:text-stone-50';
  const idleClass =
    'text-stone-600 hover:bg-stone-900/[0.06] hover:text-stone-900 active:bg-stone-900/10 dark:text-stone-300 dark:hover:bg-white/10 dark:hover:text-white dark:active:bg-white/15';
</script>

{#snippet badge(count: number)}
  <span
    class="ml-auto rounded-md bg-accent-600 px-1.5 text-[0.7rem] leading-5 font-semibold text-white tabular-nums dark:bg-accent-400 dark:text-stone-900"
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
        ? 'text-accent-700 dark:text-accent-300'
        : 'text-stone-500 dark:text-stone-400'}"
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
              class="mt-0.5 ml-6 flex flex-col gap-0.5 border-l border-stone-900/10 pl-3 dark:border-white/10"
            >
              {#each item.children as child (child.href)}
                {@const childCurrent = current?.href === child.href}
                <li>
                  <a
                    href={href(child.href)}
                    onclick={onnavigate}
                    aria-current={childCurrent ? 'page' : undefined}
                    class="flex items-center rounded-lg px-2 py-1.5 text-sm transition focus-visible:ring-2 focus-visible:ring-accent-500 focus-visible:outline-none active:bg-stone-900/10 dark:active:bg-white/15 {childCurrent
                      ? 'font-medium text-accent-700 dark:text-accent-300'
                      : 'text-stone-500 hover:bg-stone-900/[0.04] hover:text-stone-800 dark:text-stone-400 dark:hover:bg-white/5 dark:hover:text-stone-200'}"
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
        <li class={area.utility ? 'mt-3 border-t border-stone-900/10 pt-3 dark:border-white/10' : ''}>
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
                <span class="absolute -top-1 -right-1 h-2 w-2 rounded-full bg-accent-600 dark:bg-accent-400"></span>
              {/if}
            </span>
          </a>
        </li>
      {/each}
    </ul>
  {:else}
    {#each AREAS as area (area.id)}
      {#if area.utility}
        <div class="mt-3 border-t border-stone-900/10 pt-3 dark:border-white/10">{@render single(area)}</div>
      {:else if area.pages.length === 1}
        <div class={area.id === 'home' ? '' : 'mt-3'}>{@render single(area)}</div>
      {:else}
        {@render group(area)}
      {/if}
    {/each}
  {/if}
</nav>
