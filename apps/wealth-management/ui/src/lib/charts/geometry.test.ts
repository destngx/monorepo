import { describe, expect, it } from 'vitest';
import { arcPath, areaPath, extent, linePath, linePoints, radarPoints, slices, ticks } from './geometry';

describe('linePoints', () => {
  it('spans the width and maps max to the top', () => {
    const points = linePoints([0, 5, 10], 100, 50, 0);
    expect(points).toEqual([
      [0, 50],
      [50, 25],
      [100, 0],
    ]);
  });

  it('centres a flat series vertically', () => {
    const [[, y]] = linePoints([7, 7], 100, 50, 0);
    expect(y).toBe(25);
  });

  it('builds line and area paths', () => {
    const points: [number, number][] = [
      [0, 10],
      [10, 0],
    ];
    expect(linePath(points)).toBe('M0,10 L10,0');
    expect(areaPath(points, 20)).toBe('M0,10 L10,0 L10,20 L0,20 Z');
  });
});

describe('extent', () => {
  it('can include zero for bar charts', () => {
    expect(extent([5, 10], true)).toEqual([0, 10]);
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

describe('ticks', () => {
  it('chooses round steps', () => {
    expect(ticks([0, 9_500_000], 4)).toEqual([0, 5_000_000]);
    expect(ticks([0, 100], 4)).toEqual([0, 50, 100]);
  });
});
