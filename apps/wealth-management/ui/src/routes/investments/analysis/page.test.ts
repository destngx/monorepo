import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const endpoints = vi.hoisted(() => ({ marketAnalysis: vi.fn(), tickerInsight: vi.fn() }));
vi.mock('$lib/api/endpoints', () => ({ api: endpoints }));
const navigation = vi.hoisted(() => ({ goto: vi.fn(), invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', () => navigation);
vi.mock('$app/paths', () => ({ base: '/ui' }));
const appState = vi.hoisted(() => ({ page: { url: new URL('http://127.0.0.1:5173/ui/investments/analysis') } }));
vi.mock('$app/state', () => appState);

import AnalysisPage from './+page.svelte';

const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
  await new Promise((resolve) => setTimeout(resolve, 0));
};
const open = (query = '') => {
  appState.page.url = new URL(`http://127.0.0.1:5173/ui/investments/analysis${query}`);
  return render(AnalysisPage);
};

describe('Ticker Analysis page', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('Given no symbol When the page opens Then it asks for one without calling the engine', () => {
    open();

    expect(screen.getByText('Pick a ticker')).toBeTruthy();
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Analyse' }).disabled).toBe(true);
    expect(endpoints.marketAnalysis).not.toHaveBeenCalled();
  });

  it('Given a symbol is loading When the page renders Then the form shows at once and Analyse is busy', async () => {
    endpoints.marketAnalysis.mockReturnValue(new Promise(() => undefined));
    open('?symbol=FPT&tf=1d');
    await flush();

    expect(screen.getByLabelText<HTMLInputElement>('Symbol').value).toBe('FPT');
    expect(screen.getByRole('group', { name: 'Quick picks' })).toBeTruthy();
    expect(screen.getByRole('button', { name: 'Analysing…' }).getAttribute('aria-busy')).toBe('true');
    expect(screen.getByText('Loading the FPT analysis…')).toBeTruthy();
  });

  it('Given the analysis fails When it settles Then it explains the timeframe and Retry asks again past the cache', async () => {
    endpoints.marketAnalysis.mockRejectedValueOnce(new Error('no hourly candles'));
    open('?symbol=SJC&tf=1h');
    await flush();

    expect(screen.getByText('Could not load the SJC analysis on 1H')).toBeTruthy();
    expect(screen.getByText(/SJC gold is only published daily/)).toBeTruthy();
    endpoints.marketAnalysis.mockReturnValue(new Promise(() => undefined));
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));

    expect(endpoints.marketAnalysis).toHaveBeenLastCalledWith('SJC', '1h', { force: true });
  });

  it('Given an invalid symbol When typed Then the field is marked invalid and Analyse stays disabled', async () => {
    open();

    await fireEvent.input(screen.getByLabelText('Symbol'), { target: { value: '!!!' } });

    expect(screen.getByLabelText('Symbol').getAttribute('aria-invalid')).toBe('true');
    expect(screen.getByRole<HTMLButtonElement>('button', { name: 'Analyse' }).disabled).toBe(true);
  });
});
