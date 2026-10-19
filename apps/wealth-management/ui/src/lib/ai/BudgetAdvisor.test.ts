import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { BudgetAdvice, SuggestedLimit } from '$lib/api/types';

const endpoints = vi.hoisted(() => ({ budgetAdvice: vi.fn(), updateBudgetLimit: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn() }));

import BudgetAdvisor from './BudgetAdvisor.svelte';

const limit = (overrides: Partial<SuggestedLimit>): SuggestedLimit => ({
  category: 'Dining',
  nature: 'flexible',
  currentLimit: 1_000_000,
  averageSpend: 2_000_000,
  yearToDateAverage: 1_333_333,
  trend: 'rising',
  pattern: 'steady',
  recommendedLimit: 2_000_000,
  suggestedLimit: 2_000_000,
  direction: 'increase',
  rationale: 'Savings are on track, so this want keeps its usual level.',
  ...overrides,
});

const advice = (overrides: Partial<BudgetAdvice> = {}): BudgetAdvice => ({
  month: '2026-10',
  briefing: 'Spending is on plan.',
  savingsCheck: {
    averageIncome: 20_000_000,
    goalNeed: 4_000_000,
    required: 6_000_000,
    usualSpend: 7_000_000,
    onTrack: true,
    recommendedSpend: 7_000_000,
  },
  suggestedLimits: [
    limit({}),
    limit({
      category: 'Gym',
      nature: undefined,
      trend: 'steady',
      currentLimit: 2_000_000,
      suggestedLimit: 550_000,
      direction: 'lower',
    }),
  ],
  projection: [],
  alerts: [],
  goalImpact: [],
  forecastedSavings: 0,
  unusualVelocity: [],
  ...overrides,
});

async function showAdvice(value: BudgetAdvice) {
  endpoints.budgetAdvice.mockResolvedValue(value);
  render(BudgetAdvisor);
  await fireEvent.click(screen.getByRole('button', { name: /Get advice/ }));
  await screen.findByText('Spending is on plan.');
}

describe('BudgetAdvisor', () => {
  beforeEach(() => vi.clearAllMocks());

  it('shows each suggestion as a move in words with what the category is', async () => {
    await showAdvice(advice());

    const [, dining, gym] = screen.getAllByRole('row');
    expect(within(dining).getByText('Increase')).toBeTruthy();
    expect(within(dining).getByText('Want · Rising')).toBeTruthy();
    expect(within(gym).getByText('Lower')).toBeTruthy();
    expect(within(gym).getByText('Untagged · Steady')).toBeTruthy();
  });

  it('says whether savings leave room for wants', async () => {
    await showAdvice(advice());
    expect(screen.getByTestId('savings-check').textContent).toContain('Savings on track');
  });

  it('says needs are never cut when savings are short', async () => {
    const short = advice();
    short.savingsCheck = { ...short.savingsCheck, onTrack: false };
    await showAdvice(short);

    expect(screen.getByTestId('savings-check').textContent).toMatch(/Savings short.*needs are\s+never cut/s);
  });
});
