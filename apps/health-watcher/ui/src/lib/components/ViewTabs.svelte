<script lang="ts">
  import { Icon, type IconName } from '@libs/ui-kit';

  // Views of one thing as links, so each view has its own address: a
  // person's timeline and metrics, a document's results and pages.
  interface Tab {
    key: string;
    label: string;
    icon: IconName;
    href: string;
  }
  let { tabs, current, label }: { tabs: Tab[]; current: string; label: string } = $props();
</script>

<nav aria-label={label} class="mb-5 flex gap-1">
  {#each tabs as tab (tab.key)}
    <a
      href={tab.href}
      class="btn-ghost px-3 py-1.5 {tab.key === current
        ? 'border-edge bg-ink-900 text-ink-50 hover:bg-ink-900 hover:text-ink-50 dark:bg-ink-100 dark:text-ink-950 dark:hover:bg-ink-100 dark:hover:text-ink-950'
        : ''}"
      aria-current={tab.key === current ? 'page' : undefined}><Icon name={tab.icon} size={16} />{tab.label}</a
    >
  {/each}
</nav>
