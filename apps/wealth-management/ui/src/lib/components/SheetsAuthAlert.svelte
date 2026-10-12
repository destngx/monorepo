<script lang="ts">
  import { toasts } from '$lib/state/toasts.svelte';
  import Icon from './Icon.svelte';
  import SheetsAuthSteps from './SheetsAuthSteps.svelte';
</script>

<!-- Dedicated re-auth or setup instructions for Google credential failures (WM-UI-4). -->
{#if toasts.sheetsAuthProblem}
  {@const problem = toasts.sheetsAuthProblem}
  <div role="alert" class="glass mb-6 border border-rose-500/40 p-4">
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
        class="opacity-60 hover:opacity-100"
        aria-label="Dismiss"
        onclick={() => (toasts.sheetsAuthProblem = null)}
      >
        <Icon name="x" size={16} />
      </button>
    </div>
  </div>
{/if}
