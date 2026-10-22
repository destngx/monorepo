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
    /** True while the dialog's action runs: it cannot be closed until the action ends. */
    busy?: boolean;
  }

  let { open, title, onclose, children, footer, wide = false, dismissible = true, busy = false }: Props = $props();
  const closable = $derived(dismissible && !busy);
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
  class="panel m-auto w-[calc(100%-2rem)] p-0 text-inherit shadow-hard-lg backdrop:bg-stone-950/45 {wide
    ? 'max-w-3xl'
    : 'max-w-lg'}"
  {onclose}
  onpointerdown={(event) => (pressedBackdrop = event.target === dialog)}
  onclick={(event) => {
    // The panel fills the dialog edge to edge, so only the backdrop targets the dialog itself.
    // Escape is the keyboard way to dismiss.
    if (closable && pressedBackdrop && event.target === dialog) onclose();
    pressedBackdrop = false;
  }}
  oncancel={(event) => {
    if (!closable) event.preventDefault();
  }}
>
  {#if open}
    <div class="flex max-h-[85vh] flex-col">
      <header class="flex items-center justify-between border-b-[1.5px] border-edge px-5 py-4">
        <h2 class="font-serif text-lg font-semibold">{title}</h2>
        {#if dismissible}
          <button
            type="button"
            class="icon-btn size-8"
            aria-label="Close"
            title="Close"
            disabled={busy}
            onclick={onclose}
          >
            <Icon name="x" size={16} />
          </button>
        {/if}
      </header>
      <div class="overflow-y-auto px-5 py-4" data-modal-body>{@render children()}</div>
      {#if footer}
        <footer class="flex justify-end gap-2 border-t-[1.5px] border-edge px-5 py-3">
          {@render footer()}
        </footer>
      {/if}
    </div>
  {/if}
</dialog>
