import { fireEvent, render, screen } from '@testing-library/svelte';
import { afterEach, describe, expect, it } from 'vitest';
import { MASK_TOKEN_LABEL } from '$lib/components/maskTokens';
import { formatCompact, MASK } from '$lib/format';
import { mask } from '$lib/state/mask.svelte';
import MultiLineChart from './MultiLineChart.svelte';

afterEach(() => mask.hide());

const props = {
  dates: ['2026-01-01', '2026-02-01', '2026-03-01'],
  series: [{ name: 'Refund YTD', color: '#0f0', values: [100_000, 180_000, 262_000] }],
  format: formatCompact,
};

describe('MultiLineChart', () => {
  it('Given public prices When it renders Then nothing is masked', () => {
    render(MultiLineChart, props);
    expect(screen.getByText('262K')).toBeTruthy();
    expect(screen.queryByRole('button')).toBeNull();
  });

  it('Given private amounts When masked Then the readout and axis are tokens that reveal on click', async () => {
    const { container } = render(MultiLineChart, { ...props, masked: true });
    expect(screen.getByRole('button', { name: MASK_TOKEN_LABEL }).textContent).toBe(MASK);
    expect(container.querySelector('svg text')?.textContent).toBe('•••');

    await fireEvent.click(screen.getByRole('button', { name: 'Hidden axis amounts, click to reveal' }));

    expect(mask.revealed).toBe(true);
    expect(screen.getByText('262K')).toBeTruthy();
    expect(screen.queryByRole('button')).toBeNull();
  });
});
