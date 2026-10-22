import { fireEvent, render, screen } from '@testing-library/svelte';
import { describe, expect, it, vi } from 'vitest';
import type { Metric, TrendPoint } from '$lib/api/types';
import TrendChart from './TrendChart.svelte';

// jsdom has no ResizeObserver, which bind:clientWidth uses; the chart then keeps its 640px default.
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver;

const metric: Metric = {
  code: 'glucose',
  loinc: '',
  name: 'Glucose',
  nameVi: '',
  category: 'glucose',
  unit: 'mmol/L',
  aliases: [],
};
const points: TrendPoint[] = [
  {
    date: '2024-01-10',
    value: 5.2,
    printed: '5.2',
    printedUnit: 'mmol/L',
    refLow: 3.9,
    refHigh: 6.1,
    source: 'paper',
    documentId: 'd1',
    provider: 'Lab A',
  },
  { date: '2024-06-01', printed: 'Âm tính', source: 'paper', documentId: 'd2' },
  {
    date: '2025-01-10',
    value: 6.994,
    printed: '126',
    printedUnit: 'mg/dL',
    refLow: 3.885,
    refHigh: 6.106,
    flag: 'H',
    source: 'paper',
    documentId: 'd3',
    provider: 'Lab B',
  },
];

describe('TrendChart', () => {
  it('reads each numeric value with the keyboard and opens its paper on Enter', async () => {
    const onopen = vi.fn();
    render(TrendChart, { points, metric, onopen });
    const chart = screen.getByRole('group');
    expect(chart.getAttribute('aria-label')).toContain('2 values from 10/01/2024 to 10/01/2025');

    await fireEvent.keyDown(chart, { key: 'ArrowLeft' });
    const readout = screen.getByTestId('trend-readout');
    expect(readout.textContent).toContain('6.994');
    expect(readout.textContent).toContain('High');
    expect(readout.textContent).toContain('Lab B');
    expect(readout.textContent).toContain("Paper's range 3.885–6.106");
    expect(readout.textContent).toContain('Printed as 126 mg/dL');

    await fireEvent.keyDown(chart, { key: 'Enter' });
    expect(onopen).toHaveBeenCalledWith(expect.objectContaining({ documentId: 'd3' }));

    await fireEvent.keyDown(chart, { key: 'ArrowLeft' });
    expect(screen.getByTestId('trend-readout').textContent).toContain('Lab A');
    await fireEvent.keyDown(chart, { key: 'Escape' });
    expect(screen.queryByTestId('trend-readout')).toBeNull();
  });

  it('shades one band per point with a printed range', () => {
    const { container } = render(TrendChart, { points, metric });
    expect(container.querySelectorAll('rect')).toHaveLength(2);
  });

  it('says so when nothing is numeric', () => {
    render(TrendChart, { points: [points[1]], metric });
    expect(screen.getByText(/No numeric values to chart/)).toBeTruthy();
  });
});
