import type { Insight, Recommendation, RecommendationList } from '$lib/api/types';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ recommendations: vi.fn(), setRecommendationStatus: vi.fn() }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));

import RecommendationsPanel from './RecommendationsPanel.svelte';

const item: Recommendation = {
  id: 'rec-0a1b2c3d',
  row: 2,
  created: '2026-10-01',
  source: 'budget-review',
  title: 'Cook at home',
  action: 'Cook twice a week',
  impactVnd: null,
  evidencePath: '/budget',
  evidence: 'Food over its limit',
  status: 'open',
  page: '/budget',
  etag: 'etag-1',
};

const list = (items: Recommendation[]): RecommendationList => ({
  tabExists: true,
  items,
  summary: { open: items.length, snoozed: 0, done: 0, dismissed: 0, openImpactVnd: 0, openWithoutImpact: items.length },
  issues: [],
});

const insight: Insight = {
  summary: 'Food is over its limit.',
  sections: [{ icon: 'wallet', title: 'Food', content: 'Over.', severity: 'warning' }],
  actionItems: [],
  recorded: { created: [item], duplicates: 0, checked: [], ignoredOutcomes: 0 },
};

describe('RecommendationsPanel', () => {
  beforeEach(() => {
    endpoints.recommendations.mockReset();
    endpoints.setRecommendationStatus.mockReset();
  });

  it("lists the page's open items and reloads them after a review", async () => {
    endpoints.recommendations.mockResolvedValueOnce(list([])).mockResolvedValue(list([item]));
    const review = vi.fn().mockResolvedValue(insight);
    render(RecommendationsPanel, { title: 'Recommendations', page: '/budget', review });

    expect(endpoints.recommendations).toHaveBeenCalledWith({ page: '/budget', status: 'open' }, { force: false });
    expect(review).not.toHaveBeenCalled();
    await fireEvent.click(await screen.findByRole('button', { name: /Review/ }));

    expect(review).toHaveBeenCalledWith(true, expect.any(AbortSignal));
    expect(await screen.findByText('Cook twice a week')).toBeTruthy();
    expect(screen.getByText('Food is over its limit.')).toBeTruthy();
    expect(screen.getByText('Recommendations: 1 added.')).toBeTruthy();
    expect(screen.getByText('1 open recommendation')).toBeTruthy();
  });

  it('marks an item done with its etag', async () => {
    endpoints.recommendations.mockResolvedValueOnce(list([item])).mockResolvedValue(list([]));
    endpoints.setRecommendationStatus.mockResolvedValue({ ...item, status: 'done' });
    render(RecommendationsPanel, { title: 'Recommendations', page: '/budget', review: vi.fn() });

    await fireEvent.click(await screen.findByRole('button', { name: /Done/ }));
    expect(endpoints.setRecommendationStatus).toHaveBeenCalledWith('rec-0a1b2c3d', 'etag-1', { status: 'done' });
    expect(await screen.findByText(/reads the figures on this page/)).toBeTruthy();
  });
});
