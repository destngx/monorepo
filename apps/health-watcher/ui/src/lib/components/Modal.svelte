<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from './Icon.svelte';

  interface Props {
    open: boolean;
    title: string;
    onclose: () => void;
    children: Snippet;
    footer?: Snippet;
    wide?: boolean;
    /** False for blocking dialogs: no close button and Escape does nothing. */
    dismissible?: boolean;
    /** Asked before Escape or the close button closes it; closing waits for OK. */
    confirmClose?: string;
  }

  let { open, title, onclose, children, footer, wide = false, dismissible = true, confirmClose }: Props = $props();

  function requestClose() {
    if (!confirmClose || confirm(confirmClose)) onclose();
  }
  let dialog: HTMLDialogElement | undefined = $state();

  $effect(() => {
    if (!dialog) return;
    if (open && !dialog.open) dialog.showModal();
    if (!open && dialog.open) dialog.close();
  });
</script>

<dialog
  bind:this={dialog}
  class="panel-strong m-auto w-[calc(100%-2rem)] p-0 text-inherit backdrop:bg-ink-950/50 {wide
    ? 'max-w-3xl'
    : 'max-w-lg'}"
  {onclose}
  oncancel={(event) => {
    if (!dismissible || confirmClose) event.preventDefault();
    if (dismissible && confirmClose) requestClose();
  }}
>
  {#if open}
    <div class="flex max-h-[85vh] flex-col">
      <header class="flex items-center justify-between border-b-2 border-edge px-5 py-4">
        <h2 class="heading text-xl">{title}</h2>
        {#if dismissible}
          <button type="button" class="icon-btn size-8" aria-label="Close" title="Close" onclick={requestClose}>
            <Icon name="x" size={16} />
          </button>
        {/if}
      </header>
      <div class="overflow-y-auto px-5 py-4">{@render children()}</div>
      {#if footer}
        <footer class="flex justify-end gap-2 border-t-2 border-edge px-5 py-3">
          {@render footer()}
        </footer>
      {/if}
    </div>
  {/if}
</dialog>
