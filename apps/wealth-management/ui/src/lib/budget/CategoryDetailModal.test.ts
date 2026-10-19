import { render, screen } from '@testing-library/svelte';
import { afterEach, beforeAll, describe, expect, it, vi } from 'vitest';
import type { BudgetLine, SpendingWindows, Transaction } from '$lib/api/types';
import { mask } from '$lib/state/mask.svelte';
import CategoryDetailModal from './CategoryDetailModal.svelte';

// Invented figures.
const line: BudgetLine = {
  category: 'Food & Groceries',
  categoryType: 'expense',
  limit: 4_000_000,
  spent: 2_000_000,
  remaining: 2_000_000,
  percentUsed: 50,
  status: 'healthy',
  yearlyLimit: 0,
  yearlySpent: 0,
  yearlyRemaining: 0,
  burnOutDate: null,
  splits: [{ tag: 'eating-out', spent: 800_000, percentOfSpent: 40 }],
};

const tx = (id: string, date: string, payee: string, payment: number, tags: string[] = []) =>
  ({ id, date, payee, category: 'Food & Groceries', payment, deposit: null, tags }) as unknown as Transaction;

const category = (spent: number, split: number) => ({
  category: 'food & groceries',
  spent,
  percentOfSpent: 50,
  monthlyAverage: 0,
  previousSpent: 0,
  changePercent: null,
  splits: [{ tag: 'eating-out', spent: split, percentOfSpent: 0 }],
});

const spending = {
  windows: [
    { days: 30, categories: [category(2_500_000, 900_000)] },
    { days: 90, categories: [category(6_000_000, 2_400_000)] },
  ],
} as unknown as SpendingWindows;

const cells = () =>
  [...screen.getByTestId('detail-splits').querySelectorAll('tbody td')].map((cell) =>
    cell.textContent?.replace(/\s+/g, ' ').trim(),
  );

describe('CategoryDetailModal', () => {
  beforeAll(() => {
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
  afterEach(() => mask.hide());

  it('Given a category with a split tag When its detail opens Then eating out shows for the month, 30 days and 3 months', () => {
    mask.toggle();
    render(CategoryDetailModal, {
      line,
      month: '2026-10',
      transactions: [
        tx('1', '2026-10-03', 'Market', 1_200_000),
        tx('2', '2026-10-04', 'Noodle shop', 800_000, ['card-cashback', 'Eating-Out']),
      ],
      spending,
      onclose: vi.fn(),
      onmove: vi.fn(),
    });

    // 2.4M over 90 days is about 810.667 a month.
    expect(cells()).toEqual(['#eating-out', '800.000 ₫ (40%)', '900.000 ₫', '810.667 ₫']);
    const rows = screen.getAllByRole('listitem').map((row) => row.textContent?.replace(/\s+/g, ' ').trim());
    expect(rows.find((row) => row?.includes('Noodle shop'))).toContain('#eating-out');
    expect(rows.find((row) => row?.includes('Market'))).not.toContain('#eating-out');
    expect(screen.getByTestId('bar-legend').textContent).toMatch(/Rest of the category\s*#eating-out/);
  });

  it('Given the spending windows have not loaded When the detail opens Then the month share still shows', () => {
    render(CategoryDetailModal, { line, month: '2026-10', transactions: [], onclose: vi.fn(), onmove: vi.fn() });
    expect(cells()).toEqual(['#eating-out', '•••••• (40%)', '-', '-']);
  });
});
