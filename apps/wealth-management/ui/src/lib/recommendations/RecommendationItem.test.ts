import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import type { Recommendation } from '$lib/api/types';
import RecommendationItem from './RecommendationItem.svelte';

const item: Recommendation = {
  id: 'rec-0a1b2c3d',
  row: 2,
  created: '2026-10-01',
  source: 'budget-review',
  title: 'Cook at home',
  action: 'Cook twice a week',
  impactVnd: 1_200_000,
  evidencePath: '/budget',
  evidence: 'Food over its limit',
  status: 'open',
  page: '/budget',
  etag: 'e',
};

describe('RecommendationItem', () => {
  it('offers done, dismiss and two snoozes on an open item', async () => {
    const onstatus = vi.fn();
    render(RecommendationItem, { item, onstatus, showStatus: true });

    expect(screen.getByText('Cook twice a week')).toBeTruthy();
    expect(screen.getByText('Open').closest('[data-status]')?.getAttribute('data-status')).toBe('warning');
    expect(screen.getByRole('link', { name: 'Evidence' }).getAttribute('href')).toContain('/budget');
    expect(screen.queryByRole('button', { name: /Reopen/ })).toBeNull();

    await fireEvent.click(screen.getByRole('button', { name: /Done/ }));
    expect(onstatus).toHaveBeenLastCalledWith(item, { status: 'done' });
    await fireEvent.click(screen.getByRole('button', { name: /Snooze 30 days/ }));
    expect(onstatus).toHaveBeenLastCalledWith(item, { status: 'snoozed', snoozeDays: 30 });
    await fireEvent.click(screen.getByRole('button', { name: /Dismiss/ }));
    expect(onstatus).toHaveBeenLastCalledWith(item, { status: 'dismissed' });
  });

  it('reopens a closed item and shows its outcome', async () => {
    const onstatus = vi.fn();
    render(RecommendationItem, {
      item: {
        ...item,
        status: 'done',
        statusChanged: '2026-10-10',
        outcome: 'Met: Food is under the limit',
        impactVnd: null,
      },
      onstatus,
    });
    expect(screen.getByText('Met: Food is under the limit')).toBeTruthy();
    expect(screen.getByText('No estimate')).toBeTruthy();
    expect(screen.queryByRole('button', { name: /Done/ })).toBeNull();
    await fireEvent.click(screen.getByRole('button', { name: /Reopen/ }));
    expect(onstatus).toHaveBeenCalledWith(expect.objectContaining({ status: 'done' }), { status: 'open' });
  });
});
