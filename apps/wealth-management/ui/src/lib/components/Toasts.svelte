<script lang="ts">
  import { toasts, type ToastKind } from '$lib/state/toasts.svelte';
  import Icon from './Icon.svelte';

  const tones: Record<ToastKind, string> = {
    error: 'border-rose-500/40 text-rose-800 dark:text-rose-200',
    warning: 'border-amber-500/40 text-amber-800 dark:text-amber-200',
    info: 'border-sky-500/40 text-sky-800 dark:text-sky-200',
    success: 'border-emerald-500/40 text-emerald-800 dark:text-emerald-200',
  };
</script>

<div
  class="pointer-events-none fixed right-4 bottom-4 z-50 flex w-[min(24rem,calc(100%-2rem))] flex-col gap-2"
  aria-live="polite"
>
  {#each toasts.items as toast (toast.id)}
    <div
      role={toast.kind === 'error' ? 'alert' : 'status'}
      class="glass pointer-events-auto flex items-start gap-3 border px-4 py-3 text-sm {tones[toast.kind]}"
    >
      <p class="flex-1 break-words">{toast.message}</p>
      <button
        type="button"
        class="opacity-60 hover:opacity-100"
        aria-label="Dismiss"
        onclick={() => toasts.dismiss(toast.id)}
      >
        <Icon name="x" size={16} />
      </button>
    </div>
  {/each}
</div>
