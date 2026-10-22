import { fireEvent, render, screen } from '@testing-library/svelte';
import { createRawSnippet } from 'svelte';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import Modal from './Modal.svelte';

const children = createRawSnippet(() => ({ render: () => '<p>QR code</p>' }));

describe('Modal', () => {
  // jsdom has no modal dialogs.
  beforeAll(() => {
    HTMLDialogElement.prototype.showModal = function (this: HTMLDialogElement) {
      this.open = true;
    };
    HTMLDialogElement.prototype.close = function (this: HTMLDialogElement) {
      this.open = false;
    };
  });
  afterEach(() => vi.unstubAllGlobals());

  it('asks before closing when confirmClose is set, and stays open on Cancel', async () => {
    const confirm = vi.fn(() => false);
    vi.stubGlobal('confirm', confirm);
    const onclose = vi.fn();
    const { container } = render(Modal, { open: true, title: 'Phone', onclose, children, confirmClose: 'Close it?' });

    await fireEvent.click(screen.getByRole('button', { name: 'Close', hidden: true }));
    expect(confirm).toHaveBeenCalledWith('Close it?');
    expect(onclose).not.toHaveBeenCalled();

    const escape = new Event('cancel', { cancelable: true });
    container.querySelector('dialog')!.dispatchEvent(escape);
    expect(escape.defaultPrevented).toBe(true);
    expect(onclose).not.toHaveBeenCalled();

    confirm.mockReturnValue(true);
    await fireEvent.click(screen.getByRole('button', { name: 'Close', hidden: true }));
    expect(onclose).toHaveBeenCalledOnce();
  });

  it('closes without asking when confirmClose is not set', async () => {
    const confirm = vi.fn(() => true);
    vi.stubGlobal('confirm', confirm);
    const onclose = vi.fn();
    render(Modal, { open: true, title: 'Phone', onclose, children });

    await fireEvent.click(screen.getByRole('button', { name: 'Close', hidden: true }));
    expect(confirm).not.toHaveBeenCalled();
    expect(onclose).toHaveBeenCalledOnce();
  });
});
