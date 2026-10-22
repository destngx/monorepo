import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import MenuButton from './MenuButton.svelte';

describe('MenuButton', () => {
  it('opens its items, runs the one picked and closes', async () => {
    const files = vi.fn();
    render(MenuButton, {
      label: 'Add more',
      items: [
        { label: 'Add files', onselect: files },
        { label: 'Take photos', onselect: vi.fn() },
      ],
    });
    expect(screen.queryByRole('menu')).toBeNull();

    await fireEvent.click(screen.getByRole('button', { name: /Add more/ }));
    expect(screen.getAllByRole('menuitem').map((b) => b.textContent?.trim())).toEqual(['Add files', 'Take photos']);

    await fireEvent.click(screen.getByRole('menuitem', { name: 'Add files' }));
    expect(files).toHaveBeenCalledOnce();
    expect(screen.queryByRole('menu')).toBeNull();
  });

  it('closes on Escape and on a click outside', async () => {
    render(MenuButton, { label: 'Add more', items: [{ label: 'Add files', onselect: vi.fn() }] });
    const toggle = screen.getByRole('button', { name: /Add more/ });

    await fireEvent.click(toggle);
    await fireEvent.keyDown(screen.getByRole('menu'), { key: 'Escape' });
    expect(screen.queryByRole('menu')).toBeNull();

    await fireEvent.click(toggle);
    await fireEvent.pointerDown(document.body);
    expect(screen.queryByRole('menu')).toBeNull();
  });
});
