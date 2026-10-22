import { describe, expect, it } from 'vitest';
import {
  arcPath,
  areaPath,
  dateTicks,
  extent,
  fromDay,
  linePath,
  linePoints,
  radarPoints,
  scale,
  slices,
  ticks,
  toDay,
} from './geometry';

describe('extent', () => {
  it('Given values When asked Then it spans them; an empty or flat set is padded', () => {
    expect(extent([2, 9])).toEqual([2, 9]);
    expect(extent([])).toEqual([0, 1]);
    expect(extent([5, 5])).toEqual([4.5, 5.5]);
  });

  it('Given bars When zero is included Then the extent reaches the baseline', () => {
    expect(extent([5, 10], { includeZero: true })).toEqual([0, 10]);
  });

  it('Given a pad When asked Then both ends move out by that share of the span', () => {
    expect(extent([0, 10], { pad: 0.1 })).toEqual([-1, 11]);
  });
});

describe('scale', () => {
  it('maps a domain onto a range, upside down for y', () => {
    const y = scale([0, 10], [100, 0]);
    expect(y(0)).toBe(100);
    expect(y(5)).toBe(50);
  });
});

describe('ticks', () => {
  it('Given a count When asked Then the steps are round and at most that many', () => {
    expect(ticks([0, 9_500_000], 4)).toEqual([0, 5_000_000]);
    expect(ticks([0, 100], 4)).toEqual([0, 50, 100]);
    expect(ticks([0, 100], 5)).toEqual([0, 20, 40, 60, 80, 100]);
    expect(ticks([3.4, 7.9], 5)).toEqual([4, 5, 6, 7]);
  });

  it('Given small steps When asked Then no float noise is left', () => {
    expect(ticks([0, 0.3], 3)).toEqual([0, 0.1, 0.2, 0.3]);
    expect(ticks([0, 0.9], 4)).toEqual([0, 0.5]);
  });
});

describe('days', () => {
  it('counts days from the calendar part of a date, and back', () => {
    expect(toDay('1970-01-02')).toBe(1);
    expect(toDay('2025-03-01') - toDay('2025-02-28')).toBe(1);
    expect(fromDay(toDay('2026-10-22T23:00:00+07:00'))).toBe('2026-10-22');
  });

  it('spreads date ticks across the span, first and last included', () => {
    const spread = dateTicks(toDay('2024-01-01'), toDay('2025-01-01'), 3);
    expect(spread).toHaveLength(3);
    expect(spread[0]).toBe(toDay('2024-01-01'));
    expect(spread[2]).toBe(toDay('2025-01-01'));
    expect(dateTicks(5, 5, 4)).toEqual([5]);
  });
});

describe('linePoints', () => {
  it('spans the width and maps max to the top', () => {
    expect(linePoints([0, 5, 10], 100, 50, 0)).toEqual([
      [0, 50],
      [50, 25],
      [100, 0],
    ]);
  });

  it('centres a flat series vertically', () => {
    const [[, y]] = linePoints([7, 7], 100, 50, 0);
    expect(y).toBe(25);
  });

  it('Given x positions When scaled Then uneven gaps stay uneven', () => {
    expect(linePoints([1, 2, 3], 100, 10, 0, [0, 1, 4]).map(([x]) => x)).toEqual([0, 25, 100]);
  });

  it('builds line and area paths, rounded to hundredths', () => {
    const points: [number, number][] = [
      [0, 10],
      [10, 0],
    ];
    expect(linePath(points)).toBe('M0,10 L10,0');
    expect(linePath([[1.23456, 2]])).toBe('M1.23,2');
    expect(areaPath(points, 20)).toBe('M0,10 L10,0 L10,20 L0,20 Z');
  });
});

describe('slices and arcs', () => {
  it('splits a circle by value and ignores negatives', () => {
    const result = slices([1, 3, -2]);
    expect(result.map((s) => s.fraction)).toEqual([0.25, 0.75, 0]);
    expect(result[1].end).toBeCloseTo(Math.PI * 2);
  });

  it('draws a closed annular sector', () => {
    const path = arcPath(50, 50, 40, 30, 0, Math.PI / 2);
    expect(path.startsWith('M50,10 A40,40 0 0 1 90,50')).toBe(true);
    expect(path.endsWith('Z')).toBe(true);
  });
});

describe('radarPoints', () => {
  it('places the first axis at 12 o clock and clamps', () => {
    expect(radarPoints([1, 2], 50, 50, 10)).toEqual([
      [50, 40],
      [50, 60],
    ]);
  });
});
