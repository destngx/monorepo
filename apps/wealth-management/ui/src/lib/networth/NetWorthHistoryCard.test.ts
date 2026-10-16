import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import NetWorthHistoryCard from './NetWorthHistoryCard.svelte';
import { netWorthHistory } from './history.testdata';

const text = (id: string) => screen.getByTestId(id).textContent?.replace(/\s+/g, ' ').trim() ?? '';

describe('NetWorthHistoryCard', () => {
  afterEach(() => mask.hide());

  it('leads with net worth and the change over the range, masked by default', () => {
    render(NetWorthHistoryCard, { history: netWorthHistory() });

    expect(text('networth-headline')).toBe(`Net worth ${MASK}; ${MASK} (+20.0%) over the last year, from an estimate.`);
    expect(text('networth-estimated')).toContain('1 of 4 months are estimated from transactions');
  });

  it('marks estimated months by shape and dash, and milestones with a diamond', () => {
    const { container } = render(NetWorthHistoryCard, { history: netWorthHistory() });

    expect(container.querySelectorAll('circle[data-estimated="true"]')).toHaveLength(1);
    expect(container.querySelectorAll('line[data-dashed="true"]')).toHaveLength(1);
    expect(container.querySelectorAll('[data-milestone="2026-10"]')).toHaveLength(1);
    expect(screen.getByLabelText('Milestones').textContent).toContain('All-time high');
  });

  it('reads out the hovered month with its change split', async () => {
    mask.toggle();
    const { container } = render(NetWorthHistoryCard, { history: netWorthHistory() });

    await fireEvent.pointerEnter(container.querySelector('g[data-month="2026-10"]')!);

    const readout = text('networth-readout');
    expect(readout).toContain('Oct 2026');
    expect(readout).toContain('Snapshot');
    expect(readout).toContain('Savings 12.000.000 ₫ · Market and other 3.000.000 ₫');
    expect(readout).toContain('◆ Crossed 100.000.000 ₫');
  });

  it('shows a table with the source in words, newest first', async () => {
    render(NetWorthHistoryCard, { history: netWorthHistory() });

    await fireEvent.click(screen.getByText('Show as a table'));

    const rows = [...screen.getByTestId('networth-table').querySelectorAll('tbody tr')].map((row) =>
      row.querySelectorAll('td')[1].textContent?.trim(),
    );
    expect(rows).toEqual(['Today', 'Snapshot', 'Snapshot', 'Estimated']);
  });

  it('says so when there is not enough history', () => {
    render(NetWorthHistoryCard, { history: netWorthHistory({ points: [], milestones: [], estimated: 0 }) });

    expect(screen.getByText('Not enough history to draw a trend yet.')).toBeTruthy();
  });
});
