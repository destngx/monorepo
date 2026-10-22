<script lang="ts">
  import type { Snippet } from 'svelte';
  import Icon from './Icon.svelte';

  // A native <dialog> opened with showModal(), so the page behind is inert
  // and focus returns to the opener when it closes.
  interface Props {
    open: boolean;
    title: string;
    onclose: () => void;
    children: Snippet;
    footer?: Snippet;
    wide?: boolean;
    /** False for blocking dialogs: no close button, and Escape and the backdrop do nothing. */
    dismissible?: boolean;
    /** True while the dialog's action runs: it cannot be closed until the action ends. */
    busy?: boolean;
    /** Asked before Escape, the backdrop or the close button closes it; closing waits for OK. */
    confirmClose?: string;
  }

  let {
    open,
    title,
    onclose,
    children,
    footer,
    wide = false,
    dismissible = true,
    busy = false,
    confirmClose,
  }: Props = $props();
  const closable = $derived(dismissible && !busy);

  function requestClose() {
    if (!confirmClose || confirm(confirmClose)) onclose();
  }
  const FIRST_FIELD =
    '[data-autofocus], [data-modal-body] :is(input:not([type="hidden"]), select, textarea):not(:disabled)';

  let dialog: HTMLDialogElement | undefined = $state();
  // Where the press began: a press that starts in the panel (selecting text in a field)
  // and ends on the backdrop is not an outside click.
  let pressedBackdrop = false;

  $effect(() => {
    if (!dialog) return;
    if (open && !dialog.open) {
      dialog.showModal();
      // showModal focuses the first control, the header's Close button; a form starts at its first field,
      // or the one marked data-autofocus.
      // Not on touch screens, where focusing a field opens the keyboard over the form.
      if (!globalThis.matchMedia?.('(pointer: coarse)').matches)
        dialog.querySelector<HTMLElement>(FIRST_FIELD)?.focus();
    }
    if (!open && dialog.open) dialog.close();
  });
</script>

<dialog
  bind:this={dialog}
  class="panel-strong m-auto w-[calc(100%-2rem)] p-0 text-inherit backdrop:bg-ink-950/50 {wide
    ? 'max-w-3xl'
    : 'max-w-lg'}"
  {onclose}
  onpointerdown={(event) => (pressedBackdrop = event.target === dialog)}
  onclick={(event) => {
    // The panel fills the dialog edge to edge, so only the backdrop targets the dialog itself.
    // Escape is the keyboard way to dismiss.
    if (closable && pressedBackdrop && event.target === dialog) requestClose();
    pressedBackdrop = false;
  }}
  oncancel={(event) => {
    // With a confirmation, Escape closes through requestClose so Cancel can stop it.
    if (!closable || confirmClose) event.preventDefault();
    if (closable && confirmClose) requestClose();
  }}
>
  {#if open}
    <div class="flex max-h-[85vh] flex-col">
      <header class="flex items-center justify-between border-b-2 border-edge px-5 py-4">
        <h2 class="heading text-xl">{title}</h2>
        {#if dismissible}
          <button
            type="button"
            class="icon-btn size-8"
            aria-label="Close"
            title="Close"
            disabled={busy}
            onclick={requestClose}
          >
            <Icon name="x" size={16} />
          </button>
        {/if}
      </header>
      <div class="overflow-y-auto px-5 py-4" data-modal-body>{@render children()}</div>
      {#if footer}
        <footer class="flex justify-end gap-2 border-t-2 border-edge px-5 py-3">
          {@render footer()}
        </footer>
      {/if}
    </div>
  {/if}
</dialog>
