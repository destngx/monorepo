<script lang="ts">
  import { sheetsAuth } from '$lib/state/toasts.svelte';
  import { Icon } from '@libs/ui-kit';
  import SheetsAuthSteps from './SheetsAuthSteps.svelte';
</script>

<!-- Dedicated re-auth or setup instructions for Google credential failures (WM-UI-4). -->
{#if sheetsAuth.problem}
  {@const problem = sheetsAuth.problem}
  <div role="alert" class="panel mb-6 border border-rose-500/40 p-4">
    <div class="flex items-start gap-3">
      <Icon name="alert" class="mt-0.5 text-rose-600 dark:text-rose-400" />
      <div class="flex-1 text-sm">
        <p class="font-semibold text-rose-700 dark:text-rose-300">
          {problem.kind === 'missing' ? 'Google Sheets is not configured' : 'Google Sheets access needs attention'}
        </p>
        <p class="muted mt-1 break-words">{problem.message}</p>
        <SheetsAuthSteps kind={problem.kind} class="mt-2" />
      </div>
      <button
        type="button"
        class="icon-btn"
        aria-label="Dismiss"
        title="Dismiss"
        onclick={() => (sheetsAuth.problem = null)}
      >
        <Icon name="x" size={16} />
      </button>
    </div>
  </div>
{/if}
