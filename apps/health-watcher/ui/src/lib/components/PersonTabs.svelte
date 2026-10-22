<script lang="ts">
  import { base } from '$app/paths';
  import { Icon, type IconName } from '@libs/ui-kit';

  // The views of one person: the visit timeline and the metrics dashboard.
  let { personId, current }: { personId: string; current: 'timeline' | 'metrics' } = $props();

  const tabs: { key: string; label: string; icon: IconName; href: string }[] = $derived([
    { key: 'timeline', label: 'Timeline', icon: 'calendar', href: `${base}/people/${personId}` },
    { key: 'metrics', label: 'Metrics', icon: 'activity', href: `${base}/people/${personId}/metrics` },
  ]);
</script>

<nav aria-label="Person" class="mb-5 flex gap-1">
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
