import { fireEvent, render, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { defer, settled, type Settled, type Streamed } from '$lib/api/load';
import type { AIStatus } from '$lib/api/types';
import { initialState, startState, type ThinkTankState } from '$lib/investments/thinktank';
import { MASK_TOKEN_LABEL } from '$lib/components/maskTokens';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';

const run = vi.hoisted(() => ({
  thinkTank: { state: null as unknown as ThinkTankState, running: false, start: vi.fn(), stop: vi.fn() },
}));
vi.mock('$lib/investments/thinktank.svelte', () => run);
vi.mock('$lib/investments/newsStore.svelte', () => ({ newsStore: { reports: [] } }));
vi.mock('$app/paths', () => ({ base: '/ui' }));
const navigation = vi.hoisted(() => ({ invalidateAll: vi.fn(() => Promise.resolve()) }));
vi.mock('$app/navigation', () => navigation);
// The follow-up chat is not under test.
vi.mock('$lib/api/endpoints', () => ({ api: new Proxy({}, { get: () => () => new Promise(() => undefined) }) }));

import ThinkTankPage from './+page.svelte';

const aiStatus = settled({ configured: true, missingPrompts: [] } as unknown as AIStatus);
const never = <T>(): Streamed<T> => new Promise<Settled<T>>(() => undefined);
const flush = async () => {
  for (let i = 0; i < 5; i++) await tick();
};

describe('Think Tank page', () => {
  beforeEach(() => {
    run.thinkTank.state = initialState();
    run.thinkTank.start.mockReset();
    run.thinkTank.running = false;
    mask.hide();
  });

  it('Given the AI status is still loading When the page renders Then the controls are usable at once', async () => {
    render(ThinkTankPage, { data: { aiStatus: never() } as never });

    expect(screen.getByRole('heading', { level: 1, name: 'Think Tank' })).toBeTruthy();
    expect(screen.getByText(/Checking the AI service/)).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: /Run analysis/ }));
    expect(run.thinkTank.start).toHaveBeenCalled();
  });

  it('Given AI is not configured When the status arrives Then the reason shows and Run is disabled', async () => {
    render(ThinkTankPage, {
      data: { aiStatus: settled({ configured: false, missingPrompts: [] } as unknown as AIStatus) } as never,
    });
    await flush();

    expect(screen.getByText(/AI is not configured/)).toBeTruthy();
    expect(screen.getByRole<HTMLButtonElement>('button', { name: /Run analysis/ }).disabled).toBe(true);
  });

  it('Given the AI status request fails When it settles Then it shows the error with a Retry', async () => {
    render(ThinkTankPage, { data: { aiStatus: defer(Promise.reject(new TypeError('fetch failed'))) } as never });
    await flush();

    expect(screen.getByText('Could not load the AI status')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: 'Retry' }));
    expect(navigation.invalidateAll).toHaveBeenCalled();
  });

  it('Given a run in progress When the page shows it Then Run is busy, Stop is offered and the phase is named', () => {
    run.thinkTank.running = true;
    const state = startState();
    run.thinkTank.state = { ...state, phases: { ...state.phases, context: { status: 'running' } } };
    render(ThinkTankPage, { data: { aiStatus } as never });

    const busy = screen.getByRole('button', { name: /Analysing/ });
    expect(busy.getAttribute('aria-busy')).toBe('true');
    expect(screen.getByRole('button', { name: 'Stop' })).toBeTruthy();
    expect(screen.getByText(/Gathering your portfolio/)).toBeTruthy();
  });

  it('Given commands that quote amounts When masked Then the amounts are clickable mask tokens', () => {
    run.thinkTank.state = {
      ...startState(),
      actions: {
        name: 'Actions',
        commands: [{ action: 'ALLOCATE', target: 'FUND-A', rationale: 'Move 5,000,000 VND from cash.' }],
      },
    };
    render(ThinkTankPage, { data: { aiStatus } as never });

    expect(screen.queryByText(/5,000,000/)).toBeNull();
    const token = screen.getByRole('button', { name: MASK_TOKEN_LABEL });
    expect(token.textContent).toBe(MASK);
    expect(screen.getByText(/Move/).textContent).not.toContain('5,000,000');
  });

  it('Given no symbols When running Then it analyses the holdings', async () => {
    render(ThinkTankPage, { data: { aiStatus } as never });

    expect(screen.getByText(/Leave empty to analyse your current holdings/)).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: /Run analysis/ }));
    expect(run.thinkTank.start).toHaveBeenCalledWith([], []);
  });

  it('Given typed symbols When running Then only those are sent', async () => {
    render(ThinkTankPage, { data: { aiStatus } as never });

    await fireEvent.input(screen.getByLabelText(/Symbols/), { target: { value: 'fpt, vcb' } });
    expect(screen.getByText('Analyses FPT, VCB.')).toBeTruthy();
    await fireEvent.click(screen.getByRole('button', { name: /Run analysis/ }));
    expect(run.thinkTank.start).toHaveBeenCalledWith([], ['FPT', 'VCB']);
  });

  it('Given a run that recorded recommendations When it shows the actions Then it says what went to the list', () => {
    run.thinkTank.state = {
      ...startState(),
      context: {
        accounts: [],
        moneyLent: { owedToYou: 0, youOwe: 0, open: [] },
        recentTransactions: [],
        budget: { income: 0, expenses: 0 },
        searches: null,
        searchContext: '',
        focus: { source: 'holdings', symbols: ['BTC', 'FUND-A'] },
      },
      actions: {
        name: 'Actions',
        commands: [],
        recorded: { created: [], duplicates: 2, checked: [], ignoredOutcomes: 0 },
      },
    };
    render(ThinkTankPage, { data: { aiStatus } as never });

    expect(screen.getByText(/Your holdings: BTC, FUND-A/)).toBeTruthy();
    expect(screen.getByText(/Recommendations: 2 already on the list\./)).toBeTruthy();
    expect(screen.getByRole('link', { name: 'See the recommendations' }).getAttribute('href')).toBe(
      '/ui/recommendations?page=%2Finvestments%2Fthink-tank',
    );
  });
});
