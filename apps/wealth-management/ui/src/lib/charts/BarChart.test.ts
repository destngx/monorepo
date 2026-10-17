import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { MASK_TOKEN_LABEL } from '$lib/components/maskTokens';
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

    await fireEvent.pointerLeave(container.firstElementChild as HTMLElement);
    expect(legend()).toBe('Refund Missed');
  });

  it('Given a hovered column When the pointer moves from the chart onto the legend Then the readout stays', async () => {
    mask.toggle();
    const { container } = render(BarChart, props);

    await fireEvent.pointerEnter(container.querySelectorAll('svg > g')[0]);
    await fireEvent.pointerLeave(container.querySelectorAll('svg > g')[0]);

    expect(legend()).toBe('Jan Refund 120K Missed 0');
  });

  it('masks hovered values until revealed', async () => {
    const { container } = render(BarChart, props);
    await fireEvent.pointerEnter(container.querySelectorAll('svg > g')[0]);
    expect(legend()).toBe(`Jan Refund ${MASK} Missed ${MASK}`);
  });

  it('Given a masked readout When its token is clicked Then every amount reveals', async () => {
    const { container } = render(BarChart, props);
    await fireEvent.pointerEnter(container.querySelectorAll('svg > g')[1]);

    await fireEvent.click(screen.getAllByRole('button', { name: MASK_TOKEN_LABEL })[0]);

    expect(mask.revealed).toBe(true);
    expect(legend()).toBe('Feb Refund 80K Missed 15K');
  });

  it('Given masked axis ticks When the axis is clicked Then amounts reveal and the axis button goes away', async () => {
    const { container } = render(BarChart, props);
    const axis = screen.getByRole('button', { name: 'Hidden axis amounts, click to reveal' });
    expect(axis.tagName).toBe('BUTTON');
    expect(axis.closest('svg')).toBeNull();
    expect(container.querySelector('svg text')?.textContent).toBe('•••');

    await fireEvent.click(axis);

    expect(mask.revealed).toBe(true);
    expect(screen.queryByRole('button', { name: 'Hidden axis amounts, click to reveal' })).toBeNull();
    expect(container.querySelector('svg text')?.textContent).not.toBe('•••');
  });

  it('Given masked bars When they render Then their tooltips stay plain text', () => {
    const { container } = render(BarChart, props);
    expect(container.querySelector('rect title')?.textContent).toBe(`Refund: ${MASK}`);
  });
});
