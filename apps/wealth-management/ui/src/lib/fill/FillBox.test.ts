import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ fill: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));

import { ApiError } from '$lib/api/client';
import FillBox from './FillBox.svelte';

describe('FillBox', () => {
  beforeEach(() => {
    endpoints.fill.mockReset();
  });

  it('sends the text and what the form holds, then hands over the answer', async () => {
    const answer = {
      form: 'goal',
      values: { name: 'Trip' },
      filled: ['name'],
      notes: [{ field: 'target', label: 'Target', message: '"a lot" is not an amount.' }],
    };
    endpoints.fill.mockResolvedValue(answer);
    const onfill = vi.fn();
    render(FillBox, { form: 'goal', current: () => ({ status: 'active' }), onfill });

    const box = screen.getByRole('textbox', { name: 'Describe it' });
    expect(box.getAttribute('placeholder')).toContain('save 50M');
    await fireEvent.input(box, { target: { value: 'save for a trip' } });
    await fireEvent.click(screen.getByRole('button', { name: 'Fill' }));

    expect(endpoints.fill).toHaveBeenCalledWith(
      { form: 'goal', text: 'save for a trip', images: undefined, current: { status: 'active' } },
      expect.any(AbortSignal),
    );
    expect(onfill).toHaveBeenCalledWith(answer);
    expect(await screen.findByText('Filled 1 field. Check them, then save.')).toBeTruthy();
    expect(screen.getByText('Target:')).toBeTruthy();
    expect(screen.getByText('"a lot" is not an amount.', { exact: false })).toBeTruthy();
  });

  it('cannot fill without text and offers no images on a text-only form', () => {
    render(FillBox, { form: 'budget-move', onfill: vi.fn() });
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Fill' }).disabled).toBe(true);
    expect(screen.queryByRole('button', { name: 'Add image' })).toBeNull();
  });

  it('shows why a fill failed', async () => {
    endpoints.fill.mockRejectedValue(new ApiError(503, 'AI is not configured: set AI_BASE_URL and AI_MODEL'));
    const onfill = vi.fn();
    render(FillBox, { form: 'recurring', onfill, initialText: 'Netflix 260k' });
    expect(screen.getByRole('button', { name: 'Add image' })).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Fill' }));
    expect(await screen.findByText('AI is not configured: set AI_BASE_URL and AI_MODEL')).toBeTruthy();
    expect(onfill).not.toHaveBeenCalled();
  });
});

describe('FillBox: a chat fill (WM-FILL-7)', () => {
  beforeEach(() => {
    endpoints.fill.mockReset();
  });

  it('applies a fill the chat stashed for its form once, with the chat text', async () => {
    const { pendingFills } = await import('./pending.svelte');
    const result = { form: 'goal' as const, values: { name: 'Trip' }, filled: ['name'], notes: [] };
    pendingFills.set({ form: 'goal', title: 'Goal', page: '/accounts/goals', text: 'save for a trip', result });
    const onfill = vi.fn();
    render(FillBox, { form: 'goal', onfill });

    await vi.waitFor(() => expect(onfill).toHaveBeenCalledWith(result));
    expect(screen.getByRole<HTMLTextAreaElement>('textbox', { name: 'Describe it' }).value).toBe('save for a trip');
    expect(screen.getByText('Filled 1 field. Check them, then save.')).toBeTruthy();
    expect(pendingFills.waiting('goal')).toBe(false);
    expect(endpoints.fill).not.toHaveBeenCalled();
  });

  it('leaves a fill for another form waiting', async () => {
    const { pendingFills } = await import('./pending.svelte');
    const result = { form: 'card' as const, values: {}, filled: [], notes: [] };
    pendingFills.set({ form: 'card', title: 'Credit card', page: '/accounts/credit-cards', text: 'x', result });
    const onfill = vi.fn();
    render(FillBox, { form: 'goal', onfill });

    await Promise.resolve();
    expect(onfill).not.toHaveBeenCalled();
    expect(pendingFills.take('card')).not.toBeNull();
  });
});
