<script lang="ts">
  import { base } from '$app/paths';
  import { AIUsageGauge } from '@libs/svelte-ai-usage';
  import { formatDateTime } from '$lib/format';
  import { leftColor } from './usage';
  import { aiUsage } from './usage.svelte';

  // Menu bar gauge of the OpenAI quota left (WM-SET-4), in this app's colours.
</script>

<!-- Below sm the details hang from the header's right edge (the sticky header is their containing
     block), since the gauge sits mid-header and a panel under it would run off the screen. -->
<AIUsageGauge
  class="sm:relative"
  source={aiUsage}
  {leftColor}
  formatReset={(resetsAt) => formatDateTime(resetsAt)}
  errorClass="text-rose-600 dark:text-rose-400"
  settingsHref="{base}/settings"
/>
