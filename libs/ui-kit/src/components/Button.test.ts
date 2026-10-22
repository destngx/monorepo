import { fireEvent, render, screen } from '@testing-library/svelte';
import { createRawSnippet, tick } from 'svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { toasts } from '../state/toasts.svelte';
import Button from './Button.svelte';

const text = (value: string) => createRawSnippet(() => ({ render: () => `<span>${value}</span>` }));

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('Button', () => {
  afterEach(() => {
    toasts.items = [];
  });

  it('Given an async action When clicked Then it is busy with a spinner until the action settles', async () => {
    let finish!: () => void;
    const onclick = vi.fn(() => new Promise<void>((done) => (finish = done)));
    render(Button, { onclick, children: text('Save'), busyText: 'Saving…' });
    const button = screen.getByRole('button');

    await fireEvent.click(button);
    expect(button.getAttribute('aria-busy')).toBe('true');
    expect(button.getAttribute('aria-disabled')).toBe('true');
    expect(button.querySelector('[data-spinner]')).toBeTruthy();
    expect(button.textContent).toContain('Saving…');

    await fireEvent.click(button);
    expect(onclick).toHaveBeenCalledOnce();

    finish();
    await flush();
    expect(button.getAttribute('aria-busy')).toBeNull();
    expect(button.textContent).toContain('Save');
  });

  it('Given a success message When the action finishes Then a success toast confirms it', async () => {
    render(Button, { onclick: () => Promise.resolve(), success: 'Saved.', children: text('Save') });
    await fireEvent.click(screen.getByRole('button'));
    await flush();
    expect(toasts.items.map((toast) => [toast.kind, toast.message])).toEqual([['success', 'Saved.']]);
  });

  it('Given an action that fails When clicked Then the error is toasted, not swallowed', async () => {
    render(Button, {
      onclick: () => Promise.reject(new Error('sheet locked')),
      errorContext: 'Could not save',
      children: text('Save'),
    });
    await fireEvent.click(screen.getByRole('button'));
    await flush();
    expect(toasts.items.map((toast) => [toast.kind, toast.message])).toEqual([
      ['error', 'Could not save: sheet locked'],
    ]);
    expect(screen.getByRole('button').getAttribute('aria-busy')).toBeNull();
  });

  it('Given the busy prop When rendered Then it shows busy without a click', () => {
    render(Button, { busy: true, children: text('Sync') });
    expect(screen.getByRole('button').getAttribute('aria-busy')).toBe('true');
  });

  it('Given an icon-only button When rendered Then the label is its accessible name and tooltip', () => {
    render(Button, { icon: 'refresh', label: 'Refresh prices', size: 'icon' });
    const button = screen.getByRole('button', { name: 'Refresh prices' });
    expect(button.getAttribute('title')).toBe('Refresh prices');
  });

  it('Given a sync handler When clicked Then it runs without a busy state', async () => {
    const onclick = vi.fn();
    render(Button, { onclick, variant: 'primary', children: text('Open') });
    await fireEvent.click(screen.getByRole('button'));
    expect(onclick).toHaveBeenCalledOnce();
    expect(screen.getByRole('button').getAttribute('aria-busy')).toBeNull();
    expect(screen.getByRole('button').className).toContain('btn-primary');
  });
});
