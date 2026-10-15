import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import BarChart from './BarChart.svelte';

afterEach(() => mask.hide());

const props = {
  labels: ['Jan', 'Feb'],
  series: [
    { name: 'Refund', color: '#0f0', values: [120_000, 80_000] },
    { name: 'Missed', color: '#f00', values: [0, 15_000] },
  ],
};

const legend = () => screen.getByTestId('bar-legend').textContent?.replace(/\s+/g, ' ').trim();

describe('BarChart', () => {
  it('shows the hovered column label and every series value', async () => {
    mask.toggle();
    const { container } = render(BarChart, props);
    expect(legend()).toBe('Refund Missed');

    const columns = container.querySelectorAll('svg > g');
    await fireEvent.pointerEnter(columns[1]);
    expect(legend()).toBe('Feb Refund 80K Missed 15K');

    await fireEvent.pointerLeave(columns[1]);
    expect(legend()).toBe('Refund Missed');
  });

  it('masks hovered values until revealed', async () => {
    const { container } = render(BarChart, props);
    await fireEvent.pointerEnter(container.querySelectorAll('svg > g')[0]);
    expect(legend()).toBe(`Jan Refund ${MASK} Missed ${MASK}`);
  });
});
