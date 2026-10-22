import { render, screen } from '@testing-library/svelte';
import { describe, expect, it } from 'vitest';
import Donut from './Donut.svelte';
import { PALETTE } from './palette';
import Radar from './Radar.svelte';
import Ring from './Ring.svelte';
import Sparkline from './Sparkline.svelte';

describe('Sparkline', () => {
  it('Given a label When drawn Then it is an image with that name', () => {
    render(Sparkline, { values: [1, 3, 2], label: 'Net worth trend' });
    expect(screen.getByRole('img', { name: 'Net worth trend' })).toBeTruthy();
  });

  it('Given no label When drawn Then it is decoration hidden from screen readers', () => {
    const { container } = render(Sparkline, { values: [1, 3] });
    expect(container.querySelector('[data-sparkline]')?.getAttribute('aria-hidden')).toBe('true');
  });

  it('Given an area and a dot When drawn Then the fill and the last point show in the colour', () => {
    const { container } = render(Sparkline, { values: [1, 3], area: true, dot: true, color: '#017c5d' });
    expect(container.querySelector('linearGradient stop')?.getAttribute('stop-color')).toBe('#017c5d');
    expect(container.querySelector('circle')?.getAttribute('fill')).toBe('#017c5d');
  });

  it('Given one value When drawn Then the empty text shows instead of a line', () => {
    const { container } = render(Sparkline, { values: [4], empty: 'No history' });
    expect(container.querySelector('svg')).toBeNull();
    expect(screen.getByText('No history')).toBeTruthy();
  });
});

describe('Donut', () => {
  it('Given items When drawn Then each slice above zero takes the next series colour and names its share', () => {
    const { container } = render(Donut, {
      items: [
        { label: 'Cash', value: 1 },
        { label: 'Gold', value: 3 },
        { label: 'None', value: 0 },
      ],
      label: 'Assets',
    });
    const paths = [...container.querySelectorAll('path')];
    expect(paths.map((path) => path.getAttribute('fill'))).toEqual([PALETTE[0], PALETTE[1]]);
    expect(paths[1].textContent).toBe('Gold: 75.0%');
    expect(screen.getByRole('img', { name: 'Assets' })).toBeTruthy();
  });
});

describe('Ring', () => {
  it('Given a percent When drawn Then the name carries it, rounded', () => {
    render(Ring, { percent: 42.4, color: PALETTE[0], label: 'Goal' });
    expect(screen.getByRole('img', { name: 'Goal: 42%' })).toBeTruthy();
  });
});

describe('Radar', () => {
  it('Given fewer than three axes When drawn Then nothing shows', () => {
    const { container } = render(Radar, { axes: [{ label: 'A', fraction: 1 }] });
    expect(container.querySelector('svg')).toBeNull();
  });

  it('Given three axes When drawn Then each axis is labelled', () => {
    render(Radar, {
      axes: [
        { label: 'Cash', fraction: 0.5 },
        { label: 'Gold', fraction: 1 },
        { label: 'Stocks', fraction: 0.2 },
      ],
    });
    expect(screen.getByRole('img', { name: 'Radar' }).textContent).toContain('Stocks');
  });
});
