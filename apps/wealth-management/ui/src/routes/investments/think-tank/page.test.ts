import { fireEvent, render, screen } from '@testing-library/svelte';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { initialState, startState, type ThinkTankState } from '$lib/investments/thinktank';

const run = vi.hoisted(() => ({
  thinkTank: { state: null as unknown as ThinkTankState, running: false, start: vi.fn(), stop: vi.fn() },
}));
vi.mock('$lib/investments/thinktank.svelte', () => run);
vi.mock('$lib/investments/newsStore.svelte', () => ({ newsStore: { reports: [] } }));
vi.mock('$app/paths', () => ({ base: '/ui' }));

import ThinkTankPage from './+page.svelte';

const aiStatus = { configured: true, missingPrompts: [] };

describe('Think Tank page', () => {
  beforeEach(() => {
    run.thinkTank.state = initialState();
    run.thinkTank.start.mockReset();
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
