import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { MarketPulse } from '$lib/api/types';

const endpoints = vi.hoisted(() => ({ marketPulse: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
const toastMock = vi.hoisted(() => ({ push: vi.fn(), error: vi.fn(), success: vi.fn(), info: vi.fn() }));
vi.mock('$lib/state/toasts.svelte', () => ({ toasts: toastMock }));
vi.mock('$app/navigation', () => ({ goto: vi.fn(), invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/paths', () => ({ base: '/ui' }));
vi.mock('$app/state', () => ({ page: { url: new URL('http://127.0.0.1:5173/ui/investments/market?tf=1d') } }));

import MarketPage from './+page.svelte';

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
};

// An invented pulse with no quotes.
const pulse: MarketPulse = {
  timeframe: '1d',
  assets: [],
  topMovers: [],
  correlation: { symbols: [], values: [] },
  capitalFlow: { signal: 'MIXED', summary: 'Flows are split.' },
  scenario: { name: 'Range-bound', regime: 'Neutral', summary: 'Little trend.', action: 'Hold.' },
  index: null,
  sentiment: null,
  notes: [],
  generatedAt: '2026-10-17T03:00:00Z',
};

describe('Market Pulse page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    globalThis.localStorage?.clear();
  });

  it('Given the pulse is still loading When the page renders Then the header controls show at once', () => {
    endpoints.marketPulse.mockReturnValue(new Promise(() => undefined));
    render(MarketPage);

    expect(screen.getByRole('heading', { level: 1, name: 'Market Pulse' })).toBeTruthy();
    expect(screen.getByRole('group', { name: 'Timeframe' })).toBeTruthy();
    expect(screen.getByRole('combobox', { name: 'Auto-refresh interval' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Refresh' })).toBeTruthy();
    expect(screen.getByText('Loading asset quotes…')).toBeTruthy();
    expect(endpoints.marketPulse).toHaveBeenCalledWith('1d', { force: false });
  });

  it('Given the pulse fails When it settles Then it shows the error and a Retry that skips the cache', async () => {
    endpoints.marketPulse.mockRejectedValueOnce(new Error('market data server down'));
    render(MarketPage);
    await flush();

    expect(screen.getByText('Could not load the market pulse')).toBeTruthy();
    expect(screen.getByText('market data server down')).toBeTruthy();
    endpoints.marketPulse.mockResolvedValue(pulse);
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    await flush();

    expect(endpoints.marketPulse).toHaveBeenLastCalledWith('1d', { force: true });
    expect(screen.getByText('Range-bound')).toBeTruthy();
  });

  it('Given a shown pulse When Refresh runs Then it is busy and keeps the pulse until the new one arrives', async () => {
    endpoints.marketPulse.mockResolvedValueOnce(pulse);
    render(MarketPage);
    await flush();
    expect(screen.getByText('Range-bound')).toBeTruthy();

    endpoints.marketPulse.mockReturnValue(new Promise(() => undefined));
    await fireEvent.click(screen.getByRole('button', { name: 'Refresh' }));
    await flush();

    expect(screen.getByRole('button', { name: 'Refreshing…' }).getAttribute('aria-busy')).toBe('true');
    expect(screen.getByText('Range-bound')).toBeTruthy();
    expect(document.querySelector('[data-refreshing]')).toBeTruthy();
  });
});
