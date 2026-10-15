<script lang="ts">
  import { api } from '$lib/api/endpoints';
  import AIExplain from './AIExplain.svelte';

  // Sparkle button that explains the data behind a chart (WM-UI-8).
  interface Props {
    chartType: string;
    title: string;
    data: unknown;
    description?: string;
    timeframe?: string;
  }

  let { chartType, title, data, description, timeframe }: Props = $props();
</script>

<AIExplain
  title="{title} insight"
  label="Explain {title} with AI"
  load={(force, signal) => api.chartInsight({ chartType, title, data, description, timeframe }, { force, signal })}
/>
