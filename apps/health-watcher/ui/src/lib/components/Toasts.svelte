<script lang="ts">
  import { toasts, type ToastKind } from '$lib/state/toasts.svelte';
  import Icon, { type IconName } from './Icon.svelte';

  const icons: Record<ToastKind, IconName> = { error: 'alertCircle', warning: 'alert', info: 'info', success: 'check' };
  const tones: Record<ToastKind, string> = {
    error: 'border-l-danger-600 text-danger-800 dark:border-l-danger-400 dark:text-danger-200',
    warning: 'border-l-warn-500 text-warn-800 dark:border-l-warn-400 dark:text-warn-200',
    info: 'border-l-info-500 text-info-800 dark:border-l-info-400 dark:text-info-200',
    success: 'border-l-accent-600 text-accent-800 dark:border-l-accent-400 dark:text-accent-200',
  };
</script>

<div
  class="pointer-events-none fixed right-4 bottom-24 z-50 lg:bottom-4 flex w-[min(24rem,calc(100%-2rem))] flex-col gap-2"
  aria-live="polite"
>
  {#each toasts.items as toast (toast.id)}
    <div
      role={toast.kind === 'error' ? 'alert' : 'status'}
      class="panel pointer-events-auto flex items-start gap-3 border-l-8 px-4 py-3 text-sm motion-safe:animate-[toast-in_160ms_ease-out] {tones[
        toast.kind
      ]}"
      data-kind={toast.kind}
    >
      <Icon name={icons[toast.kind]} size={16} class="mt-0.5" />
      <p class="flex-1 break-words">{toast.message}</p>
      <button
        type="button"
        class="icon-btn -my-1 -mr-2 size-6 text-current"
        aria-label="Dismiss"
        title="Dismiss"
        onclick={() => toasts.dismiss(toast.id)}
      >
        <Icon name="x" size={14} />
      </button>
    </div>
  {/each}
</div>

<style>
  @keyframes -global-toast-in {
    from {
      opacity: 0;
      translate: 0 0.5rem;
    }
  }
</style>
