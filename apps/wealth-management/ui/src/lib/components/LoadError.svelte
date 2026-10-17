<script lang="ts">
  import type { LoadFailure } from '$lib/api/load';
  import { STATUS } from '$lib/status';
  import Button from './Button.svelte';
  import Icon from './Icon.svelte';

  // A section that failed to load, in its place (not the whole-page error):
  // what failed, why, what to do and a Retry. Google Sheets credential
  // problems point to the re-auth steps the layout shows at the top.
  interface Props {
    failure: LoadFailure;
    /** What failed to load, e.g. "net worth". */
    label: string;
    retry?: () => unknown;
    /** A one-line strip above data that is still shown (a failed refresh). */
    compact?: boolean;
    class?: string;
  }

  let { failure, label, retry, compact = false, class: className = '' }: Props = $props();

  const style = STATUS.warning;
  const advice = $derived.by(() => {
    if (failure.sheetsAuth) return 'Google Sheets needs attention: follow the steps at the top of the page.';
    if (failure.kind === 'unreachable') return 'Check that the engine is running, then retry.';
    return failure.hint;
  });
</script>

{#if compact}
  <div
    role="alert"
    class="flex flex-wrap items-center gap-x-3 gap-y-1 rounded-xl border px-3 py-1.5 text-xs {style.box} {className}"
    data-load-error={failure.kind}
  >
    <Icon name={style.icon} size={14} />
    <span class="min-w-0 flex-1 break-words">Could not refresh {label}: {failure.message}</span>
    {#if retry}<Button size="sm" onclick={() => retry?.()}>Retry</Button>{/if}
  </div>
{:else}
  <div
    role="alert"
    class="rounded-2xl border p-4 text-sm backdrop-blur-xl {style.box} {className}"
    data-load-error={failure.kind}
  >
    <div class="flex items-start gap-3">
      <Icon name={style.icon} class="mt-0.5" />
      <div class="min-w-0 flex-1">
        <p class="font-medium">Could not load {label}</p>
        <p class="mt-0.5 text-xs break-words opacity-90">{failure.message}</p>
        {#if advice}<p class="mt-1 text-xs opacity-90">{advice}</p>{/if}
      </div>
      {#if retry}<Button size="sm" icon="refresh" onclick={() => retry?.()}>Retry</Button>{/if}
    </div>
  </div>
{/if}
