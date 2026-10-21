import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import AIExplain from './AIExplain.svelte';

describe('AIExplain', () => {
  it('asks the AI only when opened and regenerates on demand', async () => {
    const load = vi.fn().mockResolvedValue('**Shopping** rule applied');
    render(AIExplain, { title: 'Cashback estimate', label: 'Explain the cashback estimate', icon: 'info', load });
    expect(load).not.toHaveBeenCalled();

    await fireEvent.click(screen.getByRole('button', { name: 'Explain the cashback estimate' }));
    expect(await screen.findByText('Shopping')).toBeTruthy();
    expect(screen.getByRole('dialog', { name: 'Cashback estimate' })).toBeTruthy();
    expect(load).toHaveBeenCalledWith(false, expect.any(AbortSignal));

    await fireEvent.click(screen.getByRole('button', { name: 'Regenerate' }));
    expect(load).toHaveBeenLastCalledWith(true, expect.any(AbortSignal));
  });

  it('closes on Escape and keeps the answer for the next open', async () => {
    const load = vi.fn().mockResolvedValue('Explained');
    render(AIExplain, { title: 'Insight', label: 'Explain', load });
    const button = screen.getByRole('button', { name: 'Explain' });

    await fireEvent.click(button);
    await screen.findByText('Explained');
    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(screen.queryByRole('dialog')).toBeNull();

    await fireEvent.click(button);
    expect(screen.getByText('Explained')).toBeTruthy();
    expect(load).toHaveBeenCalledTimes(1);
  });

  it('renders the popover at the end of the body so no card can cover it', async () => {
    const { container } = render(AIExplain, { title: 'Insight', label: 'Explain', load: () => Promise.resolve('x') });
    await fireEvent.click(screen.getByRole('button', { name: 'Explain' }));
    const dialog = screen.getByRole('dialog', { name: 'Insight' });
    expect(dialog.parentElement).toBe(document.body);
    expect(container.contains(dialog)).toBe(false);
    expect(dialog.classList.contains('fixed')).toBe(true);
  });

  it('moves focus into the popover, and back to its button when closed', async () => {
    render(AIExplain, { title: 'Insight', label: 'Explain', load: () => Promise.resolve('x') });
    const button = screen.getByRole('button', { name: 'Explain' });
    await fireEvent.click(button);
    expect(document.activeElement).toBe(screen.getByRole('dialog', { name: 'Insight' }));

    await fireEvent.keyDown(window, { key: 'Escape' });
    expect(document.activeElement).toBe(button);
  });

  it('closes on a press outside it, but not on a press inside it', async () => {
    render(AIExplain, { title: 'Insight', label: 'Explain', load: () => Promise.resolve('Explained') });
    await fireEvent.click(screen.getByRole('button', { name: 'Explain' }));
    await fireEvent.pointerDown(await screen.findByText('Explained'));
    expect(screen.getByRole('dialog', { name: 'Insight' })).toBeTruthy();

    await fireEvent.pointerDown(document.body);
    expect(screen.queryByRole('dialog')).toBeNull();
  });
});
