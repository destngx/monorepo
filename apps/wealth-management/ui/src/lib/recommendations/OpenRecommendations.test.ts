import type { Recommendation, RecommendationList } from '$lib/api/types';
import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ recommendations: vi.fn(), setRecommendationStatus: vi.fn() }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), info: vi.fn(), success: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));

import OpenRecommendations from './OpenRecommendations.svelte';

const item: Recommendation = {
  id: 'rec-0a1b2c3d',
  row: 2,
  created: '2026-10-01',
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

const list = (items: Recommendation[]): RecommendationList => ({
  tabExists: true,
  items,
  summary: { open: items.length, snoozed: 0, done: 0, dismissed: 0, openImpactVnd: 0, openWithoutImpact: items.length },
  issues: [],
});

describe('OpenRecommendations', () => {
  beforeEach(() => {
    endpoints.recommendations.mockReset();
    endpoints.setRecommendationStatus.mockReset();
  });

  it("lists the page's open items", async () => {
    endpoints.recommendations.mockResolvedValue(list([item]));
    render(OpenRecommendations, { page: '/budget', empty: 'Nothing yet.' });

    expect(await screen.findByText('Cook twice a week')).toBeTruthy();
    expect(screen.getByText('1 open recommendation')).toBeTruthy();
    expect(endpoints.recommendations).toHaveBeenCalledWith({ page: '/budget', status: 'open' }, { force: false });
  });

  it('says so when nothing is open', async () => {
    endpoints.recommendations.mockResolvedValue(list([]));
    render(OpenRecommendations, { page: '/budget', empty: 'Nothing yet.' });

    expect(await screen.findByText('Nothing yet.')).toBeTruthy();
  });

  it('re-reads the list, bypassing the cache, when its refresh signal changes', async () => {
    endpoints.recommendations.mockResolvedValueOnce(list([])).mockResolvedValue(list([item]));
    const { rerender } = render(OpenRecommendations, { page: '/budget', empty: 'Nothing yet.', refresh: 0 });
    await screen.findByText('Nothing yet.');

    await rerender({ page: '/budget', empty: 'Nothing yet.', refresh: 1 });

    expect(await screen.findByText('Cook twice a week')).toBeTruthy();
    expect(endpoints.recommendations).toHaveBeenLastCalledWith({ page: '/budget', status: 'open' }, { force: true });
  });

  it('marks an item done and re-reads the stored state', async () => {
    endpoints.recommendations.mockResolvedValueOnce(list([item])).mockResolvedValue(list([]));
    endpoints.setRecommendationStatus.mockResolvedValue({ ...item, status: 'done' });
    render(OpenRecommendations, { page: '/budget', empty: 'Nothing yet.' });

    await fireEvent.click(await screen.findByRole('button', { name: /Done/ }));

    expect(endpoints.setRecommendationStatus).toHaveBeenCalledWith('rec-0a1b2c3d', 'etag-1', { status: 'done' });
    expect(await screen.findByText('Nothing yet.')).toBeTruthy();
  });
});
