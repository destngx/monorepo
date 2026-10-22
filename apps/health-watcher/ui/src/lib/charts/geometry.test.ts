import { describe, expect, it } from 'vitest';
import { dateTicks, extent, linePath, scale, ticks, toDay } from './geometry';

describe('chart geometry', () => {
  it('pads a flat extent and a normal one by a share of the span', () => {
    expect(extent([5, 5])).toEqual([4.5, 5.5]);
    expect(extent([0, 10], 0.1)).toEqual([-1, 11]);
    expect(extent([])).toEqual([0, 1]);
  });

  it('maps a domain onto a range, upside down for y', () => {
    const y = scale([0, 10], [100, 0]);
    expect(y(0)).toBe(100);
    expect(y(5)).toBe(50);
  });

  it('gives round ticks inside the extent', () => {
    expect(ticks([3.4, 7.9])).toEqual([4, 5, 6, 7]);
    expect(ticks([0, 100], 4)).toEqual([0, 20, 40, 60, 80, 100]);
  });

  it('counts days from the calendar part of a date', () => {
    expect(toDay('1970-01-02')).toBe(1);
    expect(toDay('2025-03-01') - toDay('2025-02-28')).toBe(1);
  });

  it('spreads date ticks across the span, first and last included', () => {
    const t = dateTicks(toDay('2024-01-01'), toDay('2025-01-01'), 3);
    expect(t).toHaveLength(3);
    expect(t[0]).toBe(toDay('2024-01-01'));
    expect(t[2]).toBe(toDay('2025-01-01'));
    expect(dateTicks(5, 5, 4)).toEqual([5]);
  });

  it('draws a path through points', () => {
    expect(
      linePath([
        [0, 1],
        [2, 3],
      ]),
    ).toBe('M0,1 L2,3');
  });
});
