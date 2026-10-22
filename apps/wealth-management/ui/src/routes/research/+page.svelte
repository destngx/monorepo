<script lang="ts">
  import { base } from '$app/paths';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { AREAS } from '$lib/components/nav';

  // Research area landing (ROADMAP M5, WM-UI-14): the market tools, kept
  // apart from the user's own money. Market data comes from the market data
  // server; nothing here changes the ledger. Nothing loads: the tools fetch
  // their own data, so the landing renders at once.
  const tools = (AREAS.find((area) => area.id === 'research')?.pages ?? []).filter((page) => page.href !== '/research');
</script>

<PageHeader title="Research" subtitle="Markets, news and ideas, aimed at what you hold" />

<ul class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3" aria-label="Research tools">
  {#each tools as tool (tool.href)}
    <li>
      <a
        href="{base}{tool.href}"
        class="panel interactive group flex h-full items-start gap-3 p-5 hover:bg-white/75 dark:hover:bg-stone-900/70"
      >
        <span
          class="grid size-10 shrink-0 place-items-center rounded-xl bg-accent-600/10 text-accent-700 transition-colors group-hover:bg-accent-600 group-hover:text-white dark:bg-accent-400/15 dark:text-accent-300 dark:group-hover:bg-accent-500"
        >
          <Icon name={tool.icon} />
        </span>
        <span class="min-w-0">
          <span class="flex items-center gap-1 font-semibold"
            >{tool.label}<Icon
              name="chevronRight"
              size={14}
              class="opacity-0 transition group-hover:translate-x-0.5 group-hover:opacity-60"
            /></span
          >
          {#if tool.description}<span class="muted mt-1 block text-sm">{tool.description}</span>{/if}
        </span>
      </a>
    </li>
  {/each}
</ul>
