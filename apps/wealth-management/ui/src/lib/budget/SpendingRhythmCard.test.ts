import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import type { SpendingWindow, SpendingWindows } from '$lib/api/types';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import SpendingRhythmCard from './SpendingRhythmCard.svelte';

// Invented figures.
const window = (days: number, overrides: Partial<SpendingWindow> = {}): SpendingWindow => ({
  days,
  from: '2026-09-20',
  to: '2026-10-19',
  spent: 9_000_000,
  fixed: 6_000_000,
  dayToDay: 3_000_000,
  dailyAverage: 100_000,
  monthlyAverage: 9_120_000,
  previous: { from: '2026-08-21', to: '2026-09-19', spent: 8_000_000, dayToDay: 2_500_000, dailyAverage: 83_333 },
  dailyAverageChangePercent: 20,
  categories: [
    {
      category: 'Rent',
      nature: 'fixed',
      spent: 6_000_000,
      percentOfSpent: 66.67,
      monthlyAverage: 6_080_000,
      previousSpent: 6_000_000,
      changePercent: 0,
    },
    {
      category: 'Food',
      nature: 'essential',
      spent: 3_000_000,
      percentOfSpent: 33.33,
      monthlyAverage: 3_040_000,
      previousSpent: 0,
      changePercent: null,
      splits: [{ tag: 'eating-out', spent: 1_200_000, percentOfSpent: 40 }],
    },
  ],
  ...overrides,
});

const spending = (overrides: Partial<SpendingWindows> = {}): SpendingWindows => ({
  asOf: '2026-10-19',
  today: { date: '2026-10-19', spent: 250_000, dayToDay: 250_000, typicalDay: 100_000, changePercent: 150 },
  windows: [window(30), window(90, { spent: 24_000_000, dailyAverage: 80_000, categories: [] })],
  daily: Array.from({ length: 90 }, (_, index) => ({ date: `2026-07-${index}`, fixed: 0, dayToDay: 1 })),
  recommendations: [
    { kind: 'faster', tone: 'warning', current: 100_000, usual: 80_000, amount: 608_000, changePercent: 25 },
    {
      kind: 'split_rising',
      tone: 'warning',
      category: 'Food',
      tag: 'eating-out',
      nature: 'essential',
      current: 1_200_000,
      usual: 600_000,
      amount: 608_000,
      changePercent: 100,
    },
    {
      kind: 'category_rising',
      tone: 'info',
      category: 'Food',
      nature: 'essential',
      current: 3_000_000,
      usual: 2_000_000,
      amount: 1_013_333,
      changePercent: 50,
    },
  ],
  ...overrides,
});

const text = (id: string) => screen.getByTestId(id).textContent?.replace(/\s+/g, ' ').trim();

describe('SpendingRhythmCard', () => {
  afterEach(() => mask.hide());

  it('Given the windows When the card renders Then today, the last 30 days and 3 months show with their change', () => {
    mask.toggle();
    render(SpendingRhythmCard, { spending: spending() });

    const stats = text('spending-stats');
    expect(stats).toContain('Today 250.000 ₫ +150.0% vs a typical day of 100.000 ₫');
    expect(stats).toContain('Last 30 days 9.000.000 ₫ +20.0% 100.000 ₫ a day, day to day');
    expect(stats).toContain('Last 3 months 24.000.000 ₫');
  });

  it('Given advice When it shows Then each has a word for its tone and a need is never told to be cut', () => {
    mask.toggle();
    render(SpendingRhythmCard, { spending: spending() });

    const advice = text('spending-advice');
    expect(advice).toContain('Act Day-to-day spending is up 25% on your 3-month average');
    expect(advice).toContain(
      'Act #eating-out in Food is up 100%: 1.200.000 ₫ in the last 30 days against a usual 600.000 ₫',
    );
    expect(advice).toContain('Look Food is up 50%');
    expect(advice).toContain('do not cut it blindly');
  });

  it('Given the window switch When the 3 months are chosen Then the table and summary follow it', async () => {
    mask.toggle();
    render(SpendingRhythmCard, { spending: spending() });
    expect(text('spending-categories')).toMatch(/Food Need of which #eating-out 1\.200\.000 ₫ \(40%\)/);
    expect(text('spending-categories')).toContain('new');

    await fireEvent.click(screen.getByRole('button', { name: 'Last 3 months' }));

    expect(screen.getByRole('button', { name: 'Last 3 months' }).getAttribute('aria-pressed')).toBe('true');
    expect(screen.queryByTestId('spending-categories')).toBeNull();
    expect(text('spending-window-summary')).toContain('24.000.000 ₫ spent');
  });

  it('Given no advice When spending is usual Then it says nothing stands out, and amounts stay masked', () => {
    render(SpendingRhythmCard, { spending: spending({ recommendations: [] }) });
    expect(text('spending-advice')).toBe(
      'Spending is in line with your usual: nothing stands out in the last 30 days.',
    );
    expect(text('spending-stats')).toContain(MASK);
  });
});
