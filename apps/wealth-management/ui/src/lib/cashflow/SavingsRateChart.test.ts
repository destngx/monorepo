import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import SavingsRateChart from './SavingsRateChart.svelte';

afterEach(() => mask.hide());

const flow = (month: string, income: number, expense: number) => ({
  month,
  income,
  expense,
  cashFlow: income - expense,
  savingsRate: income > 0 ? ((income - expense) / income) * 100 : 0,
});

const props = {
  months: [flow('2026-01', 100, 75), flow('2026-02', 0, 40), flow('2026-03', 0, 0)],
  monthsElapsed: 2,
  target: { watch: 10, healthy: 20, target: 30 },
};

const legend = () => screen.getByTestId('savings-legend').textContent?.replace(/\s+/g, ' ').trim();

describe('SavingsRateChart', () => {
  it('names the band and the target in the legend', () => {
    render(SavingsRateChart, { props });
    expect(legend()).toBe('Savings rate Healthy 20-30% Target 30%');
  });

  it('reads out the hovered month, masking the amount', async () => {
    const { container } = render(SavingsRateChart, { props });
    const months = container.querySelectorAll('svg > g');
    await fireEvent.pointerEnter(months[0]);
    expect(legend()).toBe(`Jan Kept 25.0% ${MASK} saved`);
    await fireEvent.pointerEnter(months[1]);
    expect(legend()).toBe('Feb No income');
    await fireEvent.pointerEnter(months[2]);
    expect(legend()).toBe('Mar Not yet');
  });

  it('draws a bar only for elapsed months with income', () => {
    const { container } = render(SavingsRateChart, { props });
    const bars = [...container.querySelectorAll('svg > g')].map((g) => g.querySelectorAll('rect').length);
    expect(bars).toEqual([2, 1, 1]);
  });
});
