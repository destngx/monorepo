<script lang="ts">
  import { base } from '$app/paths';
  import { type Tracked, Card, Icon, LoadError, Spinner } from '@libs/ui-kit';
  import type { AIStatus } from '$lib/api/types';

  // Whether a research page's AI flow can run, above its controls: a quiet
  // "Checking AI" line while the status streams in (the controls stay usable),
  // the reason and a Settings link when it cannot run, or the status error
  // with a Retry.
  interface Props {
    status: Tracked<AIStatus | null>;
    /** From aiUnavailable(). */
    unavailable: string | null;
    retry: () => unknown;
  }

  let { status, unavailable, retry }: Props = $props();
</script>

{#if status.loading}
  <p class="muted mb-3 flex items-center gap-2 text-xs" role="status">
    <Spinner size={12} /> Checking the AI service…
  </p>
{:else if status.failure}
  <LoadError failure={status.failure} label="the AI status" {retry} class="mb-4" />
{:else if unavailable}
  <Card class="mb-4">
    <p class="flex flex-wrap items-center gap-2 text-sm text-amber-700 dark:text-amber-300" role="alert">
      <Icon name="alert" size={16} />
      {unavailable} See <a class="link" href="{base}/settings?tab=ai">Settings</a>.
    </p>
  </Card>
{/if}
