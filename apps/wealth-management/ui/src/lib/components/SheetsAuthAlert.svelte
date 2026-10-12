<script lang="ts">
  import { toasts } from '$lib/state/toasts.svelte';
  import Icon from './Icon.svelte';
</script>

<!-- Dedicated re-auth instructions for Google credential failures (WM-UI-4). -->
{#if toasts.sheetsAuthProblem}
  <div role="alert" class="glass mb-6 border border-rose-500/40 p-4">
    <div class="flex items-start gap-3">
      <Icon name="alert" class="mt-0.5 text-rose-600 dark:text-rose-400" />
      <div class="flex-1 text-sm">
        <p class="font-semibold text-rose-700 dark:text-rose-300">Google Sheets access needs attention</p>
        <p class="muted mt-1 break-words">{toasts.sheetsAuthProblem}</p>
        <p class="mt-2">
          Re-authorise by running <code class="rounded bg-slate-900/10 px-1.5 py-0.5 dark:bg-white/10"
            >pnpm run auth:sheets</code
          > in the repository, then restart the engine. OAuth apps in Testing mode issue refresh tokens that expire after
          7 days.
        </p>
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
