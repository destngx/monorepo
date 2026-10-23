import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ fill: vi.fn(), updateBudgetLimit: vi.fn() }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn(), success: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn() }));

import type { BudgetLine } from '$lib/api/types';
import LimitsFillModal from './LimitsFillModal.svelte';

const line = (category: string, limit: number) => ({ category, limit }) as BudgetLine;

describe('LimitsFillModal (WM-FILL-5)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
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

  it('lists the limits read from a sentence, keeps an unmatched one with its note, and saves the rest', async () => {
    endpoints.fill.mockResolvedValue({
      form: 'budget-limits',
      values: {},
      rows: [
        { category: 'Food', limit: 6_000_000, scope: 'month' },
        { category: 'Transport', limit: 2_000_000, scope: 'year' },
        { category: 'Pets', limit: 500_000 },
      ],
      filled: ['rows'],
      notes: [{ field: 'category', label: 'Category', row: 2, message: 'No category matches "Pets".' }],
    });
    endpoints.updateBudgetLimit.mockResolvedValue(undefined);
    const onclose = vi.fn();
    render(LimitsFillModal, {
      open: true,
      month: '2026-10',
      lines: [line('Food', 5_000_000), line('Transport', 1_500_000)],
      onclose,
    });

    await fireEvent.input(screen.getByRole('textbox', { name: 'Describe it' }), {
      target: { value: 'food 6M, transport 2M for every month, pets 500k' },
    });
    await fireEvent.click(screen.getByRole('button', { name: 'Fill' }));

    expect(await screen.findByText('Filled 3 limits. Check them, then save.')).toBeTruthy();
    expect(screen.getAllByText('No category matches "Pets".').length).toBeGreaterThan(0);
    expect(endpoints.fill).toHaveBeenCalledWith(
      expect.objectContaining({ form: 'budget-limits', current: { month: '2026-10' } }),
      expect.any(AbortSignal),
    );

    await fireEvent.click(screen.getByRole('button', { name: 'Save 2 limits' }));
    expect(endpoints.updateBudgetLimit).toHaveBeenNthCalledWith(1, {
      year: 2026,
      category: 'Food',
      month: '2026-10',
      limit: 6_000_000,
    });
    expect(endpoints.updateBudgetLimit).toHaveBeenNthCalledWith(2, {
      year: 2026,
      category: 'Transport',
      month: undefined,
      limit: 2_000_000,
    });
    expect(toastMock.push).toHaveBeenCalledWith('success', 'Saved 2 limits.');
    // The unmatched row is still waiting, so the modal stays open.
    expect(onclose).not.toHaveBeenCalled();
  });
});
