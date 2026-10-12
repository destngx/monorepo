<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    open: boolean;
    title: string;
    onclose: () => void;
    children: Snippet;
    footer?: Snippet;
    wide?: boolean;
  }

  let { open, title, onclose, children, footer, wide = false }: Props = $props();
  let dialog: HTMLDialogElement | undefined = $state();

  $effect(() => {
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  });
</script>

<dialog
  bind:this={dialog}
  class="glass m-auto w-[calc(100%-2rem)] p-0 text-inherit backdrop:bg-slate-950/40 backdrop:backdrop-blur-sm {wide
    ? 'max-w-3xl'
    : 'max-w-lg'}"
  {onclose}
>
  {#if open}
    <div class="flex max-h-[85vh] flex-col">
      <header class="flex items-center justify-between border-b border-slate-900/10 px-5 py-4 dark:border-white/10">
        <h2 class="font-semibold">{title}</h2>
        <button type="button" class="btn-ghost px-2 py-1" aria-label="Close" onclick={onclose}>✕</button>
      </header>
      <div class="overflow-y-auto px-5 py-4">{@render children()}</div>
      {#if footer}
        <footer class="flex justify-end gap-2 border-t border-slate-900/10 px-5 py-3 dark:border-white/10">
          {@render footer()}
        </footer>
      {/if}
    </div>
  {/if}
</dialog>
