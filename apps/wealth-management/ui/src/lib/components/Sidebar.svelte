<script lang="ts">
  import { base } from '$app/paths';
  import { page } from '$app/state';
  import Icon from './Icon.svelte';
  import { NAV, isActive } from './nav';

  interface Props {
    collapsed?: boolean;
    /** Called after a link is followed (closes the mobile sheet). */
    onnavigate?: () => void;
  }

  let { collapsed = false, onnavigate }: Props = $props();

  const path = $derived(page.url.pathname.slice(base.length) || '/');
</script>

<nav aria-label="Main" class="flex flex-col gap-1">
  {#each NAV as item (item.href)}
    {@const active = isActive(item.href, path)}
    <a
      href="{base}{item.href === '/' ? '/' : item.href}"
      onclick={onnavigate}
      title={collapsed ? item.label : undefined}
      aria-current={path === item.href ? 'page' : undefined}
      class="flex items-center gap-3 rounded-xl px-3 py-2 text-sm font-medium transition {active
        ? 'bg-indigo-600/10 text-indigo-700 dark:bg-indigo-400/15 dark:text-indigo-300'
        : 'text-slate-600 hover:bg-slate-900/5 dark:text-slate-300 dark:hover:bg-white/5'} {collapsed
        ? 'justify-center px-2'
        : ''}"
    >
      <Icon name={item.icon} />
      {#if !collapsed}<span>{item.label}</span>{/if}
    </a>
    {#if item.children && !collapsed && active}
      <div class="ml-6 flex flex-col gap-0.5 border-l border-slate-900/10 pl-3 dark:border-white/10">
        {#each item.children as child (child.href)}
          <a
            href="{base}{child.href}"
            onclick={onnavigate}
            aria-current={isActive(child.href, path) ? 'page' : undefined}
            class="rounded-lg px-2 py-1.5 text-sm transition {isActive(child.href, path)
              ? 'font-medium text-indigo-700 dark:text-indigo-300'
              : 'text-slate-500 hover:text-slate-800 dark:text-slate-400 dark:hover:text-slate-200'}">{child.label}</a
          >
        {/each}
      </div>
    {/if}
  {/each}
</nav>
