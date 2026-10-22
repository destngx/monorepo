import { fireEvent, render, screen } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import Modal from './Modal.svelte';

const body = createRawSnippet(() => ({ render: () => '<p>Modal body</p>' }));
const form = createRawSnippet(() => ({
  render: () => '<form><label>Name <input id="name" /></label><input id="note" /></form>',
}));

function renderModal(props: { dismissible?: boolean; busy?: boolean; confirmClose?: string } = {}) {
  const onclose = vi.fn();
  render(Modal, { open: true, title: 'Example', onclose, children: body, ...props });
  const dialog = screen.getByRole('dialog', { hidden: true });
  return { onclose, dialog, content: screen.getByText('Modal body') };
}

describe('Modal', () => {
  beforeEach(() => {
    // jsdom has no modal dialogs; the Modal only needs them to toggle.
    if (!('showModal' in HTMLDialogElement.prototype)) {
      Object.assign(HTMLDialogElement.prototype, {
        showModal(this: HTMLDialogElement) {
          this.open = true;
        },
        close(this: HTMLDialogElement) {
          this.open = false;
        },
      });
    }
  });

  it('closes when the backdrop around the panel is clicked', async () => {
    const { onclose, dialog } = renderModal();
    await fireEvent.pointerDown(dialog);
    await fireEvent.click(dialog);
    expect(onclose).toHaveBeenCalledTimes(1);
  });

  it('stays open for clicks inside the panel', async () => {
    const { onclose, content } = renderModal();
    await fireEvent.pointerDown(content);
    await fireEvent.click(content);
    expect(onclose).not.toHaveBeenCalled();
  });

  it('stays open when a press starts inside the panel and is released on the backdrop', async () => {
    // Selecting text in a field and letting go outside the panel must not discard the form.
    const { onclose, dialog, content } = renderModal();
    await fireEvent.pointerDown(content);
    await fireEvent.click(dialog);
    expect(onclose).not.toHaveBeenCalled();
  });

  it('starts a form at its first field, not the Close button', () => {
    render(Modal, { open: true, title: 'Example', onclose: vi.fn(), children: form });
    expect(document.activeElement).toBe(screen.getByLabelText('Name'));
  });

  it('cannot be closed while its action runs', async () => {
    const { onclose, dialog } = renderModal({ busy: true });
    await fireEvent.pointerDown(dialog);
    await fireEvent.click(dialog);
    const escape = new Event('cancel', { cancelable: true });
    dialog.dispatchEvent(escape);
    expect(escape.defaultPrevented).toBe(true);
    expect(screen.getByRole('button', { name: 'Close', hidden: true })).toHaveProperty('disabled', true);
    expect(onclose).not.toHaveBeenCalled();
  });

  it('ignores backdrop clicks when it is not dismissible', async () => {
    const { onclose, dialog } = renderModal({ dismissible: false });
    await fireEvent.pointerDown(dialog);
    await fireEvent.click(dialog);
    expect(onclose).not.toHaveBeenCalled();
  });

  it('lets Escape close it natively when nothing needs confirming', () => {
    const { dialog } = renderModal();
    const escape = new Event('cancel', { cancelable: true });
    dialog.dispatchEvent(escape);
    expect(escape.defaultPrevented).toBe(false);
  });

  it('asks before closing when it has unsaved work, and stays open on Cancel', async () => {
    const confirm = vi.fn(() => false);
    vi.stubGlobal('confirm', confirm);
    const { onclose, dialog } = renderModal({ confirmClose: 'Discard the photos?' });

    dialog.dispatchEvent(new Event('cancel', { cancelable: true }));
    await fireEvent.pointerDown(dialog);
    await fireEvent.click(dialog);
    await fireEvent.click(screen.getByRole('button', { name: 'Close', hidden: true }));
    expect(confirm).toHaveBeenCalledTimes(3);
    expect(confirm).toHaveBeenCalledWith('Discard the photos?');
    expect(onclose).not.toHaveBeenCalled();

    confirm.mockReturnValue(true);
    await fireEvent.click(screen.getByRole('button', { name: 'Close', hidden: true }));
    expect(onclose).toHaveBeenCalledTimes(1);
    vi.unstubAllGlobals();
  });
});
