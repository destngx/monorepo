<script lang="ts">
  import { base } from '$app/paths';
  import ViewTabs from '$lib/components/ViewTabs.svelte';

  // The views of one person: the visit timeline and the metrics dashboard.
  let { personId, current }: { personId: string; current: 'timeline' | 'metrics' } = $props();
</script>

<ViewTabs
  label="Person"
  {current}
  tabs={[
    { key: 'timeline', label: 'Timeline', icon: 'calendar', href: `${base}/people/${personId}` },
    { key: 'metrics', label: 'Metrics', icon: 'activity', href: `${base}/people/${personId}/metrics` },
  ]}
/>
