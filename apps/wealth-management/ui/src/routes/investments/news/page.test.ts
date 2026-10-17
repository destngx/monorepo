import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import { DEFAULT_NEWS_TOPICS, type AIStatus } from '$lib/api/types';

const endpoints = vi.hoisted(() => ({ newsSentiment: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
vi.mock('$lib/investments/newsStore.svelte', () => ({
  newsStore: { reports: [], upsert: vi.fn(), remove: vi.fn(), clear: vi.fn() },
}));
const navigation = vi.hoisted(() => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', () => navigation);
vi.mock('$app/paths', () => ({ base: '/ui' }));

import NewsPage from './+page.svelte';

const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const ready = settled({ configured: true, missingPrompts: [] } as unknown as AIStatus);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};
const topic = DEFAULT_NEWS_TOPICS[0];

describe('News Scanner page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('Given the AI status is still loading When the page renders Then the topics can be scanned at once', () => {
    render(NewsPage, { data: { aiStatus: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'News Scanner' })).toBeTruthy();
    expect(screen.getByText(/Checking the AI service/)).toBeTruthy();
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Scan all defaults' }).disabled).toBe(false);
  });

  it('Given the AI routes are missing When the status settles to null Then the reason shows and scans are disabled', async () => {
    render(NewsPage, { data: { aiStatus: settled(null) } as never });
    await flush();

    expect(screen.getByText(/AI is not available on the engine/)).toBeTruthy();
    expect(screen.getByRole('link', { name: 'Settings' }).getAttribute('href')).toBe('/ui/settings?tab=ai');
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Scan all defaults' }).disabled).toBe(true);
  });

  it('Given the AI status request fails When it settles Then it shows the error with a Retry', async () => {
    render(NewsPage, { data: { aiStatus: defer(Promise.reject(new TypeError('fetch failed'))) } as never });
    await flush();

    expect(screen.getByText('Could not load the AI status')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(navigation.invalidateAll).toHaveBeenCalled();
  });

  it('Given a topic scan is running When it waits Then the topic and Scan all are busy and its card shows progress', async () => {
    endpoints.newsSentiment.mockReturnValue(new Promise(() => undefined));
    render(NewsPage, { data: { aiStatus: ready } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: topic }));
    await flush();

    expect(screen.getByRole('button', { name: topic }).getAttribute('aria-busy')).toBe('true');
    // Scan all and the card's own button.
    const scanning = screen.getAllByRole('button', { name: 'Scanning…' });
    expect(scanning).toHaveLength(2);
    expect(scanning.every((button) => button.getAttribute('aria-busy') === 'true')).toBe(true);
    expect(screen.getByText(`Loading the ${topic} news scan…`)).toBeTruthy();
  });

  it('Given a topic scan fails When it settles Then its card shows why and a Retry that scans again', async () => {
    endpoints.newsSentiment.mockRejectedValueOnce(new Error('search provider down'));
    render(NewsPage, { data: { aiStatus: ready } as never });
    await flush();

    await fireEvent.click(screen.getByRole('button', { name: topic }));
    await flush();

    expect(screen.getByRole('alert').textContent).toContain('search provider down');
    endpoints.newsSentiment.mockReturnValue(new Promise(() => undefined));
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(endpoints.newsSentiment).toHaveBeenCalledTimes(2);
  });
});
