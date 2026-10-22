<script lang="ts">
  import { toasts, type ToastKind } from '$lib/state/toasts.svelte';
  import Icon, { type IconName } from './Icon.svelte';

  const icons: Record<ToastKind, IconName> = { error: 'alertCircle', warning: 'alert', info: 'info', success: 'check' };
  const tones: Record<ToastKind, string> = {
    error: 'border-l-[6px] border-l-rose-600 text-rose-800 dark:text-rose-200',
    warning: 'border-l-[6px] border-l-amber-500 text-amber-800 dark:text-amber-200',
    info: 'border-l-[6px] border-l-sky-600 text-sky-800 dark:text-sky-200',
    success: 'border-l-[6px] border-l-emerald-600 text-emerald-800 dark:text-emerald-200',
  };
</script>

<div
  class="pointer-events-none fixed right-4 bottom-24 z-50 lg:bottom-4 flex w-[min(24rem,calc(100%-2rem))] flex-col gap-2"
  aria-live="polite"
>
  {#each toasts.items as toast (toast.id)}
    <div
      role={toast.kind === 'error' ? 'alert' : 'status'}
      class="panel pointer-events-auto flex items-start gap-3 px-4 py-3 text-sm motion-safe:animate-[toast-in_160ms_ease-out] {tones[
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
