import { fireEvent, render, screen, within } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { BudgetAdvice, Recommendation, RecommendationList, SuggestedLimit } from '$lib/api/types';

const endpoints = vi.hoisted(() => ({
  budgetAdvice: vi.fn(),
  updateBudgetLimit: vi.fn(),
  recommendations: vi.fn(),
  setRecommendationStatus: vi.fn(),
}));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$app/navigation', () => ({ invalidateAll: vi.fn() }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

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

const openList = (items: Recommendation[]): RecommendationList => ({
  tabExists: true,
  items,
  summary: { open: items.length, snoozed: 0, done: 0, dismissed: 0, openImpactVnd: 0, openWithoutImpact: items.length },
  issues: [],
});

const tracked: Recommendation = {
  id: 'rec-0a1b2c3d',
  row: 2,
  created: '2026-10-19',
  source: 'budget-review',
  title: 'Cook at home',
  action: 'Cook twice a week',
  impactVnd: null,
  evidencePath: '/budget',
  evidence: 'Dining rising',
  status: 'open',
  page: '/budget',
  etag: 'etag-1',
};

async function showAdvice(value: BudgetAdvice) {
  endpoints.budgetAdvice.mockResolvedValue(value);
  render(BudgetAdvisor);
  await fireEvent.click(screen.getByRole('button', { name: /Get advice/ }));
  await screen.findByText('Spending is on plan.');
}

describe('BudgetAdvisor', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    endpoints.recommendations.mockResolvedValue(openList([]));
  });

  it('lists the budget actions being tracked before any advice is asked for (WM-BUD-11)', async () => {
    endpoints.recommendations.mockResolvedValue(openList([tracked]));
    render(BudgetAdvisor);

    const section = screen.getByRole('region', { name: "Actions you're tracking" });
    expect(await within(section).findByText('Cook twice a week')).toBeTruthy();
    expect(endpoints.budgetAdvice).not.toHaveBeenCalled();
    expect(endpoints.recommendations).toHaveBeenCalledWith({ page: '/budget', status: 'open' }, { force: false });
  });

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

  it('says what it filed and re-reads the tracked actions (WM-BUD-11)', async () => {
    endpoints.recommendations.mockResolvedValueOnce(openList([])).mockResolvedValue(openList([tracked]));
    await showAdvice(advice({ recorded: { created: [tracked], duplicates: 0, checked: [], ignoredOutcomes: 0 } }));

    expect(screen.getByText('Recommendations: 1 added.')).toBeTruthy();
    const section = screen.getByRole('region', { name: "Actions you're tracking" });
    expect(await within(section).findByText('Cook twice a week')).toBeTruthy();
    expect(endpoints.recommendations).toHaveBeenLastCalledWith({ page: '/budget', status: 'open' }, { force: true });
  });
});
