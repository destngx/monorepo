import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { Briefing } from '$lib/api/types';

const endpoints = vi.hoisted(() => ({ briefing: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import BriefingCard from './BriefingCard.svelte';

// Invented figures.
const briefing: Briefing = {
  briefing: 'Spending is on plan.',
  alerts: [],
  patterns: [],
  facts: { netWorth: 800, assets: 900, liabilities: 100, income: 100, expense: 60, cashFlow: 40, savingsRate: 0.4 },
};

describe('BriefingCard', () => {
  beforeEach(() => {
    endpoints.briefing.mockReset();
  });

  it('loads the briefing on open without a click, reusing a cached one', async () => {
    endpoints.briefing.mockResolvedValue(briefing);
    render(BriefingCard, { compact: true });

    expect(await screen.findByText('Spending is on plan.')).toBeTruthy();
    expect(screen.queryByRole('button', { name: /Generate briefing/ })).toBeNull();
    expect(endpoints.briefing).toHaveBeenCalledOnce();
    expect(endpoints.briefing.mock.calls[0][0]).toMatchObject({ force: false });
  });

  it('regenerates on demand, skipping the cache', async () => {
    endpoints.briefing.mockResolvedValue(briefing);
    render(BriefingCard, { compact: true });

    await fireEvent.click(await screen.findByRole('button', { name: 'Regenerate briefing' }));

    expect(endpoints.briefing).toHaveBeenCalledTimes(2);
    expect(endpoints.briefing.mock.calls[1][0]).toMatchObject({ force: true });
  });

  it('shows the failure with a retry when the automatic run fails', async () => {
    endpoints.briefing.mockRejectedValue(new Error('AI service is down'));
    render(BriefingCard, { compact: true });

    expect(await screen.findByText('AI service is down')).toBeTruthy();
    expect(screen.getByRole('button', { name: /Retry/ })).toBeTruthy();
  });

  it('waits for the page data before asking, showing as loading meanwhile', async () => {
    endpoints.briefing.mockResolvedValue(briefing);
    let pageLoaded!: () => void;
    const after = new Promise<void>((resolve) => (pageLoaded = resolve));
    render(BriefingCard, { compact: true, after });

    await Promise.resolve();
    expect(endpoints.briefing).not.toHaveBeenCalled();
    expect(screen.getByRole('status', { name: 'Generating' })).toBeTruthy();
    expect(screen.queryByRole('button', { name: /Generate briefing/ })).toBeNull();

    pageLoaded();
    expect(await screen.findByText('Spending is on plan.')).toBeTruthy();
    expect(endpoints.briefing).toHaveBeenCalledOnce();
  });

  it('still asks when the page data failed', async () => {
    endpoints.briefing.mockResolvedValue(briefing);
    render(BriefingCard, { compact: true, after: Promise.reject(new Error('sheet down')) });

    expect(await screen.findByText('Spending is on plan.')).toBeTruthy();
  });
});
