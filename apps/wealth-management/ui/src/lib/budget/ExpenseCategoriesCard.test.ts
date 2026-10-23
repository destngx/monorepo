import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it, vi } from 'vitest';
import type { BudgetLine } from '$lib/api/types';
import { mask } from '$lib/state/mask.svelte';
import ExpenseCategoriesCard from './ExpenseCategoriesCard.svelte';

const line = (overrides: Partial<BudgetLine>): BudgetLine => ({
  category: 'Groceries',
  categoryType: 'expense',
  limit: 2_000_000,
  spent: 1_350_000,
  remaining: 650_000,
  percentUsed: 67.5,
  status: 'healthy',
  yearlyLimit: 0,
  yearlySpent: 0,
  yearlyRemaining: 0,
  burnOutDate: null,
  ...overrides,
});

describe('ExpenseCategoriesCard', () => {
  afterEach(() => mask.hide());

  it("shows a category's split tag share (WM-CAT-3)", () => {
    mask.toggle();
    render(ExpenseCategoriesCard, {
      lines: [
        line({ splits: [{ tag: 'eating-out', spent: 350_000, percentOfSpent: 25.93 }] }),
        line({ category: 'Rent' }),
      ],
      onselect: vi.fn(),
      onedit: vi.fn(),
    });

    const splits = screen.getAllByTestId('splits');
    expect(splits).toHaveLength(1);
    expect(splits[0].textContent).toMatch(/of which\s+#eating-out\s+350\.000 ₫ \(26%\)/);
  });

  it('Given more than eight categories When it renders Then the fullest eight show and the rest open on demand', async () => {
    const lines = Array.from({ length: 11 }, (_, i) => line({ category: `Category ${i + 1}` }));
    render(ExpenseCategoriesCard, { lines, onselect: vi.fn(), onedit: vi.fn() });

    expect(screen.getAllByRole('button', { name: /^Edit limit/ })).toHaveLength(8);
    expect(screen.queryByText('Category 9')).toBeNull();

    await fireEvent.click(screen.getByRole('button', { name: 'Show all 11' }));
    expect(screen.getAllByRole('button', { name: /^Edit limit/ })).toHaveLength(11);
    expect(screen.getByRole('button', { name: 'Show fewer' })).toBeTruthy();
  });
});
