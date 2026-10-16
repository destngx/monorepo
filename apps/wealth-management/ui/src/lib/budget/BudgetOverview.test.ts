import { render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import type { BudgetReport } from '$lib/api/types';
import { mask } from '$lib/state/mask.svelte';
import BudgetOverview from './BudgetOverview.svelte';

const report = (overrides: Partial<BudgetReport> = {}): BudgetReport => ({
  month: '2026-10',
  daysInMonth: 31,
  daysElapsed: 17,
  daysRemaining: 14,
  totalLimit: 10_000_000,
  totalSpent: 6_000_000,
  totalRemaining: 4_000_000,
  percentUsed: 60,
  dailyBurn: 352_941,
  projectedSpend: 10_941_176,
  status: 'healthy',
  lines: [],
  ...overrides,
});

describe('BudgetOverview', () => {
  afterEach(() => mask.hide());

  it('answers "Am I within budget?" first, then the pace as evidence', () => {
    mask.toggle();
    const { container } = render(BudgetOverview, { report: report() });
    expect(screen.getByText('Am I within budget?')).toBeTruthy();
    expect(screen.getByRole('heading', { level: 2 }).textContent).toContain('6.000.000');
    expect(container.querySelector('section')?.getAttribute('data-status')).toBe('good');
    expect(screen.getByTestId('budget-pace').textContent).toContain('over the limit');
  });

  it('turns critical when the budget bursts', () => {
    const { container } = render(BudgetOverview, {
      report: report({ totalSpent: 12_000_000, percentUsed: 120, status: 'urgent' }),
    });
    expect(container.querySelector('section')?.getAttribute('data-status')).toBe('critical');
    expect(screen.getByTestId('budget-pace').textContent).toContain('Over the expense budget by');
  });
});
