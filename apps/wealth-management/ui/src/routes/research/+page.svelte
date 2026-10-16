<script lang="ts">
  import { base } from '$app/paths';
  import Icon from '$lib/components/Icon.svelte';
  import PageHeader from '$lib/components/PageHeader.svelte';
  import { AREAS } from '$lib/components/nav';

  // Research area landing (ROADMAP M5, WM-UI-14): the market tools, kept
  // apart from the user's own money. Market data comes from the market data
  // server; nothing here changes the ledger.
  const tools = (AREAS.find((area) => area.id === 'research')?.pages ?? []).filter((page) => page.href !== '/research');
</script>

<PageHeader title="Research" subtitle="Markets, news and ideas, aimed at what you hold" />

<ul class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3" aria-label="Research tools">
  {#each tools as tool (tool.href)}
    <li>
      <a
        href="{base}{tool.href}"
        class="glass flex h-full items-start gap-3 p-5 transition hover:bg-white/60 focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:outline-none dark:hover:bg-white/10"
      >
        <span
          class="grid size-10 shrink-0 place-items-center rounded-xl bg-indigo-600/10 text-indigo-700 dark:bg-indigo-400/15 dark:text-indigo-300"
        >
          <Icon name={tool.icon} />
        </span>
        <span class="min-w-0">
          <span class="block font-semibold">{tool.label}</span>
          {#if tool.description}<span class="muted mt-1 block text-sm">{tool.description}</span>{/if}
        </span>
      </a>
    </li>
  {/each}
</ul>
