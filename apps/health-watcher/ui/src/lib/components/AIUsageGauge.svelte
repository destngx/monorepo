<script lang="ts">
  import { base } from '$app/paths';
  import { AIUsageGauge } from '@libs/svelte-ai-usage';
  import { formatReset, leftColor } from '$lib/aiUsage';
  import { aiUsage } from '$lib/state/aiUsage.svelte';
  import { session } from '$lib/state/session.svelte';

  // Menu bar gauge of the AI quota left (HW-OPS-10), in this app's colours;
  // hidden while AI reading is off. Admins get a link to the AI settings.
  const settingsHref = $derived(session.me?.user.admin ? `${base}/settings/engine` : undefined);
</script>

<AIUsageGauge
  source={aiUsage}
  {leftColor}
  {formatReset}
  errorClass="text-danger-600 dark:text-danger-400"
  {settingsHref}
/>
